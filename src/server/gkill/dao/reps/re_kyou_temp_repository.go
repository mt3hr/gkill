package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// ReKyouTempRepository はTX確定前のReKyou（再投稿）を置く一時リポジトリが満たす契約です。
//
// 実体は利用者ごとの一時DB（--cache_in_memory=true ならインメモリ、falseなら
// caches/temp_cache/{userID}_temp_.db）のREKYOU表で、通常の列に加えて
// USER_ID / DEVICE / TX_ID を持ちます。保持するのは再投稿元Kyouへの参照（TARGET_ID）だけです。
// AddReKyouInfo で貯め、commit_tx が GetReKyousByTXID で取り出して本リポジトリへ書き、
// discard_tx が DeleteByTXID で捨てます。
//
// 一時データを読むときは (txID, userID, device) を取るメソッドを使ってください。
// FindKyous / GetKyou / FindReKyou / GetReKyousAllLatest など下層のsqlite3実装へ委譲する
// メソッドは TX_ID で絞り込まないうえ、現状のSQLite3実装は共有の一時DBハンドルではなく
// "rekyou_temp" という名前のDBを開くため、no such table エラーになります。
type ReKyouTempRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath は一時リポジトリでは常にエラーを返します。
	// DBハンドルを外から渡される作りで、自分のファイルを持たないためです。
	GetPath(ctx context.Context, id string) (string, error)

	// UpdateCache は一時リポジトリでは何もしません。
	// 契約は Repository.UpdateCache を参照。
	UpdateCache(ctx context.Context) error

	// GetLatestDataRepositoryAddress は一時リポジトリでは常にエラーを返します。
	// 未確定データは「最新版の所在」の対象外だからです。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetRepName は固定名 "rekyou_temp" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	// 一時DBは全一時リポジトリで共有しているため、通常のライフサイクルでは呼ばれません。
	Close(ctx context.Context) error

	// FindReKyou は検索条件に一致するReKyouを返します。
	// ReKyou自身は本文を持たないため、実際の絞り込みはターゲットKyou側の解決結果に依存します。
	// 順序は保証しません。
	FindReKyou(ctx context.Context, query *find.FindQuery) ([]ReKyou, error)

	// GetReKyou は id に対応するReKyouを1件返します。
	// updateTime が nil なら最新バージョンです。見つからない場合は (nil, nil) を返します。
	GetReKyou(ctx context.Context, id string, updateTime *time.Time) (*ReKyou, error)

	// GetReKyouHistories は id の全バージョンを返します。
	GetReKyouHistories(ctx context.Context, id string) ([]ReKyou, error)

	// AddReKyouInfo は rekyou を (txID, userID, device) 付きで一時DBに追記します。
	// rekyou.TargetID（再投稿元のKyou ID）の存在確認はここでは行いません。
	// 追記専用なので、同一IDの更新は新しい UPDATE_TIME の版を足すことで表します。
	AddReKyouInfo(ctx context.Context, rekyou ReKyou, txID string, userID string, device string) error

	// GetReKyousAllLatest はターゲット解決を行わない生のReKyou一覧を返します。
	// ターゲットの存在確認やワード検索は FindReKyou / FindKyous 側で行います。
	GetReKyousAllLatest(ctx context.Context) ([]ReKyou, error)

	// GetRepositoriesWithoutReKyouRep はターゲット解決に使う「ReKyou以外のリポジトリ群」を返します。
	// 一時リポジトリは空のリポジトリ群しか持たないため、非nilで中身が空のものが返ります。
	// つまり一時リポジトリ経由ではターゲット解決はできません。
	GetRepositoriesWithoutReKyouRep(ctx context.Context) (*GkillRepositories, error)

	// GetKyousByTXID は (txID, userID, device) の一時データをKyouとして返す想定のメソッドです。
	// ただし現状のSQLite3実装は一時表に無い列（TARGET_REP_NAME）を参照するため
	// 常にエラーを返します。TX単位の取り出しには GetReKyousByTXID を使ってください。
	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]Kyou, error)

	// GetReKyousByTXID は (txID, userID, device) が一致する一時データをすべて返します。
	// ターゲット解決は行わず、TARGET_ID をそのまま載せて返します。
	// バージョンの絞り込みもしないので、同一IDに複数版があればすべて含みます。
	// 該当0件はエラーではなく空スライスです。
	GetReKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]ReKyou, error)

	// DeleteByTXID は (txID, userID, device) が一致する一時データを行ごと削除します。
	// 本リポジトリの論理削除と違って物理削除で、該当0件でもエラーになりません。
	// 再投稿元のKyouには触れません。
	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	// UnWrapTyped は ReKyouTempRepository 型のままリーフ実装へ平坦化します。
	// 契約は Repository.UnWrap を参照。
	UnWrapTyped() ([]ReKyouTempRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
