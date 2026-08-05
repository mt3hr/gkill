package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// TextTempRepository は、まだ確定していないTextを置いておく一時リポジトリが満たす契約です。
//
// 行は (txID, userID, device) でスコープされます。トランザクション中の書き込みは
// AddTextInfo でここに溜まり、/api/commit_tx が GetTextsByTXID で取り出して
// 本体のリポジトリへ書き写し、/api/discard_tx が DeleteByTXID で捨てます。
// 実体は利用者ごとの一時DB（既定はインメモリ）上のTEXT表ひとつです。
type TextTempRepository interface {
	// FindTexts の契約は TextRepository.FindTexts を参照。
	//
	// ただしsqlite3実装は基底のtext実装へ構造体変換で委譲しており、変換後は
	// 一時DBのハンドルではなく "text_temp" というファイル名でDBを開き直すため、
	// 一時表は読めずエラーになります。未確定データの取り出しには GetTextsByTXID を使ってください。
	FindTexts(ctx context.Context, query *find.FindQuery) ([]Text, error)

	// Close の契約は Repository.Close を参照。
	// 一時DBは全一時repで共有しているため閉じてはならず、sqlite3実装はDBを閉じません。
	// ただし自分でLockしたうえで同じミューテックスをLockする基底実装へ委譲するので、
	// 呼ぶとデッドロックします。呼び出し元はありません。
	Close(ctx context.Context) error

	// GetText の契約は TextRepository.GetText を参照。
	// FindTexts と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetText(ctx context.Context, id string, updateTime *time.Time) (*Text, error)

	// GetTextsByTargetID の契約は TextRepository.GetTextsByTargetID を参照。
	// FindTexts と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetTextsByTargetID(ctx context.Context, target_id string) ([]Text, error)

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
	// sqlite3実装は固定値 "text_temp" を返します。
	GetRepName(ctx context.Context) (string, error)

	// GetTextHistories の契約は TextRepository.GetTextHistories を参照。
	// FindTexts と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetTextHistories(ctx context.Context, id string) ([]Text, error)

	// AddTextInfo は text を (txID, userID, device) 付きで一時表に追記します。
	//
	// 本体のリポジトリと同じ追記専用で、更新は同一IDの新しいUPDATE_TIME版の追加、
	// 削除は IsDeleted=true 版の追加として表現します。
	// TextRepository.AddTextInfo と違い Text/TargetID の空チェックはせずそのままINSERTします。
	// 一意制約もないので、同じIDを何度追記してもエラーにはなりません。
	AddTextInfo(ctx context.Context, text Text, txID string, userID string, device string) error

	// GetTextsByTXID は txID・userID・device の3つすべてに一致する未確定Textを返します。
	//
	// 一致する行が無ければ空スライスを返します（エラーではありません）。
	// RepName は一時rep名で埋めます。最新版への絞り込みも削除済みの除外もしないので、
	// そのトランザクションで追記された行がそのまま全部返ります。
	// /api/commit_tx はこれで取り出した結果を本体のリポジトリへ書き写します。
	GetTextsByTXID(ctx context.Context, txID string, userID string, device string) ([]Text, error)

	// DeleteByTXID は (txID, userID, device) に一致する行を一時表から物理削除します。
	// 追記専用の本体リポジトリと違い、ここは実際に行を消します。
	// 一致する行が無くてもエラーにはなりません。
	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	// UnWrapTyped は UnWrap の型付き版で、TextTempRepository のまま平坦化します。
	// 一時repはラッパではないので、sqlite3実装は自分自身1件だけを返します。
	UnWrapTyped() ([]TextTempRepository, error)
}
