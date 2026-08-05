package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// TagRepository はタグのリポジトリが満たす契約です。
//
// TagはKyou本体ではなくKyouに付随するデータなので、Kyouを返す Repository とは別系統の
// インターフェースになっており、FindKyous ではなく FindTags を持ちます。
// どのKyouに付いているかは Tag.TargetID が指します。
// 実装の層構成（sqlite3 / cached / temp / 集約 TagRepositories）は Repository と同じです。
type TagRepository interface {
	// FindTags は検索条件に一致するTagを返します。
	// 契約は Repository.FindKyous を参照。戻り値はマップではなくスライスです。
	// 集約(TagRepositories)は Repository.FindKyous と同じキー規則で畳み込み、
	// query.IncludeDeletedData が false なら削除済みを除外します。
	FindTags(ctx context.Context, query *find.FindQuery) ([]Tag, error)

	// Close は契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// GetTag は id に対応するTagを1件返します。
	// 契約は Repository.GetKyou を参照（見つからない場合は (nil, nil)）。
	GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error)

	// GetTagsByTagName はタグ名が tagname と完全一致するTagを返します。
	//
	// 編集前のタグ名でヒットしないよう、IDごとの最新版のみを対象にします。
	// 集約は削除済み(IsDeleted)を除外します。
	// 一致するものがなければ空スライスを返します（エラーではありません）。
	GetTagsByTagName(ctx context.Context, tagname string) ([]Tag, error)

	// GetTagsByTargetID は target_id のKyouに付いているTagを返します。
	//
	// IDごとの最新版のみが対象です。集約は削除済み(IsDeleted)を除外します。
	// 1件も付いていなければ空スライスを返します（エラーではありません）。
	GetTagsByTargetID(ctx context.Context, target_id string) ([]Tag, error)

	// UpdateCache は契約は Repository.UpdateCache を参照。
	UpdateCache(ctx context.Context) error
	// LastUpdateCacheChanged は直近の UpdateCache で下層の実DBに変化があったかを返します。
	// 契約は Repository.UpdateCache を参照。
	LastUpdateCacheChanged() bool

	// GetLatestDataRepositoryAddress は契約は Repository.GetLatestDataRepositoryAddress を参照。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetPath は契約は Repository.GetPath を参照。
	// Tagはファイル実体を持たないため、id の有無にかかわらずTagのDBファイルのパスを返します
	// （id が非空のときだけ絶対パスに直します）。
	GetPath(ctx context.Context, id string) (string, error)

	// GetRepName は契約は Repository.GetRepName を参照。
	GetRepName(ctx context.Context) (string, error)

	// GetTagHistories は id の全バージョンを返します。
	// 契約は Repository.GetKyouHistories を参照。集約はUpdateTimeの降順に並べます。
	GetTagHistories(ctx context.Context, id string) ([]Tag, error)

	// AddTagInfo はTagを1件追加します。追記専用なので、更新は同一IDの新UPDATE_TIME版、
	// 削除は IsDeleted=true 版の追加になります。
	//
	// tag.Tag と tag.TargetID は空白のみだとエラーになります。
	// 呼び出し先は書き込み用の単一リポジトリ（WriteTagRep など）で、
	// 集約(TagRepositories)は未実装エラーを返します。
	// キャッシュ実装はキャッシュDBにだけ書くので、実DBへの書き込みとは別に呼ばれます。
	AddTagInfo(ctx context.Context, tag Tag) error

	// GetAllTagNames はタグ名を重複なしで返します。削除済みのタグ名は含みません。
	//
	// 集約は各repの GetAllTagNames を束ねず、GetAllTags 経由でrep跨ぎにIDごとの
	// 最新版を決めてから名前を集めます。各repは自分の中の最新版しか見えないため、
	// そのまま束ねると別repに新版があるタグの編集前の名前が混ざります。
	GetAllTagNames(ctx context.Context) ([]string, error)

	// GetAllTags は全Tagを返します。IDごとの最新版のみが対象です。
	//
	// リーフ実装は削除済みも返します（rep跨ぎで最新版を決めたあとに判定するため）。
	// 集約は畳み込んだあとに削除済みを除外します。
	GetAllTags(ctx context.Context) ([]Tag, error)

	// UnWrapTyped は契約は Repository.UnWrap を参照。戻り値は TagRepository のスライスです。
	UnWrapTyped() ([]TagRepository, error)
}
