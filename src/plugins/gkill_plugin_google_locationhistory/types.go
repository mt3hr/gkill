package main

const (
	// repName は gkill 上のリポジトリ表示名。manifest.json の rep_name と一致させること。
	repName = "GoogleLocation"

	// dataType は滞在地のKyouに使う data_type。
	// いまは Kyou を出さないが、manifest.json の必須項目なので予約してある。
	dataType = "google_location_visit"

	// appName はログに出す名前。
	appName = "gkill_plugin_google_locationhistory"
)

// 設定キー。
const (
	configKeySourceDirs        = "source_dirs"
	configKeyAccuracyMaxMeters = "accuracy_max_meters"
	configKeySources           = "sources"
	configKeyIncludeFitbitGPS  = "include_fitbit_gps"
	configKeyMaxPoints         = "max_points"
	configKeyVisitPoints       = "visit_points"

	// JSONにはコメントが書けないので、読み飛ばされるキーで書式を書き残す。
	configKeyComment           = "_comment"
	configKeyExampleSourceDirs = "_example_source_dirs"
)

const (
	// defaultAccuracyMaxMeters は既定の精度フィルタ。
	//
	// 実データの accuracyMm は最大2.6kmまである（CELL測位）。
	// 100mで97%が残り、街を跨ぐような粗い測位だけが落ちる。
	// これを緩めると、地図フィルタが別の街のKyouに当たるようになる。
	defaultAccuracyMaxMeters = 100

	// defaultMaxPoints は返す点数の上限。新しい順に残す。
	defaultMaxPoints = 1000000
)

// 測位の出所。
const (
	sourceFitbit   = "FITBIT"
	sourceVisit    = "VISIT"
	sourceActivity = "ACTIVITY"
	sourceUnknown  = "UNKNOWN"
)

// accuracyUnknown は精度が分からないことを表す。
// フィルタは「測れないものを消さない」ので、この値は常に残す。
const accuracyUnknown = -1

// rawPoint はパース直後の1点。フィルタ・重複除去の前の生の値。
type rawPoint struct {
	UnixMilli  int64
	LatE7      int32
	LngE7      int32
	AccuracyMm int32  // 不明は accuracyUnknown
	Source     string // GPS / WIFI / WIFI_ONLY / CELL / UNKNOWN / FITBIT / VISIT / ACTIVITY
	DeviceID   string // 分かるときだけ（表示用。キーには使わない）
}

// scannedFile は走査で見つかった取り込み対象エントリ。
type scannedFile struct {
	// Path は "<ZIPの絶対パス>!/<ZIP内のパス>"。
	Path string

	// MtimeUnix はエントリの更新時刻。表示用。
	// Takeout は全エントリに書き出し時刻を入れるので、中身の変化の判定には使えない。
	MtimeUnix int64

	Size int64

	// CRC32 は中身の変化を見るための値。差分判定は (CRC32, Size) で行う。
	CRC32 uint32

	Format string
}

// cacheStats は設定画面に出す統計。
type cacheStats struct {
	FileCountByFormat map[string]int
	UnsupportedFiles  []string
	ScanErrors        []string
	TotalPoints       int
	FilteredPoints    int
	UniquePoints      int
	OldestUnixMilli   int64
	NewestUnixMilli   int64
	PointsBySource    map[string]int
	LastScanUnix      int64
	Exports           []exportRow
	Problems          []sourceProblemRow
}

// exportRow は設定画面に出す書き出し1つぶん。
//
// 位置情報では順位付けをしない（読み出し時に (時刻, 緯度, 経度) で重複を除くので、
// 書き出しをまたいで同じ点が二重になることが無い）。表示のためだけに持つ。
type exportRow struct {
	Dir          string
	ArchiveCount int
	NewestUnix   int64
	EntryCount   int
}

// sourceProblemRow は設定画面に出す走査の問題。
type sourceProblemRow struct {
	Kind    string
	Path    string
	Message string
}
