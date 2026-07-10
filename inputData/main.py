import json
import sqlite3
import pandas as pd
import MeCab
import ipadic

# 1. MeCabの初期化（一度だけ生成して使い回す）
tagger = MeCab.Tagger()

def calculate_review_scores(reviews, sentiment_dict):
    """
    クチコミのリストから、形態素解析（原型統一＋否定表現チェック）を用いて
    各カテゴリ（toilet, rental, safety, access）のスコアを計算する関数
    
    戻り値: dict -> {"toilet": 0.XX, "rental": 0.XX, ...}
    """
    # 該当スポットの全レビューテキストを1つのトークンリストに分解
    tokens = []
    for r in reviews:
        review_text = r.get("text", "")
        node = tagger.parseToNode(review_text)
        
        while node:
            if node.surface != "":
                features = node.feature.split(',')
                # 特徴量から品詞と原型（基本形）を抽出
                pos = features[0]
                base_form = features[6] if len(features) > 6 and features[6] != "*" else node.surface
                
                # 意味を持つ主要な品詞のみをコンテキスト対象とする
                if pos in ["名詞", "形容詞", "動詞", "副詞", "助動詞"]:
                    tokens.append({
                        "surface": node.surface,
                        "pos": pos,
                        "base": base_form
                    })
            node = node.next

    # 各カテゴリの最終スコアを格納する辞書
    calculated_scores = {}

    # カテゴリごとに形態素の並びを解析
    for category, keywords in sentiment_dict.items():
        pos_count = 0
        neg_count = 0

        for i, token in enumerate(tokens):
            # 1. ポジティブ語の原型マッチング
            if token["base"] in keywords["pos"]:
                # 直後2トークンを覗き見（ルックアヘッド）して否定語があるかチェック
                is_negated = False
                for j in range(1, 3):
                    if i + j < len(tokens):
                        next_token = tokens[i + j]
                        if next_token["base"] in ["ない", "ぬ", "ず", "違う", "ダメ"] or "否定" in next_token["pos"]:
                            is_negated = True
                            break
                
                if is_negated:
                    neg_count += 1  # 「綺麗」＋「ない」 ➔ ネガティブとしてカウント
                else:
                    pos_count += 1  # 純粋なポジティブ

            # 2. ネガティブ語の原型マッチング
            elif token["base"] in keywords["neg"]:
                # 同様に「汚い」＋「ない」などの二重否定があればポジティブに反転可能（今回はシンプルにネガティブカウント）
                neg_count += 1

        # 従来のロジックと同じ計算式（基準点 0.5 から増減、0.0〜1.0に丸める）
        final_score = 0.5 + (pos_count * 0.25) - (neg_count * 0.35)
        calculated_scores[category] = round(max(0.0, min(1.0, final_score)), 2)

    return calculated_scores


# ==========================================
# メイン処理フェーズ
# ==========================================

# JSONファイルを辞書型として読み込む
json_file_path = '../data/data.json'
with open(json_file_path, 'r', encoding='utf-8') as f:
    data = json.load(f)

print("=== JSONデータの読み込みに成功しました ===")

# 解析対象のネガポジ辞書定義
sentiment_dict = {
    "toilet": {"pos": ["綺麗", "清潔", "快適", "水回り"], "neg": ["汚い", "古い"]},
    "rental": {"pos": ["手ぶら", "レンタル", "充実", "揃う"], "neg": ["無い", "不便"]},
    "safety": {"pos": ["安全", "安心", "柵", "舗装", "子供"], "neg": ["危険", "危ない"]},
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
    
    # クチコミのリストを取得
    reviews = prediction.get("reviews", [])
    
    # 🔥 【関数呼び出し】形態素解析ロジックへデータを流し込む
    scores = calculate_review_scores(reviews, sentiment_dict)
        
    # 計算結果をリストに追加（戻り値のキーをそのままマッピング）
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
# spot_name を UNIQUE にすることで、重複登録を防ぎます
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
    score_access REAL NOT NULL
)
""")

# 5. 既存データをクリアして新しいデータを流し込む
# 定期実行時、毎回まっさらな状態から最新の形態素解析結果を反映させます
cursor.execute("DELETE FROM spots")
# 🔥 これを追加：spotsテーブルの連番カウントを完全に初期化（0クリア）する
cursor.execute("DELETE FROM sqlite_sequence WHERE name='spots'")
df.to_sql("spots", conn, if_exists="append", index=False)
conn.commit()
conn.close()

print(f"✅ SQLiteデータベース '{db_file}' への構造化永続化が完了しました！")
print(df.head())