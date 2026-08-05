package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// LantanaRepository は気分値（lantana）のリポジトリが満たす契約です。
// Repository の共通契約に、気分値（Mood）を持つ Lantana 実体を直接扱うメソッドを足したものです。
// 保持するのは数値だけでテキスト列を持たないため、キーワード検索は常に0件になります。
type LantanaRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	// lantana は検索対象のテキスト列を持たないため、query.UseWords が true のクエリでは常に0件です。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	// lantana は気分値をSQLiteの中に持つため、id が非空でも返るのは同じDBファイルのパス（絶対パス）です。
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

	// GetRepName の契約は Repository.GetRepName を参照。集約は固定で "LantanaReps" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// FindLantana は検索条件に一致するLantanaを気分値込みで返します。
	//
	// 単一リポジトリは RelatedTime の降順で返しますが、集約はマップ経由で集めるため順序を保証しません。
	// 集約は Kyou.ID（query.OnlyLatestData が false なら ID に UpdateTime の Unix 秒を連結したキー）
	// ごとに UpdateTime が最も新しい1件だけを残します。
	// query.UpdateCache が true のときは検索前にキャッシュを更新します。
	// FindKyous と同じく、query.UseWords が true のクエリでは常に0件です。
	FindLantana(ctx context.Context, query *find.FindQuery) ([]Lantana, error)

	// GetLantana は id に対応するLantanaを1件、気分値込みで返します。
	// updateTime が nil なら最新バージョン、非nilならそのバージョンを返します。
	// 見つからない場合は (nil, nil) を返します（エラーではありません）。
	GetLantana(ctx context.Context, id string, updateTime *time.Time) (*Lantana, error)

	// GetLantanaHistories は id の全バージョンを気分値込みで返します。
	// 集約は UpdateTime の降順に整列して返します。
	GetLantanaHistories(ctx context.Context, id string) ([]Lantana, error)

	// AddLantanaInfo はLantanaを1件追記します。
	// 追記専用なので、更新は同一IDで新しい UpdateTime の版を、削除は IsDeleted=true の版を追加します。
	// Mood の範囲検証は行わないため、値の妥当性は呼び出し側の責任です。
	// 集約（LantanaRepositories）は未実装で常にエラーを返すため、書き込みは書き込み先リポジトリに対して行ってください。
	AddLantanaInfo(ctx context.Context, lantana Lantana) error

	// UnWrapTyped の契約は Repository.UnWrap を参照。戻り値が LantanaRepository である点だけが異なります。
	UnWrapTyped() ([]LantanaRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
