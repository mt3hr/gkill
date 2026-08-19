package reps

import (
	"context"
	"net/http"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// IDFKyouTempRepository はTX確定前のIDFKyou（ファイル）を置く一時リポジトリが満たす契約です。
//
// 実体は利用者ごとの一時DB（--cache_in_memory=true ならインメモリ、falseなら
// caches/temp_cache/{userID}_temp_.db）のIDF表で、通常の列に加えて
// USER_ID / DEVICE / TX_ID を持ちます。保持するのはファイルへの参照（TARGET_FILE）だけで、
// ファイル実体・サムネ・互換動画・ZIP展開といった派生物は一切持ちません。
// AddIDFKyouInfo で貯め、commit_tx が GetIDFKyousByTXID で取り出して本リポジトリへ書き、
// discard_tx が DeleteByTXID で捨てます。
//
// 一時データを読むときは (txID, userID, device) を取るメソッドを使ってください。
// FindKyous / GetKyou / FindIDFKyou など下層のsqlite3実装へ委譲するメソッドは
// TX_ID で絞り込まないうえ、現状のSQLite3実装は共有の一時DBハンドルではなく
// 別のDBを開くため、no such table エラーになります。
type IDFKyouTempRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	// 本リポジトリと違い対象ファイルを置くフォルダを持たないため、sqlite3実装は常にエラーを返します。
	GetPath(ctx context.Context, id string) (string, error)

	// UpdateCache は一時リポジトリでは何もしません。
	// 本リポジトリと違い自動IDF（フォルダ走査）の対象を持たないためです。
	// 契約は Repository.UpdateCache を参照。
	UpdateCache(ctx context.Context) error

	// GetLatestDataRepositoryAddress の契約は Repository.GetLatestDataRepositoryAddress を参照。
	// 未確定データは最新版の所在に含めないため、sqlite3実装は常にエラーを返します。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetRepName の契約は Repository.GetRepName を参照。
	// sqlite3実装は固定値 "IDF_TEMP" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	// 一時DBは全一時リポジトリで共有しているため、通常のライフサイクルでは呼ばれません。
	Close(ctx context.Context) error

	// FindIDFKyou は検索条件に一致するIDFKyouを返します。
	// FindKyous と違い TargetFile やファイル種別の判定結果を持つIDFKyouとして返します。
	// 順序は保証しません。
	FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]IDFKyou, error)

	// GetIDFKyou は id に対応するIDFKyouを1件返します。
	// updateTime が nil なら最新バージョンです。見つからない場合は (nil, nil) を返します。
	GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error)

	// GetIDFKyouHistories は id の全バージョンを返します。
	GetIDFKyouHistories(ctx context.Context, id string) ([]IDFKyou, error)

	// IDF は一時リポジトリでは常にエラーを返します。
	// フォルダを走査してファイルをKyou化するのは実ファイルを持つ本リポジトリの仕事です。
	IDF(ctx context.Context) error

	// AddIDFKyouInfo は idfKyou を (txID, userID, device) 付きで一時DBに追記します。
	// idfKyou.RepName は「対象ファイルを持つリポジトリ名」として TARGET_REP_NAME に格納し、
	// idfKyou.TargetFile はそのリポジトリ内での相対パスとして格納します。
	// 追記専用なので、同一IDの更新は新しい UPDATE_TIME の版を足すことで表します。
	AddIDFKyouInfo(ctx context.Context, idfKyou IDFKyou, txID string, userID string, device string) error

	// HandleFileServe は一時リポジトリでは常に500を返します。
	// ファイル配信は実ファイルを持つ本リポジトリが行います。
	HandleFileServe(w http.ResponseWriter, r *http.Request)

	// GetKyousByTXID は (txID, userID, device) が一致する一時データをKyouとして返します。
	// 画像・動画判定（IsImage / IsVideo）は付きますが、ファイルURLは付きません。
	// 該当0件はエラーではなく空スライスです。
	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]Kyou, error)

	// GetIDFKyousByTXID は (txID, userID, device) が一致する一時データをIDFKyouとして返します。
	// GetKyousByTXID と違い FileURL と種別判定（IsImage / IsVideo / IsAudio / IsZip）を埋めます。
	// TARGET_REP_NAME が空文字または "." の行は自リポジトリ名にフォールバックしてURLを組み立てます。
	// バージョンの絞り込みはしないので、同一IDに複数版があればすべて含みます。
	// 該当0件はエラーではなく空スライスです。
	GetIDFKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]IDFKyou, error)

	// DeleteByTXID は (txID, userID, device) が一致する一時データを行ごと削除します。
	// 本リポジトリの論理削除と違って物理削除で、該当0件でもエラーになりません。
	// 参照していたファイル実体には触れません。
	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	// GenerateThumbCache は一時リポジトリでは何もせず nil を返します。
	// 一時リポジトリは派生キャッシュを持たないためです。
	GenerateThumbCache(ctx context.Context) error

	// ClearThumbCache は一時リポジトリでは何もせず nil を返します。
	ClearThumbCache(userID string) error

	// GenerateVideoCache は一時リポジトリでは何もせず nil を返します。
	GenerateVideoCache(ctx context.Context) error

	// ClearVideoCache は一時リポジトリでは何もせず nil を返します。
	ClearVideoCache(userID string) error

	// ClearZipCache は一時リポジトリでは何もせず nil を返します。
	ClearZipCache(userID string) error

	// UnWrapTyped は IDFKyouTempRepository 型のままリーフ実装へ平坦化します。
	// 契約は Repository.UnWrap を参照。
	UnWrapTyped() ([]IDFKyouTempRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
