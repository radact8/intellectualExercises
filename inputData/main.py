import json
import sqlite3
import pandas as pd
import MeCab
import ipadic

# 1. MeCabの初期化
tagger = MeCab.Tagger()

def calculate_review_scores(reviews, sentiment_dict):
    tokens = []
    for r in reviews:
        review_text = r.get("text", "")
        node = tagger.parseToNode(review_text)
        
        while node:
            if node.surface != "":
                features = node.feature.split(',')
                pos = features[0]
                base_form = features[6] if len(features) > 6 and features[6] != "*" else node.surface
                
                if pos in ["名詞", "形容詞", "動詞", "副詞", "助動詞"]:
                    tokens.append({
                        "surface": node.surface,
                        "pos": pos,
                        "base": base_form
                    })
            node = node.next

    calculated_scores = {}

    for category, keywords in sentiment_dict.items():
        pos_count = 0
        neg_count = 0

        for i, token in enumerate(tokens):
            if token["base"] in keywords["pos"]:
                is_negated = False
                for j in range(1, 3):
                    if i + j < len(tokens):
                        next_token = tokens[i + j]
                        if next_token["base"] in ["ない", "ぬ", "ず", "違う", "ダメ"] or "否定" in next_token["pos"]:
                            is_negated = True
                            break
                
                if is_negated:
                    neg_count += 1
                else:
                    pos_count += 1

            elif token["base"] in keywords["neg"]:
                neg_count += 1

        final_score = 0.5 + (pos_count * 0.25) - (neg_count * 0.35)
        calculated_scores[category] = round(max(0.0, min(1.0, final_score)), 2)

    return calculated_scores


# ==========================================
# メイン処理フェーズ
# ==========================================

json_file_path = '../data/data.json'
with open(json_file_path, 'r', encoding='utf-8') as f:
    data = json.load(f)

print("=== JSONデータの読み込みに成功しました ===")

sentiment_dict = {
    "toilet": {"pos": ["綺麗", "清潔", "快適", "水回り"], "neg": ["汚い", "古い"]},
    "rental": {"pos": ["手ぶら", "レンタル", "充実", "揃う"], "neg": ["無い", "不便"]},
    "safety": {"pos": ["安全", "安心", "柵", "舗装", "子供"], "neg": ["危険", "危ない"]},
    "access": {"pos": ["アクセス", "駅", "バス", "抜群"], "neg": ["狭い", "険しい", "カーブ"]}
}

db_rows = []

for item in data["suggestions"]:
    prediction = item["placePrediction"]
    
    full_text = prediction["text"]["text"]
    spot_name = full_text.split(",")[0]
    
    leisure_type = prediction.get("leisure_type", "unknown")
    lat = prediction["geometry"]["location"]["lat"]
    lng = prediction["geometry"]["location"]["lng"]
    
    # 🔥 追加フィールドの取得（JSONに存在しない場合のデフォルト値を設定）
    address = prediction.get("address", full_text)
    url = prediction.get("url", "")
    image_url = prediction.get("image_url", "")
    description = prediction.get("description", "")
    
    reviews = prediction.get("reviews", [])
    scores = calculate_review_scores(reviews, sentiment_dict)
        
    db_rows.append({
        "spot_name": spot_name,
        "leisure_type": leisure_type,
        "lat": lat,
        "lng": lng,
        "score_toilet": scores["toilet"],
        "score_rental": scores["rental"],
        "score_safety": scores["safety"],
        "score_access": scores["access"],
        # 🔥 カラム追加
        "address": address,
        "url": url,
        "image_url": image_url,
        "description": description
    })

df = pd.DataFrame(db_rows)

db_file = "../db/data.db"
conn = sqlite3.connect(db_file)
cursor = conn.cursor()

# 🔥 新しいカラム（address, url, image_url, description）を含めたテーブル作成
cursor.execute("""
CREATE TABLE IF NOT EXISTS spots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spot_name TEXT NOT NULL UNIQUE,
    leisure_type TEXT NOT NULL,
    lat REAL NOT NULL,
    lng REAL NOT NULL,
    score_toilet REAL NOT NULL,
    score_rental REAL NOT NULL,
    score_safety REAL NOT NULL,
    score_access REAL NOT NULL,
    address TEXT DEFAULT '',
    url TEXT DEFAULT '',
    image_url TEXT DEFAULT '',
    description TEXT DEFAULT ''
)
""")

cursor.execute("DELETE FROM spots")
cursor.execute("DELETE FROM sqlite_sequence WHERE name='spots'")
df.to_sql("spots", conn, if_exists="append", index=False)
conn.commit()
conn.close()

print(f"✅ SQLiteデータベース '{db_file}' への構造化永続化が完了しました！")
print(df.head())