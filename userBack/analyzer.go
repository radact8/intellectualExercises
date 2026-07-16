package main

import (
	"strings"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

// ParseUserText はユーザーの自由入力テキストから各評価軸の補正倍率を算出します。
func ParseUserText(userInput string) (float64, float64, float64, float64) {
	mToilet, mRental, mSafety, mAccess := 1.0, 1.0, 1.0, 1.0

	if userInput == "" {
		return mToilet, mRental, mSafety, mAccess
	}

	// 1. Kagomeの初期化
	t, _ := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	tokens := t.Tokenize(userInput)

	// 2. 扱いやすいように、意味のある単語（原型）と品詞、さらに元の表記(Surface)をスライスにまとめる
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
				pos = features[0] // 品詞（名詞、動詞、形容詞、助動詞など）
			}
			if len(features) > 6 && features[6] != "*" {
				baseForm = features[6] // 原型
			}
			words = append(words, wordInfo{surface: token.Surface, base: baseForm, pos: pos})
		}
	}

	// 3. ローカル（周辺単語）のコンテキストをスキャンするヘルパー関数
	// 指定したインデックス(idx)の周辺（前後2〜3単語）に強調や否定があるかを判定する
	checkContext := func(idx int) (isStrong, isWeak bool) {
		// 局所的なキーワードの定義
		strongWords := map[string]bool{"絶対": true, "必須": true, "重視": true, "一番": true, "超": true, "とても": true}
		weakWords := map[string]bool{"気にしない": true, "適当": true, "無し": true, "ダメ": true, "ない": true, "なかった": true, "不要": true}

		// 判定対象の前後2単語を調べる（配列の範囲外に注意）
		start := idx - 2
		if start < 0 { start = 0 }
		end := idx + 3
		if end > len(words) { end = len(words) }

		for i := start; i < end; i++ {
			w := words[i].base
			if strongWords[w] {
				isStrong = true
			}
			if weakWords[w] || words[i].pos == "動詞-非自立" && w == "ある" { // 「〜ではない」の対応など
				isWeak = true
			}
		}
		return isStrong, isWeak
	}

	// 4. 各トークンを走査し、見つかったキーワードの周辺文脈を評価して倍率を決定
	for i, w := range words {
		// --- トイレの判定 ---
		if w.base == "トイレ" || w.base == "綺麗" || w.base == "清潔" || w.base == "水回り" || strings.Contains(w.surface, "綺麗") {
			isStrong, isWeak := checkContext(i)
			if isStrong { mToilet = 2.0 } else if isWeak { mToilet = 0.3 } else { mToilet = 1.5 }
		}

		// --- レンタルの判定 ---
		if w.base == "手ぶら" || w.base == "レンタル" || w.base == "道具" || w.base == "手軽" {
			isStrong, isWeak := checkContext(i)
			if isStrong { mRental = 2.0 } else if isWeak { mRental = 0.3 } else { mRental = 1.5 }
		}

		// --- 安全性の判定 ---
		if w.base == "安全" || w.base == "安心" || w.base == "子供" || w.base == "ファミリー" || w.base == "柵" {
			isStrong, isWeak := checkContext(i)
			if isStrong { mSafety = 2.0 } else if isWeak { mSafety = 0.5 } else { mSafety = 1.5 }
		}

		// --- アクセスの判定 ---
		if strings.Contains(w.base, "アクセス") || w.base == "運転" || w.base == "道" || w.base == "狭い" || w.base == "遠い" || w.base == "駅" {
			isStrong, isWeak := checkContext(i)
			// 「狭い」「遠い」は単体でアクセスへの強い懸念（強調）として扱う
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