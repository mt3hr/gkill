package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// TimeIsRepository は時間計測（timeis）のリポジトリが満たす契約です。
// Repository の共通契約に、タイトルと開始時刻・終了時刻を持つ TimeIs 実体を直接扱うメソッドを足したものです。
// 終了時刻（EndTime）は計測中なら nil で、計測終了は同一IDに EndTime 付きの新しい版を追記して表します。
type TimeIsRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	// 1件のtimeisは開始（data_type=timeis_start）と終了（timeis_end）の2つのKyouとして返り、
	// 終了のほうは query.IncludeEndTimeIs が true かつ終了時刻が入っているときだけ含まれます。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	// timeis は計測をSQLiteの中に持つため、id が非空でも返るのは同じDBファイルのパス（絶対パス）です。
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

	// GetRepName の契約は Repository.GetRepName を参照。集約は固定で "TimeIsReps" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// FindTimeIs は検索条件に一致するTimeIsを開始時刻・終了時刻込みで返します。
	//
	// 1件のtimeisは開始行（DataType=timeis_start）と終了行（timeis_end）に分かれて返ります。
	// 時刻範囲での絞り込みは開始行なら開始時刻、終了行なら終了時刻に対して効きます。
	// 終了行が含まれるのは query.IncludeEndTimeIs が true かつ終了時刻が入っているときだけです。
	// query.PlaingTime が非nilのときはその時刻を挟んでいる計測（終了時刻が無いものは計測中とみなす）
	// だけに絞り、あわせて最新版のみに限定します。
	//
	// 単一リポジトリはUNIONで組み立てるため順序を保証しません。
	// 集約は ID（query.OnlyLatestData が false なら ID に UpdateTime の Unix 秒を連結したキー）
	// ごとに UpdateTime が最も新しい1件だけを残します。
	// 開始行と終了行は同じIDで UpdateTime も同じなので、集約では片方しか残りません。
	// 両方が要るときは FindKyous を使ってください。
	// query.UpdateCache が true のときは検索前にキャッシュを更新します。
	// キーワード検索の対象列は TITLE です。
	FindTimeIs(ctx context.Context, query *find.FindQuery) ([]TimeIs, error)

	// GetTimeIs は id に対応するTimeIsを1件返します。
	// updateTime が nil なら最新バージョン、非nilならそのバージョンを返します。
	// 見つからない場合は (nil, nil) を返します（エラーではありません）。
	// FindTimeIs と違い開始・終了に分かれず、開始時刻と終了時刻を持つ1件が返ります。
	GetTimeIs(ctx context.Context, id string, updateTime *time.Time) (*TimeIs, error)

	// GetTimeIsHistories は id の全バージョンを返します。
	// 集約は UpdateTime の降順に整列して返します。
	GetTimeIsHistories(ctx context.Context, id string) ([]TimeIs, error)

	// AddTimeIsInfo はTimeIsを1件追記します。
	// 追記専用なので、計測の終了・更新は同一IDで新しい UpdateTime の版を、削除は IsDeleted=true の版を追加します。
	// timeis.EndTime が nil のときは終了時刻をNULLで格納します（計測中）。
	// timeis.Title が空白のみのときはエラーです。
	// 集約（TimeIsRepositories）は未実装で常にエラーを返すため、書き込みは書き込み先リポジトリに対して行ってください。
	AddTimeIsInfo(ctx context.Context, timeis TimeIs) error

	// UnWrapTyped の契約は Repository.UnWrap を参照。戻り値が TimeIsRepository である点だけが異なります。
	UnWrapTyped() ([]TimeIsRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
