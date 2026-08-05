package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// MiReKyouRepository はMiReKyou（既存Kyouをタスク化したもの）のリポジトリが満たす契約です。
//
// MiReKyouはタイトルを持たず TargetID の指すKyouを表示に使うため、
// 検索のたびに他のリポジトリ群を辿ってターゲットを解決し、ワード検索もターゲットへ委譲します。
// その委譲先からMiReKyou自身を外すのが GetRepositoriesWithoutMiReKyouRep です。
//
// Miと同じく1件のMiReKyouを作成/チェック/期限/開始/終了の5つの時刻へ射影するため、
// DataTypeは mirekyou_create / _check / _limit / _start / _end のいずれかになります。
// 接頭辞で判定するときは "mi" より先に "mirekyou" を見てください。
type MiReKyouRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	//
	// 1件のMiReKyouが最大5件のKyou（5射影）になります。
	// どの射影を含めるかは query.IncludeCreateMi 等で決まり、
	// 対応する時刻がNULLの射影は出ません。
	// 「最新版のみ」が効くのは作成射影だけで、他の射影は常に最新版だけを返します。
	//
	// MiReKyou固有の絞り込みとして、ターゲットが解決できないものは結果から落とし、
	// ワード条件はMiReKyou自身では判定できないためターゲットKyouへ委譲します。
	// GetRepositoriesWithoutMiReKyouRep がnilを返す構成ではターゲット解決を行わず全件通します。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	// 5射影のうち最初に見つかった1件を返すため、DataTypeがどの射影になるかは決まっていません。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	// 5射影ぶんすべてを版ごとに返すので、1バージョンにつき最大5件になります。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	//
	// MiReKyouはファイル実体を持たないため、idを指定してもデータごとのパスにはならず
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
	// TargetIDにはMiReKyou自身のIDが入り、リポスト対象のIDは TargetIDInData に入ります。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetRepName の契約は Repository.GetRepName を参照。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// FindMiReKyou は検索条件に一致するMiReKyouを返します。
	//
	// FindKyousのMiReKyou版で、射影の選択・ターゲット解決・ワード委譲の扱いは同じです。
	// 同じMiReKyouが射影の数だけDataType違いで返ります。
	// 並び順は保証しません。
	FindMiReKyou(ctx context.Context, query *find.FindQuery) ([]MiReKyou, error)

	// GetMiReKyou は id に対応するMiReKyouを1件返します。
	// updateTime が nil なら最新バージョン、非nilならそのバージョンを返します。
	// 見つからない場合は (nil, nil) を返します（エラーではありません）。
	// 作成射影だけを使うためDataTypeは常に "mirekyou_create" です。
	GetMiReKyou(ctx context.Context, id string, updateTime *time.Time) (*MiReKyou, error)

	// GetMiReKyouHistories は id の全バージョンを返します。
	// 作成射影だけを使うので1バージョンにつき1件です。集約実装は UpdateTime の降順で返します。
	GetMiReKyouHistories(ctx context.Context, id string) ([]MiReKyou, error)

	// AddMiReKyouInfo はMiReKyouを1件追記します。
	//
	// 追記専用なので、更新は同一IDで新しいUPDATE_TIMEの版を、
	// 削除は IsDeleted=true の版を追加することで表します。
	// TargetIDが空（空白のみを含む）ならエラーにします。ターゲットのないMiReKyouは表示できないためです。
	// 集約（MiReKyouRepositories）は未実装でエラーを返すため、書き込み先のリポジトリを選んで呼んでください。
	AddMiReKyouInfo(ctx context.Context, mirekyou MiReKyou) error

	// GetMiReKyousAllLatest はターゲット解決を行わない生のMiReKyou（ID毎の最新版）を返します。
	// 削除済み（IsDeleted=true）も含みます。除外は呼び出し側の責務です。
	// 集約実装はID単位で UpdateTime が最大のものへ畳み、CreateTime の降順で返します。
	GetMiReKyousAllLatest(ctx context.Context) ([]MiReKyou, error)

	// GetBoardNames はMiReKyouが属するボード名の一覧を返します。
	//
	// 順序は保証せず、ボード名を設定していないMiReKyouぶんの空文字も含みます。
	// 素のsqlite3実装は表のDISTINCTなので削除済みや古い版のボード名も混ざり、
	// 集約実装は未削除の最新版だけから集めます。
	GetBoardNames(ctx context.Context) ([]string, error)

	// GetRepositoriesWithoutMiReKyouRep はMiReKyou自身を除いたリポジトリ群のクローンを返します。
	//
	// ターゲット解決の無限再帰を防ぐためのものです。ReKyouも含めません
	// （ReKyou側はMiReKyouを含むので、両方向に含めると相互再帰します。この非対称は意図的です）。
	// リポジトリ群を持たない構成（TX中の一時リポジトリなど）では (nil, nil) を返します。
	// エラーではないので、呼び出し側はnilならターゲット解決を行わず全件通してください。
	GetRepositoriesWithoutMiReKyouRep(ctx context.Context) (*GkillRepositories, error)

	// UnWrapTyped の契約は Repository.UnWrap を参照。MiReKyouRepository型で返す版です。
	UnWrapTyped() ([]MiReKyouRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
