import json
import sqlite3
import pandas as pd
# main.py の一番最後に追加して中身を確認する用
conn = sqlite3.connect("data.db")
# Pandasを使うと、SQLの結果を綺麗な表形式で表示してくれます
df_check = pd.read_sql_query("SELECT * FROM spots", conn)
conn.close()

print("\n📊 === 現在のデータベース（spotsテーブル）の中身 ===")
print(df_check.to_string(index=False)) # index=False で余計な行番号を非表示に