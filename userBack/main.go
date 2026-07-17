package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Spot struct {
	ID             int     `json:"id"`
	SpotName       string  `json:"spot_name"`
	LeisureType    string  `json:"leisure_type"`
	Lat            float64 `json:"lat"`
	Lng            float64 `json:"lng"`
	ScoreToilet    float64 `json:"score_toilet"`
	ScoreRental    float64 `json:"score_rental"`
	ScoreSafety    float64 `json:"score_safety"`
	ScoreAccess    float64 `json:"score_access"`
	FinalScore     float64 `json:"final_score"`
	Address        string  `json:"address"`
	URL            string  `json:"url"`
	ImageURL       string  `json:"image_url"`
	Description    string  `json:"description"`
	WindSpeed      float64 `json:"wind_speed"`
	RainVolume     float64 `json:"rain_volume"`
	WeatherWarning string  `json:"weather_warning"`
}

type RecommendRequest struct {
	LeisureType  string  `json:"leisure_type"`
	Experience   string  `json:"experience"`
	WeightToilet float64 `json:"weight_toilet"`
	WeightRental float64 `json:"weight_rental"`
	UserText     string  `json:"user_text"`
	DateString   string  `json:"date_string"`
}

var leisureBaseEase = map[string]float64{
	"shopping": 1.0,
	"fishing":  0.6,
	"hiking":   0.5,
	"camp":     0.3,
}

// 2. 天候による減点判定 ＆ ユーザー向け詳細メッセージ返却
func getWeatherSafetyFactor(leisureType string, wind, rain float64) (float64, string) {
	switch leisureType {
	case "fishing":
		if wind >= 8.0 || rain >= 5.0 {
			return 0.0, fmt.Sprintf("⛔ 危険（風速%.1fm/s, 雨量%.1fmm/h）: 高波や落雷の危険があるため非推奨です", wind, rain)
		}
		if wind >= 4.0 || rain >= 1.0 {
			return 0.5, fmt.Sprintf("⚠️ 注意（風速%.1fm/s, 雨量%.1fmm/h）: やや風・雨が強くライントラブルのリスクがあります", wind, rain)
		}
	case "camp":
		if wind >= 7.0 || rain >= 10.0 {
			return 0.0, fmt.Sprintf("⛔ 危険（風速%.1fm/s, 雨量%.1fmm/h）: 狂風・大雨によりテント設営不能・浸水リスクがあります", wind, rain)
		}
		if wind >= 4.0 || rain >= 2.0 {
			return 0.5, fmt.Sprintf("⚠️ 注意（風速%.1fm/s, 雨量%.1fmm/h）: ペグ打ちの補強やタープの耐風対策が必要です", wind, rain)
		}
	case "hiking":
		if wind >= 10.0 || rain >= 8.0 {
			return 0.0, fmt.Sprintf("⛔ 危険（風速%.1fm/s, 雨量%.1fmm/h）: 滑落・低体温症・ぬかるみの危険があるため登山中止を強く推奨します", wind, rain)
		}
		if wind >= 5.0 || rain >= 3.0 {
			return 0.5, fmt.Sprintf("⚠️ 注意（風速%.1fm/s, 雨量%.1fmm/h）: 足元が滑りやすくなっているためレインウェアと十分な装備が必要です", wind, rain)
		}
	}
	return 1.0, "☀️ コンディション良好: 安全に楽しめる天候予報です"
}

func recommendHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req RecommendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// 1. テキスト解析（重視軸、レジャー意図、エリア抽出）
		multToilet, multRental, multSafety, multAccess := ParseUserText(req.UserText)
		leisureSentiments := ParseLeisureSentiment(req.UserText, req.LeisureType)
		targetAreas := ParseAreaKeyword(req.UserText) // 👈 1. エリア抽出

		detectedLeisure := ""
		if req.LeisureType == "" || req.LeisureType == "any" {
			for leisure, mult := range leisureSentiments {
				if mult > 1.0 {
					detectedLeisure = leisure
					break
				}
			}
		}

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

		targetLeisure := req.LeisureType
		if (targetLeisure == "" || targetLeisure == "any") && detectedLeisure != "" {
			targetLeisure = detectedLeisure
		}

		var query string
		var args []interface{}

		if targetLeisure == "" || targetLeisure == "any" {
			query = "SELECT id, spot_name, leisure_type, lat, lng, score_toilet, score_rental, score_safety, score_access, address, url, image_url, description FROM spots"
		} else {
			query = "SELECT id, spot_name, leisure_type, lat, lng, score_toilet, score_rental, score_safety, score_access, address, url, image_url, description FROM spots WHERE leisure_type = ?"
			args = append(args, targetLeisure)
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
			err := rows.Scan(
				&s.ID, &s.SpotName, &s.LeisureType, &s.Lat, &s.Lng,
				&s.ScoreToilet, &s.ScoreRental, &s.ScoreSafety, &s.ScoreAccess,
				&s.Address, &s.URL, &s.ImageURL, &s.Description,
			)
			if err != nil {
				log.Println(err)
				continue
			}

			// 1. エリア（住所）フィルタリング判定
			if len(targetAreas) > 0 {
				matched := false
				for _, area := range targetAreas {
					if strings.Contains(s.Address, area) {
						matched = true
						break
					}
				}
				if !matched {
					continue // エエリア条件に合致しないスポットはスキップ
				}
			}

			// 天気予報取得
			wind, rain, err := FetchWeatherForecast(s.Lat, s.Lng, req.DateString)
			if err != nil {
				http.Error(w, fmt.Sprintf("リクエストエラー: %v", err), http.StatusBadRequest)
				return
			}

			s.WindSpeed = math.Round(wind*10) / 10
			s.RainVolume = math.Round(rain*10) / 10

			// 2. 天候による安全度倍率 ＆ メッセージ設定
			safetyFactor, warningMsg := getWeatherSafetyFactor(s.LeisureType, wind, rain)
			s.WeatherWarning = warningMsg

			baseScore := (s.ScoreToilet * wToilet) +
				(s.ScoreRental * wRental) +
				(s.ScoreSafety * wSafety) +
				(s.ScoreAccess * wAccess)

			genreEase := leisureBaseEase[s.LeisureType]
			if genreEase == 0 {
				genreEase = 0.5
			}

			genreScore := genreEase
			if req.Experience == "beginner" {
				genreScore *= 1.5
			}

			if sentimentMult, exists := leisureSentiments[s.LeisureType]; exists {
				genreScore *= sentimentMult
			}

			s.FinalScore = math.Round((baseScore * safetyFactor * genreScore) * 100) / 100

			spots = append(spots, s)
		}

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