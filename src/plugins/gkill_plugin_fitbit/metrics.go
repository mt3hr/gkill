package main

import (
	"strings"
)

// aggKind は日次値の決め方。
type aggKind uint8

const (
	aggSum   aggKind = iota // 日内合計
	aggMean                 // 日内平均（サンプル数で重み付け）
	aggMax                  // 日内最大
	aggMin                  // 日内最小
	aggLast                 // その日の最後のサンプル
	aggCount                // 条件に合う行数。1行1分のファイルでは「分」になる
)

// metricDef は「ファイル名の接頭辞 → 1つのKyou系列」の定義。
//
// 1つのファイルから複数の系列を作ることがある（心拍の平均/最大/最小、
// active_minutes の light/moderate/very など）ので、同じ FilePrefix を持つ
// 定義が複数あってよい。取り込みは接頭辞ごとに1回しかファイルを読まない。
//
// Key は KyouID の材料なので絶対に変えないこと。変えると同じ日のKyouが
// 別IDで二重に出る。Title は推移グラフのグループキーになるので、これも実質変えられない。
// どちらかを変えるときは registryVersion を上げてキャッシュを作り直させること。
type metricDef struct {
	Key   string // 内部キー。KyouIDの材料。変更禁止
	Title string // 表示名（日本語）。推移グラフのグループキー
	Unit  string // 表示専用の単位。数値には含めない

	FilePrefix string // "steps" → steps.csv / steps_2024-04-01.csv だけに一致
	Daily      bool   // true: 1行=1日。timestampの先頭10文字をそのまま日付に使う

	TimeCol  string // 時刻列（正規化済みヘッダ名）
	ValueCol string // 値列（正規化済みヘッダ名）。aggCount のときだけ空でよい

	MatchCol   string // 非空なら、この列が MatchValue に等しい行だけ対象
	MatchValue string

	Agg   aggKind
	Scale float64 // 値に掛ける係数（単位換算）。0 は 1 と同じ扱い
	Round int     // 数値にするときの小数桁
}

// registryVersion はレジストリの世代。
// Key / Title / Agg / Scale / Daily を変えたら上げる。
// キャッシュはこの値が変わると作り直される（値の意味が変わるため）。
const registryVersion = "1"

// metricRegistry は取り込む指標の一覧。
//
// 対象は Takeout の Google Health/Physical Activity_GoogleData 配下。
// 別のフォルダ（睡眠スコアなど）に対応したくなったら、ここに1行足すだけでよい。
var metricRegistry = []metricDef{
	// ---- サンプル系: 日内で集計する ----
	{Key: "steps_daily", Title: "歩数(日計)", Unit: "歩",
		FilePrefix: "steps", TimeCol: "timestamp", ValueCol: "steps", Agg: aggSum, Round: 0},
	{Key: "distance_daily", Title: "距離(日計)", Unit: "m",
		FilePrefix: "distance", TimeCol: "timestamp", ValueCol: "distance", Agg: aggSum, Round: 1},
	{Key: "calories_daily", Title: "消費カロリー(日計)", Unit: "kcal",
		FilePrefix: "calories", TimeCol: "timestamp", ValueCol: "calories", Agg: aggSum, Round: 1},
	{Key: "active_energy_daily", Title: "活動カロリー(日計)", Unit: "kcal",
		FilePrefix: "active_energy_burned", TimeCol: "timestamp", ValueCol: "kilocalories", Agg: aggSum, Round: 1},
	{Key: "floors_daily", Title: "上った階数", Unit: "階",
		FilePrefix: "floors", TimeCol: "timestamp", ValueCol: "floors", Agg: aggSum, Round: 0},
	// altitude の gain はミリメートル単位（readme に明記）なのでメートルに直す
	{Key: "altitude_gain_daily", Title: "上昇高度", Unit: "m",
		FilePrefix: "altitude", TimeCol: "timestamp", ValueCol: "gain", Agg: aggSum, Scale: 0.001, Round: 1},

	// 心拍は同じファイルから3系列。ファイルを読むのは1回だけ
	{Key: "heart_rate_avg", Title: "心拍数(日平均)", Unit: "bpm",
		FilePrefix: "heart_rate", TimeCol: "timestamp", ValueCol: "beatsperminute", Agg: aggMean, Round: 1},
	{Key: "heart_rate_max", Title: "心拍数(最大)", Unit: "bpm",
		FilePrefix: "heart_rate", TimeCol: "timestamp", ValueCol: "beatsperminute", Agg: aggMax, Round: 0},
	{Key: "heart_rate_min", Title: "心拍数(最小)", Unit: "bpm",
		FilePrefix: "heart_rate", TimeCol: "timestamp", ValueCol: "beatsperminute", Agg: aggMin, Round: 0},

	{Key: "body_temperature_avg", Title: "皮膚温(日平均)", Unit: "℃",
		FilePrefix: "body_temperature", TimeCol: "timestamp", ValueCol: "temperaturecelsius", Agg: aggMean, Round: 2},
	{Key: "spo2_avg", Title: "血中酸素(日平均)", Unit: "%",
		FilePrefix: "oxygen_saturation", TimeCol: "timestamp", ValueCol: "oxygensaturationpercentage", Agg: aggMean, Round: 1},

	{Key: "active_minutes_light", Title: "活動時間(軽度)", Unit: "分",
		FilePrefix: "active_minutes", TimeCol: "timestamp", ValueCol: "light", Agg: aggSum, Round: 0},
	{Key: "active_minutes_moderate", Title: "活動時間(中程度)", Unit: "分",
		FilePrefix: "active_minutes", TimeCol: "timestamp", ValueCol: "moderate", Agg: aggSum, Round: 0},
	{Key: "active_minutes_very", Title: "活動時間(高強度)", Unit: "分",
		FilePrefix: "active_minutes", TimeCol: "timestamp", ValueCol: "very", Agg: aggSum, Round: 0},

	{Key: "azm_total", Title: "アクティブゾーン時間", Unit: "分",
		FilePrefix: "active_zone_minutes", TimeCol: "timestamp", ValueCol: "totalminutes", Agg: aggSum, Round: 0},
	{Key: "azm_fat_burn", Title: "アクティブゾーン時間(脂肪燃焼)", Unit: "分",
		FilePrefix: "active_zone_minutes", TimeCol: "timestamp", ValueCol: "totalminutes",
		MatchCol: "heartratezone", MatchValue: "FAT_BURN", Agg: aggSum, Round: 0},
	{Key: "azm_cardio", Title: "アクティブゾーン時間(有酸素)", Unit: "分",
		FilePrefix: "active_zone_minutes", TimeCol: "timestamp", ValueCol: "totalminutes",
		MatchCol: "heartratezone", MatchValue: "CARDIO", Agg: aggSum, Round: 0},
	{Key: "azm_peak", Title: "アクティブゾーン時間(ピーク)", Unit: "分",
		FilePrefix: "active_zone_minutes", TimeCol: "timestamp", ValueCol: "totalminutes",
		MatchCol: "heartratezone", MatchValue: "PEAK", Agg: aggSum, Round: 0},

	{Key: "cardio_load_total", Title: "心肺負荷",
		FilePrefix: "cardio_load", TimeCol: "timestamp", ValueCol: "total", Agg: aggSum, Round: 1},

	// カテゴリ列は「そのレベルだった行数 = 分数」として数える。
	// activity_level は1行1分なので、件数がそのまま分になる。
	{Key: "activity_level_sedentary", Title: "座位時間", Unit: "分",
		FilePrefix: "activity_level", TimeCol: "timestamp",
		MatchCol: "level", MatchValue: "SEDENTARY", Agg: aggCount, Round: 0},
	{Key: "activity_level_lightly", Title: "低活動時間", Unit: "分",
		FilePrefix: "activity_level", TimeCol: "timestamp",
		MatchCol: "level", MatchValue: "LIGHTLY_ACTIVE", Agg: aggCount, Round: 0},
	{Key: "activity_level_moderately", Title: "中活動時間", Unit: "分",
		FilePrefix: "activity_level", TimeCol: "timestamp",
		MatchCol: "level", MatchValue: "MODERATELY_ACTIVE", Agg: aggCount, Round: 0},
	{Key: "activity_level_very", Title: "高活動時間", Unit: "分",
		FilePrefix: "activity_level", TimeCol: "timestamp",
		MatchCol: "level", MatchValue: "VERY_ACTIVE", Agg: aggCount, Round: 0},

	// 単位換算が要るもの。サンプルが極端に少ないので aggLast でよい
	{Key: "weight", Title: "体重", Unit: "kg",
		FilePrefix: "weight", TimeCol: "timestamp", ValueCol: "weightgrams",
		Agg: aggLast, Scale: 0.001, Round: 1},
	{Key: "height", Title: "身長", Unit: "cm",
		FilePrefix: "height", TimeCol: "timestamp", ValueCol: "heightmillimeters",
		Agg: aggLast, Scale: 0.1, Round: 1},

	// ---- 既に日次のもの: タイムスタンプの日付部分をそのまま使う ----
	//
	// これらの時刻は「現地の暦日に見せかけの時刻を付けたもの」であって、
	// UTCの瞬間ではない（daily_readiness.csv は裸の 2024-11-13 と書く）。
	// タイムゾーン変換すると全部1日ずれるので Daily で分けている。
	{Key: "resting_heart_rate", Title: "安静時心拍数", Unit: "bpm", Daily: true,
		FilePrefix: "daily_resting_heart_rate", TimeCol: "timestamp", ValueCol: "beatsperminute",
		Agg: aggLast, Round: 1},
	{Key: "readiness_score", Title: "体調スコア", Daily: true,
		FilePrefix: "daily_readiness", TimeCol: "timestamp", ValueCol: "score", Agg: aggLast, Round: 0},
	{Key: "spo2_daily_avg", Title: "血中酸素(Fitbit日平均)", Unit: "%", Daily: true,
		FilePrefix: "daily_oxygen_saturation", TimeCol: "timestamp", ValueCol: "averagepercentage",
		Agg: aggLast, Round: 1},
	{Key: "hrv_daily", Title: "心拍変動(HRV)", Unit: "ms", Daily: true,
		FilePrefix: "daily_heart_rate_variability", TimeCol: "timestamp",
		ValueCol: "averageheartratevariabilitymilliseconds", Agg: aggLast, Round: 1},
	{Key: "non_rem_heart_rate", Title: "ノンレム時心拍数", Unit: "bpm", Daily: true,
		FilePrefix: "daily_heart_rate_variability", TimeCol: "timestamp",
		ValueCol: "nonremheartratebeatsperminute", Agg: aggLast, Round: 1},
	{Key: "respiratory_rate", Title: "呼吸数", Unit: "回/分", Daily: true,
		FilePrefix: "daily_respiratory_rate", TimeCol: "timestamp", ValueCol: "breathsperminute",
		Agg: aggLast, Round: 1},
	{Key: "vo2max", Title: "推定VO2Max", Daily: true,
		FilePrefix: "demographic_vo2max", TimeCol: "timestamp", ValueCol: "demographicvo2max",
		Agg: aggLast, Round: 2},
	{Key: "cardio_load_ratio", Title: "心肺負荷比", Daily: true,
		FilePrefix: "cardio_acute_chronic_workload_ratio", TimeCol: "timestamp", ValueCol: "ratio",
		Agg: aggLast, Round: 2},
	{Key: "sleep_skin_temperature", Title: "睡眠時皮膚温", Unit: "℃", Daily: true,
		FilePrefix: "daily_sleep_temperature_derivations", TimeCol: "timestamp",
		ValueCol: "nightlytemperaturecelsius", Agg: aggLast, Round: 2},
}

// metricsByPrefix は接頭辞から定義を引くための索引。
var metricsByPrefix = buildMetricsByPrefix()

func buildMetricsByPrefix() map[string][]metricDef {
	byPrefix := map[string][]metricDef{}
	for _, def := range metricRegistry {
		byPrefix[def.FilePrefix] = append(byPrefix[def.FilePrefix], def)
	}
	return byPrefix
}

// metricByKey はキーから定義を引くための索引。
var metricByKey = buildMetricByKey()

func buildMetricByKey() map[string]metricDef {
	byKey := map[string]metricDef{}
	for _, def := range metricRegistry {
		byKey[def.Key] = def
	}
	return byKey
}

// metricPrefixOf は "steps_2024-04-01.csv" / "weight.csv" から接頭辞を取り出す。
//
// strings.HasPrefix ではいけない。
// heart_rate_variability_2024-04-01.csv が接頭辞 "heart_rate" に食われる。
// 日付サフィックスを構造として剥がしてから完全一致で引く。
func metricPrefixOf(baseName string) (string, bool) {
	name, found := strings.CutSuffix(strings.ToLower(baseName), ".csv")
	if !found {
		return "", false
	}
	// _YYYY-MM-DD（11文字）が付いていれば剥がす
	const dateSuffixLen = len("_2024-04-01")
	if len(name) > dateSuffixLen {
		tail := name[len(name)-dateSuffixLen:]
		if tail[0] == '_' && isDateShape(tail[1:]) {
			name = name[:len(name)-dateSuffixLen]
		}
	}
	if _, exist := metricsByPrefix[name]; !exist {
		return "", false
	}
	return name, true
}

// isDateShape は "2024-04-01" の形をしているかを返す。
func isDateShape(s string) bool {
	if len(s) != len("2024-04-01") {
		return false
	}
	for i, r := range s {
		if i == 4 || i == 7 {
			if r != '-' {
				return false
			}
			continue
		}
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// normHeader はヘッダ名を比較用に正規化する。
// 空白と _ を落として小文字にする（"Beats per minute" → "beatsperminute"）。
func normHeader(header string) string {
	normalized := strings.ToLower(strings.TrimSpace(header))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "_", "")
	return normalized
}

// findCol は正規化済みヘッダ名の位置を返す。無ければ -1。
func findCol(headers []string, name string) int {
	if name == "" {
		return -1
	}
	for i, header := range headers {
		if normHeader(header) == name {
			return i
		}
	}
	return -1
}

// scaleOf は係数を返す。0 は 1 として扱う。
func (d metricDef) scaleOf() float64 {
	if d.Scale == 0 {
		return 1
	}
	return d.Scale
}
