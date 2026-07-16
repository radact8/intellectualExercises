// weather.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// OpenWeatherMapのレスポンスをパースするための構造体
type OWMForecastResponse struct {
	List []struct {
		DtTxt string `json:"dt_txt"` // 予報日時（UTC表記 "2026-07-12 06:00:00"）
		Main  struct {
			Temp float64 `json:"temp"`
		} `json:"main"`
		Wind struct {
			Speed float64 `json:"speed"` // 風速 (m/s)
		} `json:"wind"`
		Rain struct {
			ThreeH float64 `json:"3h"` // 3時間あたりの雨量 (mm)
		} `json:"rain"`
	} `json:"list"`
}

// FetchWeatherForecast は指定された緯度・経度と日時に最も近い予報データ（風速、雨量）をOWMから取得します。
func FetchWeatherForecast(lat, lng float64, targetDateStr string) (float64, float64, error) {
	var windSpeed, rainVolume float64 = 0.0, 0.0

	if targetDateStr == "" {
		return windSpeed, rainVolume, nil
	}

	// 🔥 1. 同階層の .env ファイルを読み込む
	// 読み込みに失敗しても、すでにシステム環境変数にセットされている場合があるため
	// ログ出力に留めるか、必要に応じてエラーハンドリングします。
	if err := godotenv.Load(); err != nil {
		// 開発環境で .env が見つからない場合は警告を出す
		fmt.Println("⚠️ .env file not found (Using system environment variables instead)")
	}

	// 🔥 2. 環境変数から API キーを取得する
	apiKey := os.Getenv("OPENWEATHERMAP_API_KEY")
	if apiKey == "" {
		return windSpeed, rainVolume, fmt.Errorf("APIキーが環境変数 'OPENWEATHERMAP_API_KEY' に設定されていません")
	}

	// JST（日本標準時）のロケーションを設定
	loc, _ := time.LoadLocation("Asia/Tokyo")
	targetTime, err := time.ParseInLocation("2006-01-02 15:00:00", targetDateStr, loc)
	if err != nil {
		return windSpeed, rainVolume, fmt.Errorf("日付パースエラー: %v", err)
	}

	// 3. OpenWeatherMapの 5日間/3時間予報 APIのURLを構築 (取得した apiKey を使用)
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?lat=%f&lon=%f&appid=%s&units=metric", lat, lng, apiKey)

	// 4. HTTP GET リクエストの送信
	resp, err := http.Get(url)
	if err != nil {
		return windSpeed, rainVolume, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return windSpeed, rainVolume, fmt.Errorf("APIエラー: status %d", resp.StatusCode)
	}

	// 5. JSONのデコード
	var forecastData OWMForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecastData); err != nil {
		return windSpeed, rainVolume, err
	}

	// 6. 予報リストの中から、ユーザーが指定した日時に「一番近い」予報を探索
	var minDiff int64 = 9999999999
	for _, f := range forecastData.List {
		layout := "2006-01-02 15:00:00"
		fTime, _ := time.ParseInLocation(layout, f.DtTxt, time.UTC)
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