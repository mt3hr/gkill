package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// MiTempRepository はTX確定前のMi（タスク）を置く一時リポジトリが満たす契約です。
//
// 実体は利用者ごとの一時DB（--cache_in_memory=true ならインメモリ、falseなら
// caches/temp_cache/{userID}_temp_.db）のMI表で、通常の列に加えて
// USER_ID / DEVICE / TX_ID を持ちます。AddMiInfo で貯め、commit_tx が
// GetMisByTXID で取り出して本リポジトリへ書き、discard_tx が DeleteByTXID で捨てます。
//
// 一時データを読むときは (txID, userID, device) を取るメソッドを使ってください。
// FindKyous / GetKyou / FindMi / GetBoardNames など下層のsqlite3実装へ委譲するメソッドは
// TX_ID で絞り込まないうえ、現状のSQLite3実装は共有の一時DBハンドルではなく
// "mi_temp" という名前のDBを開くため、no such table エラーか0件になります。
type MiTempRepository interface {
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
	// sqlite3実装は固定値 "mi_temp" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	// 一時DBは全一時リポジトリで共有しているため、通常のライフサイクルでは呼ばれません。
	Close(ctx context.Context) error

	// FindMi は検索条件に一致するMiを返します。
	// FindKyous と違い Title / BoardName / IsChecked / 期限・予定時刻を持つMiとして返します。
	// 順序は保証しません。
	FindMi(ctx context.Context, query *find.FindQuery) ([]Mi, error)

	// GetMi は id に対応するMiを1件返します。
	// updateTime が nil なら最新バージョンです。見つからない場合は (nil, nil) を返します。
	GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error)

	// GetMiHistories は id の全バージョンを返します。
	GetMiHistories(ctx context.Context, id string) ([]Mi, error)

	// AddMiInfo は mi を (txID, userID, device) 付きで一時DBに追記します。
	// 追記専用なので、同一IDの更新は新しい UPDATE_TIME の版を足すことで表します。
	// LimitTime / EstimateStartTime / EstimateEndTime は nil ならNULLとして格納します。
	AddMiInfo(ctx context.Context, mi Mi, txID string, userID string, device string) error

	// GetBoardNames は一時DBのMI表にあるボード名を重複排除して返します。
	// 版や削除フラグでは絞らないため、旧版や削除済みのボード名も含みます。順序は保証しません。
	GetBoardNames(ctx context.Context) ([]string, error)

	// GetKyousByTXID は (txID, userID, device) の一時データをKyouとして返す想定のメソッドです。
	// ただし現状のSQLite3実装は一時表に無い列（TARGET_REP_NAME / RELATED_TIME）を参照するため
	// 常にエラーを返します。TX単位の取り出しには GetMisByTXID を使ってください。
	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]Kyou, error)

	// GetMisByTXID は (txID, userID, device) が一致する一時データをすべて返します。
	// バージョンの絞り込みはしないので、同一IDに複数版があればすべて含みます。
	// 該当0件はエラーではなく空スライスです。
	GetMisByTXID(ctx context.Context, txID string, userID string, device string) ([]Mi, error)

	// DeleteByTXID は (txID, userID, device) が一致する一時データを行ごと削除します。
	// 本リポジトリの論理削除と違って物理削除で、該当0件でもエラーになりません。
	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	// UnWrapTyped は MiTempRepository 型のままリーフ実装へ平坦化します。
	// 契約は Repository.UnWrap を参照。
	UnWrapTyped() ([]MiTempRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
