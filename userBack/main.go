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
	FinalScore  float64 `json:"final_score"` // Go側で計算して格納する
}

// フロントエンド（ユーザー入力）から届くリクエストの構造体
type RecommendRequest struct {
	LeisureType  string  `json:"leisure_type"`  // 例: "camp"
	Experience   string  `json:"experience"`    // 例: "beginner"
	WeightToilet float64 `json:"weight_toilet"` // 0.0 ~ 2.0 などのスライダー値
	WeightRental float64 `json:"weight_rental"` // 0.0 ~ 2.0
	CurrentWind  float64 `json:"current_wind"`  // 気象API等から取得した現在の風速
	CurrentRain  float64 `json:"current_rain"`  // 現在の雨量
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
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 1. フロントからのユーザー入力（JSON）をデコード
		var req RecommendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 2. ユーザーのガチ度（習熟度）に応じた「重み（係数）」の自動補正ロジック
		wToilet := req.WeightToilet
		wRental := req.WeightRental
		var wSafety, wAccess float64 = 1.0, 1.0

		if req.Experience == "beginner" {
			wToilet *= 1.5
			wRental *= 1.5
			wSafety = 2.0 // 初心者は安全性を自動で最重視
			wAccess = 1.2
		} else {
			wToilet *= 0.6
			wRental *= 0.5
		}

		// 3. SQLで該当するレジャー種別のスポットをDBから抽出
		rows, err := db.Query("SELECT id, spot_name, leisure_type, lat, lng, score_toilet, score_rental, score_safety, score_access FROM spots WHERE leisure_type = ?", req.LeisureType)
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

			// 4. 【知能処理：動的重み付き線形結合】の計算
			baseScore := (s.ScoreToilet * wToilet) +
				(s.ScoreRental * wRental) +
				(s.ScoreSafety * wSafety) +
				(s.ScoreAccess * wAccess)

			// 5. 【セーフティネット判定】
			safetyFactor := getWeatherSafetyFactor(s.LeisureType, req.CurrentWind, req.CurrentRain)
			
			// 最終スコアの算出（危険なら 0 になる）
			s.FinalScore = math.Round((baseScore*safetyFactor)*100) / 100

			spots = append(spots, s)
		}

		// 6. 計算された FinalScore の高い順にソート（Sliceの並び替え）
		sort.Slice(spots, func(i, j int) bool {
			return spots[i].FinalScore > spots[j].FinalScore
		})

		// 7. 結果をJSONとしてフロントに返却
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spots)
	}
}

func main() {
	fmt.Println("check")
	// SQLiteデータベースファイルに接続
	db, err := sql.Open("sqlite3", "../db/data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/api/recommend", recommendHandler(db))
	fmt.Println("🚀 Go API Server running on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}