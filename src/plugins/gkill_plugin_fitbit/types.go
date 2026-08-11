package main

const (
	// repName は gkill 上のリポジトリ表示名。manifest.json の rep_name と一致させること。
	repName = "Fitbit"

	// dataType は返すKyouのdata_type。
	//
	// "kc" ちょうどにしてあるのは意図的。クライアントは data_type の接頭辞で
	// 型別ビューを出し分けるので、"fitbit_daily" のような独自の値にすると
	// KCの値を取りに行く経路が一度も走らず、推移グラフで集計できなくなる。
	dataType = "kc"

	// appName は Kyou の CreateApp / UpdateApp に入れる名前。
	appName = "gkill_plugin_fitbit"
)

// 設定キー。
const (
	configKeySourceDirs  = "source_dirs"
	configKeyTimezone    = "timezone"
	configKeyMetrics     = "metrics"
	configKeyScanWorkers = "scan_workers"

	// JSONにはコメントが書けないので、読み飛ばされるキーで書式を書き残す。
	configKeyComment           = "_comment"
	configKeyExampleSourceDirs = "_example_source_dirs"
)

// defaultTimezone は「この日はどの日か」を決める既定のタイムゾーン。
const defaultTimezone = "Asia/Tokyo"

// dailyMetric は1日1指標ぶんの集計結果。キャッシュから読み出したそのままの形。
type dailyMetric struct {
	KyouID      string
	MetricKey   string
	DateLocal   string // YYYY-MM-DD（設定タイムゾーンでの日付）
	Title       string
	NumValue    string // json.Number に入れる文字列表現
	Unit        string
	SampleCount int
	MinValue    float64
	MaxValue    float64
	Devices     string // "\n" 区切り
	SourcePaths string // "\n" 区切り
	HourSums    string // 24個のカンマ区切り
	HourCounts  string // 24個のカンマ区切り
	SearchText  string
	RelatedUnix int64
	UpdateUnix  int64
}

// partialDaily はファイル1つが1日に寄与する部分集計。
//
// ファイル単位で持つのが要点。心拍のように現地1日がUTC2ファイルにまたがる場合でも、
// 変化したファイルの寄与だけを差し替えて、日次の値は畳み直せる。
type partialDaily struct {
	MetricKey  string
	DateLocal  string
	SumValue   float64
	CountValue int64
	MinValue   float64
	MaxValue   float64
	LastValue  float64
	LastUnix   int64
	ExportID   string
	Devices    map[string]struct{}
	HourSums   [24]float64
	HourCounts [24]int64
}

// scannedFile は走査で見つかった取り込み対象エントリ。
type scannedFile struct {
	// Path は "<ZIPの絶対パス>!/<ZIP内のパス>"。
	Path string

	// MtimeUnix はエントリの更新時刻。書き出しの新旧の判定にだけ使う。
	// Takeout は全エントリに同じ値を入れるので、中身が変わったかの判定には使えない。
	MtimeUnix int64

	Size int64

	// CRC32 は中身の変化を見るための値。差分判定は (CRC32, Size) で行う。
	CRC32 uint32

	// ExportID は取り込み世代。同じ世代の寄与は足し、別の世代は新しいほうを採る。
	ExportID string

	// Prefix は一致したメトリクスの接頭辞。対象外なら空。
	Prefix string
}

// cacheStats は設定画面に出す統計。
type cacheStats struct {
	TargetFileCount  int
	ScannedFileCount int
	DayCount         int
	MetricCount      int
	KyouCount        int
	LastScanUnix     int64
	Exports          []exportRow
	Problems         []sourceProblemRow
	BuildState       string
	BuildDoneFiles   int
	BuildTotalFiles  int
	BuildError       string
	Timezone         string
}

// exportRow は設定画面に出す書き出し1つぶん。
type exportRow struct {
	ExportID     string
	Dir          string
	ArchiveCount int
	NewestUnix   int64
	Rank         int
	DayCount     int
}

// sourceProblemRow は設定画面に出す走査の問題。
type sourceProblemRow struct {
	Kind    string
	Path    string
	Message string
}
