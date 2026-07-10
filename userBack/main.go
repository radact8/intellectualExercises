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
	LeisureType  string  `json:"leisure_type"`
	Experience   string  `json:"experience"`
	WeightToilet float64 `json:"weight_toilet"` // スライダー等の基本値（なければ1.0）
	WeightRental float64 `json:"weight_rental"`
	CurrentWind  float64 `json:"current_wind"`
	CurrentRain  float64 `json:"current_rain"`
	UserText     string  `json:"user_text"` // ユーザーの自由入力テキスト
}

// レジャーごとの気象セーフティネット判定
func getWeatherSafetyFactor(leisureType string, wind, rain float64) float64 {
	switch leisureType {
	case "fishing":
		if wind >= 8.0 || rain >= 5.0 {
			return 0.0
		}
		if wind >= 4.0 || rain >= 1.0 {
			return 0.5
		}
	case "camp":
		if wind >= 7.0 || rain >= 10.0 {
			return 0.0
		}
		if wind >= 4.0 || rain >= 2.0 {
			return 0.5
		}
	case "hiking":
		if wind >= 10.0 || rain >= 8.0 {
			return 0.0
		}
		if wind >= 5.0 || rain >= 3.0 {
			return 0.5
		}
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

		// 🔥 2. 別ファイル（analyzer.go）の関数を呼び出して自由入力を形態素解析
		multToilet, multRental, multSafety, multAccess := ParseUserText(req.UserText)

		// 3. 基本の重みにテキストからの補正倍率を掛け合わせる
		wToilet := req.WeightToilet * multToilet
		wRental := req.WeightRental * multRental
		wSafety := 1.0 * multSafety
		wAccess := 1.0 * multAccess

		// 4. ユーザーの習熟度（初心者かどうか）による自動補正ロジックを適用
		if req.Experience == "beginner" {
			wToilet *= 1.5
			wRental *= 1.5
			wSafety *= 2.0 // 初心者は安全性を自動で最重視
			wAccess *= 1.2
		} else {
			wToilet *= 0.6
			wRental *= 0.5
		}

		// 5. SQLで該当するレジャー種別のスポットをDBから抽出
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

			// 6. 【知能処理：動的重み付き線形結合】の計算
			baseScore := (s.ScoreToilet * wToilet) +
				(s.ScoreRental * wRental) +
				(s.ScoreSafety * wSafety) +
				(s.ScoreAccess * wAccess)

			// 7. 【セーフティネット判定】天候による安全係数を計算
			safetyFactor := getWeatherSafetyFactor(s.LeisureType, req.CurrentWind, req.CurrentRain)

			// 最終スコアの算出（危険なら 0 になる）
			s.FinalScore = math.Round((baseScore*safetyFactor)*100) / 100

			spots = append(spots, s)
		}

		// 8. 計算された FinalScore の高い順にソート（並び替え）
		sort.Slice(spots, func(i, j int) bool {
			return spots[i].FinalScore > spots[j].FinalScore
		})

		// 9. 結果をJSONとしてフロントに返却
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(spots)
	}
}

func main() {
	// データベースファイル（data.db）に接続
	db, err := sql.Open("sqlite3", "../db/data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/api/recommend", recommendHandler(db))
	fmt.Println("🚀 Go API Server running on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}