package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// TextRepository はKyouに付随するテキストのリポジトリが満たす契約です。
//
// TextはKyou本体ではなくKyouに付随するデータなので、Kyouを返す Repository とは別系統の
// インターフェースになっており、FindKyous ではなく FindTexts を持ちます。
// どのKyouに付いているかは Text.TargetID が指します。
// 実装の層構成（sqlite3 / cached / temp / 集約 TextRepositories）は Repository と同じです。
type TextRepository interface {
	// FindTexts は検索条件に一致するTextを返します。
	// 契約は Repository.FindKyous を参照。戻り値はマップではなくスライスです。
	// 集約(TextRepositories)は Repository.FindKyous と同じキー規則で畳み込み、
	// query.IncludeDeletedData が false なら削除済みを除外します。
	FindTexts(ctx context.Context, query *find.FindQuery) ([]Text, error)

	// Close は契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// GetText は id に対応するTextを1件返します。
	// 契約は Repository.GetKyou を参照（見つからない場合は (nil, nil)）。
	GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error)

	// GetTextsByTargetID は target_id のKyouに付いているTextを返します。
	//
	// IDごとの最新版のみが対象です。GetTagsByTargetID と違い削除済み(IsDeleted)は
	// 除外しないので、呼び出し側で落としてください。
	// 集約はUpdateTimeの降順に並べます。
	// 1件も付いていなければ空スライスを返します（エラーではありません）。
	GetTextsByTargetID(ctx context.Context, target_id string) ([]Text, error)

	// UpdateCache は契約は Repository.UpdateCache を参照。
	UpdateCache(ctx context.Context) error
	// LastUpdateCacheChanged は直近の UpdateCache で下層の実DBに変化があったかを返します。
	// 契約は Repository.UpdateCache を参照。
	LastUpdateCacheChanged() bool

	// GetLatestDataRepositoryAddress は契約は Repository.GetLatestDataRepositoryAddress を参照。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetPath は契約は Repository.GetPath を参照。
	// Textはファイル実体を持たないため、id の有無にかかわらずTextのDBファイルのパスを返します
	// （id が非空のときだけ絶対パスに直します）。
	GetPath(ctx context.Context, id string) (string, error)

	// GetRepName は契約は Repository.GetRepName を参照。
	GetRepName(ctx context.Context) (string, error)

	// GetTextHistories は id の全バージョンを返します。
	// 契約は Repository.GetKyouHistories を参照。集約はUpdateTimeの降順に並べます。
	GetTextHistories(ctx context.Context, id string) ([]Text, error)

	// AddTextInfo はTextを1件追加します。追記専用なので、更新は同一IDの新UPDATE_TIME版、
	// 削除は IsDeleted=true 版の追加になります。
	//
	// text.Text と text.TargetID は空白のみだとエラーになります。
	// 呼び出し先は書き込み用の単一リポジトリ（WriteTextRep など）で、
	// 集約(TextRepositories)は未実装エラーを返します。
	// キャッシュ実装はキャッシュDBにだけ書くので、実DBへの書き込みとは別に呼ばれます。
	AddTextInfo(ctx context.Context, text Text) error

	// UnWrapTyped は契約は Repository.UnWrap を参照。戻り値は TextRepository のスライスです。
	UnWrapTyped() ([]TextRepository, error)
}
