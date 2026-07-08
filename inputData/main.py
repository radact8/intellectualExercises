import json
import sqlite3
import pandas as pd

# 1. JSONファイルを辞書型として読み込む
json_file_path = '../data/data.json'
with open(json_file_path, 'r', encoding='utf-8') as f:
    data = json.load(f)

print("=== JSONデータの読み込みに成功しました ===")

# --- 簡易的なネガポジ・キーワード判定用の辞書 ---
sentiment_dict = {
    "toilet": {"pos": ["綺麗", "清潔", "快適", "水回り"], "neg": ["汚い", "古い"]},
    "rental": {"pos": ["手ぶら", "レンタル", "充実", "揃って"], "neg": ["無い", "不便"]},
    "safety": {"pos": ["安全", "安心", "安全柵", "舗装", "子供"], "neg": ["危険", "危ない"]},
    "access": {"pos": ["アクセス", "駅", "バス", "抜群"], "neg": ["狭い", "険しい", "カーブ"]}
}

db_rows = []

# 2. 階層を順番に掘り下げて特徴量（粗点）を計算するループ処理
for item in data["suggestions"]:
    prediction = item["placePrediction"]
    
    # 施設名（カンマの前だけを取得）
    full_text = prediction["text"]["text"]
    spot_name = full_text.split(",")[0]
    
    leisure_type = prediction.get("leisure_type", "unknown")
    lat = prediction["geometry"]["location"]["lat"]
    lng = prediction["geometry"]["location"]["lng"]
    
    # 全クチコミテキストを結合
    all_text = " ".join([r["text"] for r in prediction.get("reviews", [])])
    
    # 各評価軸のスコアリング（簡易テキストマッチング）
    scores = {}
    for category, keywords in sentiment_dict.items():
        pos_count = sum(all_text.count(word) for word in keywords["pos"])
        neg_count = sum(all_text.count(word) for word in keywords["neg"])
        
        # 基準点 0.5 からポジ・ネガに応じて増減
        final_score = 0.5 + (pos_count * 0.25) - (neg_count * 0.35)
        scores[category] = round(max(0.0, min(1.0, final_score)), 2)
        
    # 計算結果をリストに追加
    db_rows.append({
        "spot_name": spot_name,
        "leisure_type": leisure_type,
        "lat": lat,
        "lng": lng,
        "score_toilet": scores["toilet"],
        "score_rental": scores["rental"],
        "score_safety": scores["safety"],
        "score_access": scores["access"]
    })
# 計算結果から Pandas のデータフレーム 'df' を作成する
df = pd.DataFrame(db_rows)

# 3. SQLite3 データベースファイルを作成して接続
db_file = "../db/data.db"
conn = sqlite3.connect(db_file)
cursor = conn.cursor()

# 4. テーブルの作成（SQL）
cursor.execute("""
CREATE TABLE IF NOT EXISTS spots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spot_name TEXT NOT NULL,
    leisure_type TEXT NOT NULL,
    lat REAL NOT NULL,
    lng REAL NOT NULL,
    score_toilet REAL NOT NULL,
    score_rental REAL NOT NULL,
    score_safety REAL NOT NULL,
    score_access REAL NOT NULL
)
""")

print(f"📊 保存直前のデータ件数: {len(df)}件")
print(df.head())

# 5. 既存データを一度クリアして、データフレーム 'df' から SQLite へ流し込む
cursor.execute("DELETE FROM spots")
df.to_sql("spots", conn, if_exists="append", index=False)

# 6. 変更を確定して閉じる
conn.commit()
conn.close()

print(f"✅ SQLiteデータベース '{db_file}' が正常に作成され、テーブルにデータが格納されました！")