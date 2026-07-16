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
	DateString   string  `json:"date_string"` // フロントから届く日時 (例: "2026-07-12 15:00:00")
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

		var req RecommendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 1. analyzer.go の関数を呼び出して自由入力を形態素解析
		multToilet, multRental, multSafety, multAccess := ParseUserText(req.UserText)

		wToilet := req.WeightToilet * multToilet
		wRental := req.WeightRental * multRental
		wSafety := 1.0 * multSafety
		wAccess := 1.0 * multAccess

		if req.Experience == "beginner" {
			wToilet *= 1.5
			wRental *= 1.5
			wSafety *= 2.0
			wAccess *= 1.2
		} else {
			wToilet *= 0.6
			wRental *= 0.5
		}

		// 2. DBから該当レジャーのスポットを抽出
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

			// 🔥 3. weather.go で定義した外部API連携関数を直接呼び出す！
			wind, rain, err := FetchWeatherForecast(s.Lat, s.Lng, req.DateString)
			if err != nil {
				log.Printf("天気取得失敗 (%s): %v", s.SpotName, err)
				wind, rain = 0.0, 0.0 // エラー時は安全側に倒して続行
			}

			// 動的重み付き線形結合の計算
			baseScore := (s.ScoreToilet * wToilet) +
				(s.ScoreRental * wRental) +
				(s.ScoreSafety * wSafety) +
				(s.ScoreAccess * wAccess)

			// セーフティネット判定にAPIから引いた風速・雨量をセット
			safetyFactor := getWeatherSafetyFactor(s.LeisureType, wind, rain)
			s.FinalScore = math.Round((baseScore*safetyFactor)*100) / 100

			spots = append(spots, s)
		}

		// 4. スコア順にソート
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