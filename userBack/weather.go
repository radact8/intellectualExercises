package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type OWMForecastResponse struct {
	List []struct {
		DtTxt string `json:"dt_txt"`
		Main  struct {
			Temp float64 `json:"temp"`
		} `json:"main"`
		Wind struct {
			Speed float64 `json:"speed"`
		} `json:"wind"`
		Rain struct {
			ThreeH float64 `json:"3h"`
		} `json:"rain"`
	} `json:"list"`
}

func FetchWeatherForecast(lat, lng float64, targetDateStr string) (float64, float64, error) {
	var windSpeed, rainVolume float64 = 0.0, 0.0

	// 1. 未入力判定（エラーを返す）
	if targetDateStr == "" {
		return windSpeed, rainVolume, fmt.Errorf("予定日時（date_string）が入力されていません")
		return windSpeed, rainVolume, fmt.Errorf("予定日時（date_string）が入力されていません")
	}

	// 2. JSTでパース（フォーマットは 15:04:05）
	loc, _ := time.LoadLocation("Asia/Tokyo")
	targetTime, err := time.ParseInLocation("2006-01-02 15:04:05", targetDateStr, loc)
	if err != nil {
		return windSpeed, rainVolume, fmt.Errorf("日時のフォーマットが不正です (正しい形式例: 'YYYY-MM-DD HH:mm:00'): %v", err)
	}

	// 3. .env 読み込み
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️ .env file not found (Using system environment variables instead)")
	}

	apiKey := os.Getenv("OPENWEATHERMAP_API_KEY")
	if apiKey == "" {
		return windSpeed, rainVolume, fmt.Errorf("APIキーが環境変数 'OPENWEATHERMAP_API_KEY' に設定されていません")
	}

	// 4. APIリクエスト
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?lat=%f&lon=%f&appid=%s&units=metric", lat, lng, apiKey)
	resp, err := http.Get(url)
	if err != nil {
		return windSpeed, rainVolume, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return windSpeed, rainVolume, fmt.Errorf("APIエラー: status %d", resp.StatusCode)
	}

	var forecastData OWMForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecastData); err != nil {
		return windSpeed, rainVolume, err
	}

	// 5. 直近予報の探索
	var minDiff int64 = 9999999999
	for _, f := range forecastData.List {
		fTime, err := time.ParseInLocation("2006-01-02 15:04:05", f.DtTxt, time.UTC)
		if err != nil {
			continue
		}
		fTimeJST := fTime.In(loc)

		diff := int64(targetTime.Sub(fTimeJST).Seconds())
		if diff < 0 {
			diff = -diff
		}

		if diff < minDiff {
			minDiff = diff
			windSpeed = f.Wind.Speed
			rainVolume = f.Rain.ThreeH / 3.0
		}
	}

	return windSpeed, rainVolume, nil
}