package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"

	_ "github.com/mattn/go-sqlite3"
)

// DBから取得するスポットの構造体
type Spot struct {
	ID          int     `json:"id"`
	SpotName    string  `json:"spot_name"`
	LeisureType string  `json:"leisure_type"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	ScoreToilet float64 `json:"score_toilet"`
	ScoreRental float64 `json:"score_rental"`
	ScoreSafety float64 `json:"score_safety"`
	ScoreAccess float64 `json:"score_access"`
	FinalScore  float64 `json:"final_score"`
}

// フロントエンドから届くリクエストの構造体
type RecommendRequest struct {
	LeisureType  string  `json:"leisure_type"`
	Experience   string  `json:"experience"`
	WeightToilet float64 `json:"weight_toilet"`
	WeightRental float64 `json:"weight_rental"`
	UserText     string  `json:"user_text"`
	DateString   string  `json:"date_string"` // 🔥 これを追加！
}
var leisureBaseEase = map[string]float64{
	"shopping": 1.0,
	"fishing":  0.6,
	"hiking":   0.5,
	"camp":     0.3,
}

// レジャーごとの気象セーフティネット判定
func getWeatherSafetyFactor(leisureType string, wind, rain float64) float64 {
	switch leisureType {
	case "fishing":
		if wind >= 8.0 || rain >= 5.0 { return 0.0 }
		if wind >= 4.0 || rain >= 1.0 { return 0.5 }
	case "camp":
		if wind >= 7.0 || rain >= 10.0 { return 0.0 }
		if wind >= 4.0 || rain >= 2.0 { return 0.5 }
	case "hiking":
		if wind >= 10.0 || rain >= 8.0 { return 0.0 }
		if wind >= 5.0 || rain >= 3.0 { return 0.5 }
	}
	return 1.0
}
func recommendHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 🔥 1. CORSヘッダーの設定（すべてのオリジンからのアクセスを許可）
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// 🔥 2. プリフライトリクエスト (OPTIONS) の場合はここで 200 OK を返して即終了する
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 3. POSTメソッド以外の拒否チェック（OPTIONSを処理した後に判定する）
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RecommendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 1. 自由入力テキストの解析（analyzer.go）
		multToilet, multRental, multSafety, multAccess := ParseUserText(req.UserText)

		wToilet := req.WeightToilet * multToilet
		wRental := req.WeightRental * multRental
		wSafety := 1.0 * multSafety
		wAccess := 1.0 * multAccess

		if req.Experience == "beginner" {
			wToilet *= 1.5
			wRental *= 1.5
			wSafety *= 2.0
			wSafety *= 2.0
			wAccess *= 1.2
		} else {
			wToilet *= 0.6
			wRental *= 0.5
		}

		// 🔥 【配置場所①】SQLクエリの分岐（ジャンル指定なし・ありの切り替え）
		var query string
		var args []interface{}

		if req.LeisureType == "" || req.LeisureType == "any" {
			// レジャー未選択時：全スポットを抽出
			query = "SELECT id, spot_name, leisure_type, lat, lng, score_toilet, score_rental, score_safety, score_access FROM spots"
		} else {
			// レジャー指定時：そのジャンルのみ抽出
			query = "SELECT id, spot_name, leisure_type, lat, lng, score_toilet, score_rental, score_safety, score_access FROM spots WHERE leisure_type = ?"
			args = append(args, req.LeisureType)
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var spots []Spot
		for rows.Next() {
			var s Spot
			err := rows.Scan(&s.ID, &s.SpotName, &s.LeisureType, &s.Lat, &s.Lng, &s.ScoreToilet, &s.ScoreRental, &s.ScoreSafety, &s.ScoreAccess)
			if err != nil {
				log.Println(err)
				continue
			}

			// 2. 天気予報の取得（weather.go）
			wind, rain, err := FetchWeatherForecast(s.Lat, s.Lng, req.DateString)
			if err != nil {
				log.Printf("天気取得失敗 (%s): %v", s.SpotName, err)
				wind, rain = 0.0, 0.0
			}

			// 3. スポット個別の素点＋重み付けスコア（ベーススコア）
			baseScore := (s.ScoreToilet * wToilet) +
				(s.ScoreRental * wRental) +
				(s.ScoreSafety * wSafety) +
				(s.ScoreAccess * wAccess)

			// 4. 天候セーフティネット判定
			safetyFactor := getWeatherSafetyFactor(s.LeisureType, wind, rain)

			// 🔥 【配置場所②】レジャー自体の適合度（手軽さ）補正の計算
			genreEase := leisureBaseEase[s.LeisureType]
			if genreEase == 0 {
				genreEase = 0.5 // 定義されていないジャンルの初期値
			}

			genreScore := genreEase
			if req.Experience == "beginner" {
				genreScore *= 1.5 // 初心者の場合は「手軽なレジャー」を強力ブースト
			}

			// 5. 最終スコアの算出（スポット評価 × 天候リスク × レジャー手軽さ）
			s.FinalScore = math.Round((baseScore * safetyFactor * genreScore) * 100) / 100

			spots = append(spots, s)
		}

		// 6. スコア順にソートしてレスポンス返却
		sort.Slice(spots, func(i, j int) bool {
			return spots[i].FinalScore > spots[j].FinalScore
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spots)
	}
}
func main() {
	db, err := sql.Open("sqlite3", "../db/data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/api/recommend", recommendHandler(db))
	fmt.Println("🚀 Go API Server running on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}