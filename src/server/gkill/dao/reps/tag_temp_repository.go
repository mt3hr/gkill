package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// TagTempRepository は、まだ確定していないTagを置いておく一時リポジトリが満たす契約です。
//
// 行は (txID, userID, device) でスコープされます。トランザクション中の書き込みは
// AddTagInfo でここに溜まり、/api/commit_tx が GetTagsByTXID で取り出して
// 本体のリポジトリへ書き写し、/api/discard_tx が DeleteByTXID で捨てます。
// 実体は利用者ごとの一時DB（既定はインメモリ）上のTAG表ひとつです。
type TagTempRepository interface {
	// FindTags の契約は TagRepository.FindTags を参照。
	//
	// ただしsqlite3実装は基底のtag実装へ構造体変換で委譲しており、変換後は
	// 一時DBのハンドルではなく "tag_temp" というファイル名でDBを開き直すため、
	// 一時表は読めずエラーになります。未確定データの取り出しには GetTagsByTXID を使ってください。
	FindTags(ctx context.Context, query *find.FindQuery) ([]Tag, error)

	// Close の契約は Repository.Close を参照。
	// 一時DBは全一時repで共有しているため閉じてはならず、sqlite3実装はDBを閉じません。
	// ただし自分でLockしたうえで同じミューテックスをLockする基底実装へ委譲するので、
	// 呼ぶとデッドロックします。呼び出し元はありません。
	Close(ctx context.Context) error

	// GetTag の契約は TagRepository.GetTag を参照。
	// FindTags と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error)

	// GetTagsByTagName の契約は TagRepository.GetTagsByTagName を参照。
	// FindTags と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetTagsByTagName(ctx context.Context, tagname string) ([]Tag, error)

	// GetTagsByTargetID の契約は TagRepository.GetTagsByTargetID を参照。
	// FindTags と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetTagsByTargetID(ctx context.Context, target_id string) ([]Tag, error)

	// UpdateCache の契約は Repository.UpdateCache を参照。
	// 一時repはキャッシュも変更検知も持たないため、sqlite3実装では何もしません。
	UpdateCache(ctx context.Context) error

	// GetLatestDataRepositoryAddress の契約は Repository.GetLatestDataRepositoryAddress を参照。
	// 未確定データは最新版の所在に含めないため、sqlite3実装は常にエラーを返します。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetPath の契約は Repository.GetPath を参照。
	// 一時repは自分のファイルを持たないため、sqlite3実装は常にエラーを返します。
	GetPath(ctx context.Context, id string) (string, error)

	// GetRepName の契約は Repository.GetRepName を参照。
	// sqlite3実装は固定値 "tag_temp" を返します。
	GetRepName(ctx context.Context) (string, error)

	// GetTagHistories の契約は TagRepository.GetTagHistories を参照。
	// FindTags と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetTagHistories(ctx context.Context, id string) ([]Tag, error)

	// AddTagInfo は tag を (txID, userID, device) 付きで一時表に追記します。
	//
	// 本体のリポジトリと同じ追記専用で、更新は同一IDの新しいUPDATE_TIME版の追加、
	// 削除は IsDeleted=true 版の追加として表現します。
	// TagRepository.AddTagInfo と違い Tag/TargetID の空チェックはせずそのままINSERTします。
	// 一意制約もないので、同じIDを何度追記してもエラーにはなりません。
	AddTagInfo(ctx context.Context, tag Tag, txID string, userID string, device string) error

	// GetAllTagNames の契約は TagRepository.GetAllTagNames を参照。
	// FindTags と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetAllTagNames(ctx context.Context) ([]string, error)

	// GetAllTags の契約は TagRepository.GetAllTags を参照。
	// FindTags と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetAllTags(ctx context.Context) ([]Tag, error)

	// GetTagsByTXID は txID・userID・device の3つすべてに一致する未確定Tagを返します。
	//
	// 一致する行が無ければ空スライスを返します（エラーではありません）。
	// RepName は一時rep名で埋めます。最新版への絞り込みも削除済みの除外もしないので、
	// そのトランザクションで追記された行がそのまま全部返ります。
	// /api/commit_tx はこれで取り出した結果を本体のリポジトリへ書き写します。
	GetTagsByTXID(ctx context.Context, txID string, userID string, device string) ([]Tag, error)

	// DeleteByTXID は (txID, userID, device) に一致する行を一時表から物理削除します。
	// 追記専用の本体リポジトリと違い、ここは実際に行を消します。
	// 一致する行が無くてもエラーにはなりません。
	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	// UnWrapTyped は UnWrap の型付き版で、TagTempRepository のまま平坦化します。
	// 一時repはラッパではないので、sqlite3実装は自分自身1件だけを返します。
	UnWrapTyped() ([]TagTempRepository, error)
}
