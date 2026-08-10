package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// MiRepository はタスク（mi）のリポジトリが満たす契約です。
// Repository の共通契約に、タイトル・チェック状態・ボード名・期限・予定開始終了を持つ Mi 実体を
// 直接扱うメソッドを足したものです。
// 1件のタスクは作成・チェック・期限・予定開始・予定終了という5種類の時刻の切り口を持ち、
// 検索ではその切り口ごとに別のKyou（data_type=mi_create / mi_check / mi_limit / mi_start / mi_end）として現れます。
type MiRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	// 1件のタスクが最大5つのKyou（mi_create / mi_check / mi_limit / mi_start / mi_end）として返り、
	// どれを含めるかは query.IncludeCreateMi 等のフラグで決まります。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	// mi はタスクをSQLiteの中に持つため、id が非空でも返るのは同じDBファイルのパス（絶対パス）です。
	// 集約は id を含むリポジトリが1つも無いときエラーを返します。
	GetPath(ctx context.Context, id string) (string, error)

	// UpdateCache の契約は Repository.UpdateCache を参照。
	UpdateCache(ctx context.Context) error
	// LastUpdateCacheChanged は直近の UpdateCache の時点で実DBファイルに変更があったかを返します。
	// キャッシュ実装はこれが false のときフルリビルドをスキップします。
	// キャッシュ実装自身は常に true を返し、集約は配下のいずれかが true なら true を返します。
	LastUpdateCacheChanged() bool

	// GetLatestDataRepositoryAddress の契約は Repository.GetLatestDataRepositoryAddress を参照。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetRepName の契約は Repository.GetRepName を参照。集約は固定で "MiReps" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// FindMi は検索条件に一致するMiをタイトル・ボード名・各時刻込みで返します。
	//
	// query.IncludeCreateMi / IncludeCheckMi / IncludeLimitMi / IncludeStartMi / IncludeEndMi で
	// 有効にした切り口ぶんのSQLをUNIONした結果を返します。DataType にどの切り口かが入り、
	// 時刻範囲での絞り込みはその切り口の時刻（作成時刻・更新時刻・期限・予定開始・予定終了）に対して効きます。
	// 期限・予定開始・予定終了の切り口は、その時刻が入っている行だけを対象にします。
	// **切り口を1つも有効にしていない場合は空スライスを返します**（エラーではありません）。
	// query.MiBoardName が非nilのときは一致するボードだけに絞ります。
	//
	// 単一リポジトリはUNIONで組み立てるため順序を保証しません。
	// 集約は ID（query.OnlyLatestData が false なら ID に UpdateTime の Unix 秒を連結したキー）
	// ごとに UpdateTime が最も新しい1件だけを残します。
	// 同じタスクの各切り口はIDも UpdateTime も同じなので、集約では1つしか残りません。
	// 切り口ごとに扱いたいときは FindKyous を使ってください。
	// 単一リポジトリを直接呼んだ場合の query.OnlyLatestData の扱いは実装間で揃っていないため、
	// 全バージョンが返ることがあります。
	// query.UpdateCache が true のときは検索前にキャッシュを更新します。
	// キーワード検索の対象列は TITLE です。
	FindMi(ctx context.Context, query *find.FindQuery) ([]Mi, error)

	// GetMi は id に対応するMiを1件返します。
	// updateTime が nil なら最新バージョン、非nilならそのバージョンを返します。
	// 見つからない場合は (nil, nil) を返します（エラーではありません）。
	// FindMi と違い切り口ごとには分かれず、各時刻を持つ1件が返ります。
	GetMi(ctx context.Context, id string, updateTime *time.Time) (*Mi, error)

	// GetMiHistories は id の全バージョンを返します。
	// 集約は UpdateTime の降順に整列して返します。
	GetMiHistories(ctx context.Context, id string) ([]Mi, error)

	// AddMiInfo はMiを1件追記します。
	// 追記専用なので、チェックや期限の変更は同一IDで新しい UpdateTime の版を、
	// 削除は IsDeleted=true の版を追加します。
	// mi.Title が空白のみのときはエラーです。
	// 集約（MiRepositories）は未実装で常にエラーを返すため、書き込みは書き込み先リポジトリに対して行ってください。
	AddMiInfo(ctx context.Context, mi Mi) error

	// GetBoardNames は登録されているボード名の一覧を重複なしで返します。
	// 並び順は保証しません。
	// 削除済みタスクのボード名を除くのは集約だけで、単一リポジトリは表にある値をそのまま返します。
	GetBoardNames(ctx context.Context) ([]string, error)

	// UnWrapTyped の契約は Repository.UnWrap を参照。戻り値が MiRepository である点だけが異なります。
	UnWrapTyped() ([]MiRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
