package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// ReKyouRepository はReKyou（既存Kyouのリポスト）のリポジトリが満たす契約です。
//
// ReKyouはタイトルも本文も持たず TargetID の指すKyouを参照するだけなので、
// 検索のたびに他のリポジトリ群を辿ってターゲットを解決し、ワード検索もターゲットへ委譲します。
// その委譲先からReKyou自身を外すのが GetRepositoriesWithoutReKyouRep です。
type ReKyouRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	//
	// ReKyou固有の絞り込みとして、ターゲットの最新版が見つからないもの・
	// ターゲットが削除済みのものは結果から落とします。
	// ワード条件はReKyou自身では判定できないためターゲットKyouへ委譲します。
	// 時刻やタグの絞り込みはここでは行いません（呼び出し側の api.FindFilter が絞ります）。
	//
	// GetRepositoriesWithoutReKyouRep がnilを返す構成ではターゲット解決を行わず、
	// 未削除のReKyouをすべて通します。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	//
	// ReKyouはファイル実体を持たないため、idを指定してもデータごとのパスにはならず
	// リポジトリ自身を指す値を返します。
	// 集約実装はidを持つリポジトリを探して最初に見つかったものの値を返すので、
	// id空文字で「リポジトリ自身」を取ることはできません（見つからずエラーになります）。
	GetPath(ctx context.Context, id string) (string, error)

	// UpdateCache の契約は Repository.UpdateCache を参照。
	UpdateCache(ctx context.Context) error
	// LastUpdateCacheChanged は直近のUpdateCacheで下層に変更があったかを返します。
	// キャッシュ実装はこれがfalseならフルリビルドをスキップします。
	// 変更を実際に追跡するのはローカルキャッシュ実装だけで、
	// 素のsqlite3実装とキャッシュ実装は常にtrueを返します。
	LastUpdateCacheChanged() bool

	// GetLatestDataRepositoryAddress の契約は Repository.GetLatestDataRepositoryAddress を参照。
	// TargetIDにはReKyou自身のIDが入ります（リポスト対象のIDは載せません）。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetRepName の契約は Repository.GetRepName を参照。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// FindReKyou は検索条件に一致するReKyouを返します。
	//
	// 未削除の最新ReKyouのうち、ターゲットの最新版アドレスが存在するものだけを返します。
	// FindKyousと違い、ワード条件のターゲットへの委譲は行わず、
	// ターゲットが削除済みかどうかも見ません（存在だけを見ます）。
	// 並び順は保証しません。
	FindReKyou(ctx context.Context, query *find.FindQuery) ([]ReKyou, error)

	// GetReKyou は id に対応するReKyouを1件返します。
	// updateTime が nil なら最新バージョン、非nilならそのバージョンを返します。
	// 見つからない場合は (nil, nil) を返します（エラーではありません）。
	GetReKyou(ctx context.Context, id string, updateTime *time.Time) (*ReKyou, error)

	// GetReKyouHistories は id の全バージョンを返します。
	// 集約実装は UpdateTime の降順で返します。
	GetReKyouHistories(ctx context.Context, id string) ([]ReKyou, error)

	// AddReKyouInfo はReKyouを1件追記します。
	// 追記専用なので、更新は同一IDで新しいUPDATE_TIMEの版を、
	// 削除は IsDeleted=true の版を追加することで表します。
	// 集約（ReKyouRepositories）は未実装でエラーを返すため、書き込み先のリポジトリを選んで呼んでください。
	AddReKyouInfo(ctx context.Context, rekyou ReKyou) error

	// GetReKyousAllLatest はターゲット解決を行わない生のReKyouを返します。
	//
	// 削除済み（IsDeleted=true）も含みます。除外は呼び出し側の責務です。
	// 単一リポジトリ実装は版の畳み込みをしないため同一IDの複数バージョンが返ることがあり、
	// 集約実装はID単位で UpdateTime が最大のものだけに畳んで RelatedTime の降順で返します。
	GetReKyousAllLatest(ctx context.Context) ([]ReKyou, error)

	// GetRepositoriesWithoutReKyouRep はReKyou自身を除いたリポジトリ群のクローンを返します。
	//
	// ターゲット解決でReKyou→ReKyou→…の無限再帰が起きないようにするためのものです。
	// MiReKyouは含めます（ReKyouはMiReKyouをターゲットにできるため、
	// 外すとそのReKyouがワード検索から漏れます）。
	// リポジトリ群を持たない構成（TX中の一時リポジトリなど）では (nil, nil) を返します。
	// エラーではないので、呼び出し側はnilならターゲット解決を行わず全件通してください。
	GetRepositoriesWithoutReKyouRep(ctx context.Context) (*GkillRepositories, error)

	// UnWrapTyped の契約は Repository.UnWrap を参照。ReKyouRepository型で返す版です。
	UnWrapTyped() ([]ReKyouRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
