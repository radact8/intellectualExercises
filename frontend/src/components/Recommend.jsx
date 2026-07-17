import React, { useState } from 'react';

export default function RecommendApp() {
  // 1. フォームの入力状態（リクエストJSONに対応）
  const [formData, setFormData] = useState({
    leisure_type: 'any',
    experience: 'beginner',
    weight_toilet: 1.0,
    weight_rental: 1.0,
    user_text: '',
    date_string: '',
  });

  // 2. 通信状態とレスポンスデータ
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // 入力変更ハンドラー
  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  // フォーム送信（Go APIへのPOSTリクエスト）
  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError('');

    // 日時フォーマットの調整（datetime-local の "YYYY-MM-DDTHH:mm" を "YYYY-MM-DD HH:mm:00" に変換）
    let formattedDate = '';
    if (formData.date_string) {
      formattedDate = formData.date_string.replace('T', ' ') + ':00';
    }

    const payload = {
      ...formData,
      weight_toilet: parseFloat(formData.weight_toilet),
      weight_rental: parseFloat(formData.weight_rental),
      date_string: formattedDate,
    };

    // RecommendApp.jsx の handleSubmit 内
  try {
    const response = await fetch('http://localhost:8080/api/recommend', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      // 🔥 Goから返ってきた 400 エラーメッセージを取得して表示する
      const errorText = await response.text();
      throw new Error(errorText || '入力内容にエラーがあります。');
    }

    const data = await response.json();
    setResults(data);
  } catch (err) {
    setError(err.message); // 画面上に「リクエストエラー: 予定日時が入力されていません」と赤字で表示される
  } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto p-6 bg-gray-50 min-h-screen">
      <h1 className="text-2xl font-bold text-gray-800 mb-6 text-center">
        🌲 レジャー・おでかけスポット推薦
      </h1>

      {/* 📥 1. 入力フォーム領域 */}
      <form onSubmit={handleSubmit} className="bg-white p-6 rounded-lg shadow-md mb-8 space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* レジャー種別（選択式） */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">レジャー種別</label>
            <select
              name="leisure_type"
              value={formData.leisure_type}
              onChange={handleChange}
              className="w-full border border-gray-300 rounded-md p-2 focus:ring-2 focus:ring-blue-500"
            >
              <option value="any">おまかせ（全ジャンル）</option>
              <option value="fishing">釣り (fishing)</option>
              <option value="camp">キャンプ (camp)</option>
              <option value="hiking">ハイキング (hiking)</option>
              <option value="shopping">ショッピング (shopping)</option>
            </select>
          </div>

          {/* 習熟度（選択式） */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">あなたの経験度</label>
            <select
              name="experience"
              value={formData.experience}
              onChange={handleChange}
              className="w-full border border-gray-300 rounded-md p-2 focus:ring-2 focus:ring-blue-500"
            >
              <option value="beginner">初心者 (beginner)</option>
              <option value="experienced">経験者 (experienced)</option>
            </select>
          </div>

          {/* 行く予定日時（選択式・カレンダー picker） */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">予定日時</label>
            <input
              type="datetime-local"
              name="date_string"
              value={formData.date_string}
              onChange={handleChange}
              className="w-full border border-gray-300 rounded-md p-2 focus:ring-2 focus:ring-blue-500"
            />
          </div>

          {/* トイレ優先度（スライダーまたは数値選択） */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              トイレの綺麗さ重視度: {formData.weight_toilet}
            </label>
            <input
              type="range"
              min="0.5"
              max="2.0"
              step="0.1"
              name="weight_toilet"
              value={formData.weight_toilet}
              onChange={handleChange}
              className="w-full"
            />
          </div>
        </div>

        {/* 自由入力欄（テキスト入力） */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            こだわり・気になること（自由入力）
          </label>
          <input
            type="text"
            name="user_text"
            value={formData.user_text}
            onChange={handleChange}
            placeholder="例: 絶対にトイレが綺麗な手ぶらで行ける場所がいい / 山道は運転したくない"
            className="w-full border border-gray-300 rounded-md p-2 focus:ring-2 focus:ring-blue-500"
          />
        </div>

        {/* 送信ボタン */}
        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 text-white font-bold py-2 px-4 rounded-md hover:bg-blue-700 transition duration-200 disabled:bg-gray-400"
        >
          {loading ? 'AIが最適なスポットを検索中...' : 'おすすめスポットを検索'}
        </button>
      </form>

      {/* ⚠️ エラー表示 */}
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-6">
          {error}
        </div>
      )}

      {/* 📤 2. レスポンス結果表示領域 */}
<div className="space-y-4">
  <h2 className="text-xl font-bold text-gray-800 mb-4">検索結果 ({results.length}件)</h2>

  {results.map((spot, index) => (
    <div
      key={spot.id || index}
      className="bg-white p-5 rounded-lg shadow border-l-4 border-blue-500 hover:shadow-lg transition flex flex-col md:flex-row gap-4"
    >
      {/* 📸 施設画像 (存在する場合のみ表示) */}
      {spot.image_url && (
        <div className="w-full md:w-48 h-36 flex-shrink-0">
          <img
            src={spot.image_url}
            alt={spot.spot_name}
            className="w-full h-full object-cover rounded-md"
          />
        </div>
      )}

      {/* 📝 施設詳細メイン情報 */}
      <div className="flex-1">
        <div className="flex justify-between items-start mb-1">
          <div>
            <span className="text-sm font-bold text-blue-600 mr-2">#{index + 1}</span>
            <span className="text-xs bg-gray-200 text-gray-700 px-2 py-1 rounded uppercase mr-2">
              {spot.leisure_type}
            </span>
            <h3 className="text-lg font-bold text-gray-900 inline">{spot.spot_name}</h3>
          </div>
          <div className="text-right">
            <span className="text-2xl font-extrabold text-blue-600">{spot.final_score}</span>
            <span className="text-xs text-gray-500 ml-1">点</span>
          </div>
        </div>

        {/* 📍 住所 & 説明文 */}
        {spot.address && (
          <p className="text-xs text-gray-500 mb-1">📍 {spot.address}</p>
        )}
        {spot.description && (
          <p className="text-sm text-gray-600 mb-3 line-clamp-2">{spot.description}</p>
        )}

        {/* 🌤️ ピンポイント天気 & 警告情報（Goから返ってきた場合表示） */}
        {(spot.wind_speed !== undefined || spot.weather_warning) && (
          <div className="bg-blue-50 border border-blue-100 p-2.5 rounded-md mb-3 text-xs">
            <div className="flex items-center justify-between text-blue-800 font-medium">
              <span>🌤️ 現地予報 (風速: {spot.wind_speed}m/s | 雨量: {spot.rain_volume}mm/h)</span>
            </div>
            {spot.weather_warning && (
              <p className="text-red-600 font-bold mt-1">{spot.weather_warning}</p>
            )}
          </div>
        )}

        {/* 📊 各評価軸のスコア（素点）バッジ */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-2 text-xs bg-gray-50 p-2.5 rounded mb-3">
          <div>🚽 トイレ: <span className="font-bold">{spot.score_toilet}</span></div>
          <div>🎒 レンタル: <span className="font-bold">{spot.score_rental}</span></div>
          <div>🛡️ 安全性: <span className="font-bold">{spot.score_safety}</span></div>
          <div>🚗 アクセス: <span className="font-bold">{spot.score_access}</span></div>
        </div>

        {/* 🔗 公式サイト・予約リンク */}
        {spot.url && (
          <div className="text-right">
            <a
              href={spot.url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center text-xs bg-blue-600 hover:bg-blue-700 text-white font-bold py-1.5 px-3 rounded transition duration-150"
            >
              公式サイト / 予約ページを見る ↗
            </a>
          </div>
        )}
      </div>
    </div>
  ))}

  {!loading && results.length === 0 && (
    <p className="text-center text-gray-500 py-8">条件を指定して検索してください。</p>
  )}
</div>
    </div>
  );
}