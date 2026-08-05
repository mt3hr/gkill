package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// NlogRepository は支出記録（nlog）のリポジトリが満たす契約です。
// Repository の共通契約に、店名（Shop）・品目（Title）・金額（Amount）を持つ Nlog 実体を
// 直接扱うメソッドを足したものです。
// Kyou 系のメソッドはメタ情報しか返さず金額を含まないので、値が要るときは Nlog 系のメソッドを使ってください。
type NlogRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	// nlog は金額をSQLiteの中に持つため、id が非空でも返るのは同じDBファイルのパス（絶対パス）です。
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

	// GetRepName の契約は Repository.GetRepName を参照。集約は固定で "NlogReps" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// FindNlog は検索条件に一致するNlogを店名・品目・金額込みで返します。
	//
	// 単一リポジトリは RelatedTime の降順で返しますが、集約はマップ経由で集めるため順序を保証しません。
	// 集約は Kyou.ID（query.OnlyLatestData が false なら ID に UpdateTime の Unix 秒を連結したキー）
	// ごとに UpdateTime が最も新しい1件だけを残します。
	// 単一リポジトリを直接呼んだ場合の query.OnlyLatestData の扱いは実装間で揃っていないため、
	// 全バージョンが返ることがあります。
	// query.UpdateCache が true のときは検索前にキャッシュを更新します。
	// キーワード検索の対象列は TITLE と SHOP です（ID も併せて照合されます）。金額は検索対象になりません。
	FindNlog(ctx context.Context, query *find.FindQuery) ([]Nlog, error)

	// GetNlog は id に対応するNlogを1件、店名・品目・金額込みで返します。
	// updateTime が nil なら最新バージョン、非nilならそのバージョンを返します。
	// 見つからない場合は (nil, nil) を返します（エラーではありません）。
	GetNlog(ctx context.Context, id string, updateTime *time.Time) (*Nlog, error)

	// GetNlogHistories は id の全バージョンを店名・品目・金額込みで返します。
	// 集約は UpdateTime の降順に整列して返します。
	GetNlogHistories(ctx context.Context, id string) ([]Nlog, error)

	// AddNlogInfo はNlogを1件追記します。
	// 追記専用なので、更新は同一IDで新しい UpdateTime の版を、削除は IsDeleted=true の版を追加します。
	// nlog.Title が空白のみのとき、nlog.Amount が空文字のとき、
	// および nlog.Amount が整数として解釈できないときはエラーです（Shop は空でも構いません）。
	// 集約（NlogRepositories）は未実装で常にエラーを返すため、書き込みは書き込み先リポジトリに対して行ってください。
	AddNlogInfo(ctx context.Context, nlog Nlog) error

	// UnWrapTyped の契約は Repository.UnWrap を参照。戻り値が NlogRepository である点だけが異なります。
	UnWrapTyped() ([]NlogRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
