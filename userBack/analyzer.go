package main

import (
	"strings"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

var leisureKeywords = map[string][]string{
	"fishing":  {"釣り", "フィッシング", "海釣り", "川釣り"},
	"camp":     {"キャンプ", "オートキャンプ", "グランピング", "テント"},
	"hiking":   {"ハイキング", "登山", "トレッキング", "山登り", "山"},
	"shopping": {"ショッピング", "買い物", "アウトレット", "モール"},
}

var positiveWords = []string{"行きたい", "たい", "好き", "良い", "いい", "おすすめ", "やりたい"}
var negativeWords = []string{"嫌", "やりたくない", "以外", "たくない", "避ける", "無理", "ダメ", "やめて", "なし"}

// 🌏 47都道府県 ＆ 主要地方マッピング辞書
var areaMap = map[string][]string{
	// --- 地方区分 ---
	"北海道地方": {"北海道"},
	"東北":     {"青森県", "岩手県", "宮城県", "秋田県", "山形県", "福島県"},
	"関東":     {"東京都", "神奈川県", "埼玉県", "千葉県", "茨城県", "栃木県", "群馬県"},
	"中部":     {"新潟県", "富山県", "石川県", "福井県", "山梨県", "長野県", "岐阜県", "静岡県", "愛知県"},
	"北陸":     {"富山県", "石川県", "福井県", "新潟県"},
	"東海":     {"愛知県", "岐阜県", "三重県", "静岡県"},
	"関西":     {"大阪府", "京都府", "兵庫県", "奈良県", "滋賀県", "和歌山県"},
	"近畿":     {"大阪府", "京都府", "兵庫県", "奈良県", "滋賀県", "和歌山県"},
	"中国":     {"鳥取県", "島根県", "岡山県", "広島県", "山口県"},
	"四国":     {"徳島県", "香川県", "愛媛県", "高知県"},
	"九州":     {"福岡県", "佐賀県", "長崎県", "熊本県", "大分県", "宮崎県", "鹿児島県"},
	"沖縄地方":   {"沖縄県"},

	// --- 47都道府県（通称・略称含む） ---
	"北海道": {"北海道"},
	"青森":  {"青森県"}, "青森県": {"青森県"},
	"岩手":  {"岩手県"}, "岩手県": {"岩手県"},
	"宮城":  {"宮城県"}, "宮城県": {"宮城県"},
	"秋田":  {"秋田県"}, "秋田県": {"秋田県"},
	"山形":  {"山形県"}, "山形県": {"山形県"},
	"福島":  {"福島県"}, "福島県": {"福島県"},
	"茨城":  {"茨城県"}, "茨城県": {"茨城県"},
	"栃木":  {"栃木県"}, "栃木県": {"栃木県"},
	"群馬":  {"群馬県"}, "群馬県": {"群馬県"},
	"埼玉":  {"埼玉県"}, "埼玉県": {"埼玉県"},
	"千葉":  {"千葉県"}, "千葉県": {"千葉県"},
	"東京":  {"東京都"}, "東京都": {"東京都"},
	"神奈川": {"神奈川県"}, "神奈川県": {"神奈川県"},
	"新潟":  {"新潟県"}, "新潟県": {"新潟県"},
	"富山":  {"富山県"}, "富山県": {"富山県"},
	"石川":  {"石川県"}, "石川県": {"石川県"},
	"福井":  {"福井県"}, "福井県": {"福井県"},
	"山梨":  {"山梨県"}, "山梨県": {"山梨県"},
	"長野":  {"長野県"}, "長野県": {"長野県"},
	"岐阜":  {"岐阜県"}, "岐阜県": {"岐阜県"},
	"静岡":  {"静岡県"}, "静岡県": {"静岡県"},
	"愛知":  {"愛知県"}, "愛知県": {"愛知県"},
	"三重":  {"三重県"}, "三重県": {"三重県"},
	"滋賀":  {"滋賀県"}, "滋賀県": {"滋賀県"},
	"京都":  {"京都府"}, "京都府": {"京都府"},
	"大阪":  {"大阪府"}, "大阪府": {"大阪府"},
	"兵庫":  {"兵庫県"}, "兵庫県": {"兵庫県"},
	"奈良":  {"奈良県"}, "奈良県": {"奈良県"},
	"和歌山": {"和歌山県"}, "和歌山県": {"和歌山県"},
	"鳥取":  {"鳥取県"}, "鳥取県": {"鳥取県"},
	"島根":  {"島根県"}, "島根県": {"島根県"},
	"岡山":  {"岡山県"}, "岡山県": {"岡山県"},
	"広島":  {"広島県"}, "広島県": {"広島県"},
	"山口":  {"山口県"}, "山口県": {"山口県"},
	"徳島":  {"徳島県"}, "徳島県": {"徳島県"},
	"香川":  {"香川県"}, "香川県": {"香川県"},
	"愛媛":  {"愛媛県"}, "愛媛県": {"愛媛県"},
	"高知":  {"高知県"}, "高知県": {"高知県"},
	"福岡":  {"福岡県"}, "福岡県": {"福岡県"},
	"佐賀":  {"佐賀県"}, "佐賀県": {"佐賀県"},
	"長崎":  {"長崎県"}, "長崎県": {"長崎県"},
	"熊本":  {"熊本県"}, "熊本県": {"熊本県"},
	"大分":  {"大分県"}, "大分県": {"大分県"},
	"宮崎":  {"宮崎県"}, "宮崎県": {"宮崎県"},
	"鹿児島": {"鹿児島県"}, "鹿児島県": {"鹿児島県"},
	"沖縄":  {"沖縄県"}, "沖縄県": {"沖縄県"},
}

// ParseAreaKeyword は自由記述から地域・都道府県キーワードを抽出します
func ParseAreaKeyword(text string) []string {
	if text == "" {
		return nil
	}

	var matchedPrefectures []string

	// 1. 地域・都道府県辞書とのマッチング
	for key, prefList := range areaMap {
		if strings.Contains(text, key) {
			matchedPrefectures = append(matchedPrefectures, prefList...)
		}
	}

	// 2. 辞書になかった市町村名等の場合、入力テキスト自体を部分一致判定に使用
	if len(matchedPrefectures) == 0 {
		matchedPrefectures = append(matchedPrefectures, text)
	}

	return matchedPrefectures
}

func ParseUserText(userInput string) (float64, float64, float64, float64) {
	mToilet, mRental, mSafety, mAccess := 1.0, 1.0, 1.0, 1.0

	if userInput == "" {
		return mToilet, mRental, mSafety, mAccess
	}

	t, _ := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	tokens := t.Tokenize(userInput)

	type wordInfo struct {
		surface string
		base    string
		pos     string
	}
	var words []wordInfo

	for _, token := range tokens {
		if token.Surface != "" {
			features := token.Features()
			baseForm := token.Surface
			pos := ""
			if len(features) > 0 {
				pos = features[0]
			}
			if len(features) > 6 && features[6] != "*" {
				baseForm = features[6]
			}
			words = append(words, wordInfo{surface: token.Surface, base: baseForm, pos: pos})
		}
	}

	checkContext := func(idx int) (isStrong, isWeak bool) {
		strongWords := map[string]bool{"絶対": true, "必須": true, "重視": true, "一番": true, "超": true, "とても": true}
		weakWords := map[string]bool{"気にしない": true, "適当": true, "無し": true, "ダメ": true, "ない": true, "なかった": true, "不要": true}

		start := idx - 2
		if start < 0 {
			start = 0
		}
		end := idx + 3
		if end > len(words) {
			end = len(words)
		}

		for i := start; i < end; i++ {
			w := words[i].base
			if strongWords[w] {
				isStrong = true
			}
			if weakWords[w] || (words[i].pos == "動詞-非自立" && w == "ある") {
				isWeak = true
			}
		}
		return isStrong, isWeak
	}

	for i, w := range words {
		if w.base == "トイレ" || w.base == "綺麗" || w.base == "清潔" || w.base == "水回り" || strings.Contains(w.surface, "綺麗") {
			isStrong, isWeak := checkContext(i)
			if isStrong {
				mToilet = 2.0
			} else if isWeak {
				mToilet = 0.3
			} else {
				mToilet = 1.5
			}
		}
		if w.base == "手ぶら" || w.base == "レンタル" || w.base == "道具" || w.base == "手軽" {
			isStrong, isWeak := checkContext(i)
			if isStrong {
				mRental = 2.0
			} else if isWeak {
				mRental = 0.3
			} else {
				mRental = 1.5
			}
		}
		if w.base == "安全" || w.base == "安心" || w.base == "子供" || w.base == "ファミリー" || w.base == "柵" {
			isStrong, isWeak := checkContext(i)
			if isStrong {
				mSafety = 2.0
			} else if isWeak {
				mSafety = 0.5
			} else {
				mSafety = 1.5
			}
		}
		if strings.Contains(w.base, "アクセス") || w.base == "運転" || w.base == "道" || w.base == "狭い" || w.base == "遠い" || w.base == "駅" {
			isStrong, isWeak := checkContext(i)
			if isStrong || w.base == "狭い" || w.base == "遠い" {
				mAccess = 2.0
			} else if isWeak {
				mAccess = 0.5
			} else {
				mAccess = 1.5
			}
		}
	}

	return mToilet, mRental, mSafety, mAccess
}

func ParseLeisureSentiment(text string, selectedLeisure string) map[string]float64 {
	multipliers := map[string]float64{
		"fishing":  1.0,
		"camp":     1.0,
		"hiking":   1.0,
		"shopping": 1.0,
	}

	if text == "" {
		return multipliers
	}

	t, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	if err != nil {
		return multipliers
	}

	tokens := t.Tokenize(text)

	for i, token := range tokens {
		surface := token.Surface

		for leisure, keywords := range leisureKeywords {
			if selectedLeisure != "" && selectedLeisure != "any" && selectedLeisure != leisure {
				continue
			}

			for _, kw := range keywords {
				if strings.Contains(surface, kw) {
					isNegative := false
					isPositive := false

					for j := 1; j <= 3; j++ {
						if i+j < len(tokens) {
							nextSurface := tokens[i+j].Surface
							for _, neg := range negativeWords {
								if strings.Contains(nextSurface, neg) {
									isNegative = true
									break
								}
							}
							for _, pos := range positiveWords {
								if strings.Contains(nextSurface, pos) {
									isPositive = true
									break
								}
							}
						}
					}

					if isNegative {
						multipliers[leisure] = 0.1
					} else if isPositive {
						multipliers[leisure] = 2.0
					} else {
						multipliers[leisure] = 1.4
					}
				}
			}
		}
	}

	return multipliers
}