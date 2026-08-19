package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// TimeIsTempRepository はTX確定前のTimeIsを置く一時リポジトリが満たす契約です。
//
// 実体は利用者ごとの一時DB（--cache_in_memory=true ならインメモリ、falseなら
// caches/temp_cache/{userID}_temp_.db）のTIMEIS表で、通常の列に加えて
// USER_ID / DEVICE / TX_ID を持ちます。AddTimeIsInfo で貯め、commit_tx が
// GetTimeIssByTXID で取り出して本リポジトリへ書き、discard_tx が DeleteByTXID で捨てます。
//
// 一時データを読むときは (txID, userID, device) を取るメソッドを使ってください。
// FindKyous / GetKyou / FindTimeIs など下層のsqlite3実装へ委譲するメソッドは
// TX_ID で絞り込まないうえ、現状のSQLite3実装は共有の一時DBハンドルではなく
// "time_is_temp" という名前のDBを開くため、no such table エラーになります。
type TimeIsTempRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	// DBハンドルを外から渡される作りで自分のファイルを持たないため、sqlite3実装は常にエラーを返します。
	GetPath(ctx context.Context, id string) (string, error)

	// UpdateCache は一時リポジトリではキャッシュを持たないため何も作り直しません。
	// 契約は Repository.UpdateCache を参照。
	UpdateCache(ctx context.Context) error

	// GetLatestDataRepositoryAddress の契約は Repository.GetLatestDataRepositoryAddress を参照。
	// 未確定データは最新版の所在に含めないため、sqlite3実装は常にエラーを返します。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetRepName の契約は Repository.GetRepName を参照。
	// sqlite3実装は固定値 "timeis_temp" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	// 一時DBは全一時リポジトリで共有しているため、通常のライフサイクルでは呼ばれません。
	Close(ctx context.Context) error

	// FindTimeIs は検索条件に一致するTimeIsを返します。
	// FindKyous と違い Title / StartTime / EndTime を持つTimeIsとして返します。順序は保証しません。
	FindTimeIs(ctx context.Context, query *find.FindQuery) ([]TimeIs, error)

	// GetTimeIs は id に対応するTimeIsを1件返します。
	// updateTime が nil なら最新バージョンです。見つからない場合は (nil, nil) を返します。
	GetTimeIs(ctx context.Context, id string, updateTime *time.Time) (*TimeIs, error)

	// GetTimeIsHistories は id の全バージョンを返します。
	GetTimeIsHistories(ctx context.Context, id string) ([]TimeIs, error)

	// AddTimeIsInfo は timeis を (txID, userID, device) 付きで一時DBに追記します。
	// 追記専用なので、同一IDの更新は新しい UPDATE_TIME の版を足すことで表します。
	// 計測中で EndTime が nil のTimeIsはそのままNULLとして格納します。
	AddTimeIsInfo(ctx context.Context, timeis TimeIs, txID string, userID string, device string) error

	// GetKyousByTXID は (txID, userID, device) の一時データをKyouとして返す想定のメソッドです。
	// ただし現状のSQLite3実装は一時表に無い列（TARGET_REP_NAME / RELATED_TIME）を参照するため
	// 常にエラーを返します。TX単位の取り出しには GetTimeIssByTXID を使ってください。
	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]Kyou, error)

	// GetTimeIssByTXID は (txID, userID, device) が一致する一時データをすべて返します。
	// バージョンの絞り込みはしないので、同一IDに複数版があればすべて含みます。
	// 該当0件はエラーではなく空スライスです。
	GetTimeIssByTXID(ctx context.Context, txID string, userID string, device string) ([]TimeIs, error)

	// DeleteByTXID は (txID, userID, device) が一致する一時データを行ごと削除します。
	// 本リポジトリの論理削除と違って物理削除で、該当0件でもエラーになりません。
	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	// UnWrapTyped は TimeIsTempRepository 型のままリーフ実装へ平坦化します。
	// 契約は Repository.UnWrap を参照。
	UnWrapTyped() ([]TimeIsTempRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
