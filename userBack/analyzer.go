package main

import (
	"strings"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

// ParseUserText はユーザーの自由入力テキストから各評価軸の補正倍率を算出します。
// 外部（main.go）から呼べるように頭文字を大文字にしています。
func ParseUserText(userInput string) (float64, float64, float64, float64) {
	mToilet, mRental, mSafety, mAccess := 1.0, 1.0, 1.0, 1.0

	if userInput == "" {
		return mToilet, mRental, mSafety, mAccess
	}

	// 1. Kagomeの初期化（軽量なIPA辞書を使用）
	t, _ := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	tokens := t.Tokenize(userInput)

	// 2. 形態素を走査してキーワードの「原型（基本形）」を抽出
	var words []string
	for _, token := range tokens {
		if token.Surface != "" {
			features := token.Features()
			baseForm := token.Surface
			if len(features) > 6 && features[6] != "*" {
				baseForm = features[6]
			}
			words = append(words, baseForm)
		}
	}

	// 3. 強調表現のチェック（絶対、重視、必須など）
	isStrong := false
	for _, w := range words {
		if w == "絶対" || w == "必須" || w == "重視" || w == "綺麗さ" || w == "一番" {
			isStrong = true
			break
		}
	}

	// 4. 緩和・否定表現のチェック（気にしない、適当など）
	isWeak := false
	for _, w := range words {
		if w == "気にしない" || w == "適当" || w == "汚い" || w == "無し" || w == "ダメ" {
			isWeak = true
			break
		}
	}

	// 5. 各評価軸のキーワードが含まれているかで倍率を確定
	for _, w := range words {
		// --- トイレの判定 ---
		if w == "トイレ" || w == "綺麗" || w == "清潔" || w == "水回り" {
			if isStrong {
				mToilet = 2.0
			} else if isWeak {
				mToilet = 0.3
			} else {
				mToilet = 1.5
			}
		}
		// --- レンタルの判定 ---
		if w == "手ぶら" || w == "レンタル" || w == "道具" || w == "手軽" {
			if isStrong {
				mRental = 2.0
			} else if isWeak {
				mRental = 0.3
			} else {
				mRental = 1.5
			}
		}
		// --- 安全性の判定 ---
		if w == "安全" || w == "安心" || w == "子供" || w == "ファミリー" || w == "柵" {
			if isStrong {
				mSafety = 2.0
			} else if isWeak {
				mSafety = 0.5
			} else {
				mSafety = 1.5
			}
		}
		// --- アクセスの判定 ---
		if strings.Contains(w, "アクセス") || w == "運転" || w == "道" || w == "狭い" || w == "遠い" || w == "駅" {
			if isStrong || w == "狭い" || w == "遠い" {
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