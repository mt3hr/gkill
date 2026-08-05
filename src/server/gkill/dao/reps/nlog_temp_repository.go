package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// NlogTempRepository は、まだ確定していないNlog（支出記録）を置いておく一時リポジトリが満たす契約です。
//
// 行は (txID, userID, device) でスコープされます。トランザクション中の書き込みは
// AddNlogInfo でここに溜まり、/api/commit_tx が GetNlogsByTXID で取り出して
// 本体のリポジトリへ書き写し、/api/discard_tx が DeleteByTXID で捨てます。
// 実体は利用者ごとの一時DB（既定はインメモリ）上のNLOG表ひとつです。
type NlogTempRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	//
	// ただしsqlite3実装は基底のnlog実装へ構造体変換で委譲しており、変換後は
	// 一時DBのハンドルではなく "nlog_temp" というファイル名でDBを開き直すため、
	// 一時表は読めずエラーになります。未確定データの取り出しには GetNlogsByTXID を使ってください。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	// FindKyous と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	// FindKyous と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	// 一時repは自分のファイルを持たないため、sqlite3実装は常にエラーを返します。
	GetPath(ctx context.Context, id string) (string, error)

	// UpdateCache の契約は Repository.UpdateCache を参照。
	// 一時repはキャッシュも変更検知も持たないため、sqlite3実装では何もしません。
	UpdateCache(ctx context.Context) error

	// GetLatestDataRepositoryAddress の契約は Repository.GetLatestDataRepositoryAddress を参照。
	// 未確定データは最新版の所在に含めないため、sqlite3実装は常にエラーを返します。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetRepName の契約は Repository.GetRepName を参照。
	// sqlite3実装は固定値 "nlog_temp" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	// 一時DBは全一時repで共有しているため、sqlite3実装はDBを閉じません。
	Close(ctx context.Context) error

	// FindNlog は検索条件に一致するNlogを店舗名・品目・金額込みで返します。
	// FindKyous と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	FindNlog(ctx context.Context, query *find.FindQuery) ([]Nlog, error)

	// GetNlog は id に対応するNlogを1件返します。updateTime が nil なら最新版です。
	// FindKyous と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetNlog(ctx context.Context, id string, updateTime *time.Time) (*Nlog, error)

	// GetNlogHistories は id の全バージョンを返します。
	// FindKyous と同じ理由で、sqlite3実装は一時表を読めずエラーになります。
	GetNlogHistories(ctx context.Context, id string) ([]Nlog, error)

	// AddNlogInfo は nlog を (txID, userID, device) 付きで一時表に追記します。
	//
	// 本体のリポジトリと同じ追記専用で、更新は同一IDの新しいUPDATE_TIME版の追加、
	// 削除は IsDeleted=true 版の追加として表現します。
	// Amount は文字列として保存しますが、読み出す GetNlogsByTXID が整数として解釈するので、
	// 小数を含む金額を渡すと取り出し時にエラーになります。
	// 一意制約はないので、同じIDを何度追記してもエラーにはなりません。
	AddNlogInfo(ctx context.Context, nlog Nlog, txID string, userID string, device string) error

	// GetKyousByTXID は未確定NlogをKyouとして返す想定のメソッドです。
	//
	// ただしsqlite3実装のSQLが一時表に無い TARGET_REP_NAME 列を参照しているため、
	// 現状は必ずエラーになります。コミット処理は GetNlogsByTXID を使っており、
	// 本メソッドの呼び出し元はありません。
	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]Kyou, error)

	// GetNlogsByTXID は txID・userID・device の3つすべてに一致する未確定Nlogを返します。
	//
	// 一致する行が無ければ空スライスを返します（エラーではありません）。
	// Amount は整数として読み出すため、小数を含む金額が入っているとエラーになります。
	// RepName は一時rep名、DataType は "nlog" で埋めます。
	// /api/commit_tx はこれで取り出した結果を本体のリポジトリへ書き写します。
	GetNlogsByTXID(ctx context.Context, txID string, userID string, device string) ([]Nlog, error)

	// DeleteByTXID は (txID, userID, device) に一致する行を一時表から物理削除します。
	// 追記専用の本体リポジトリと違い、ここは実際に行を消します。
	// 一致する行が無くてもエラーにはなりません。
	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	// UnWrapTyped は UnWrap の型付き版で、NlogTempRepository のまま平坦化します。
	// 一時repはラッパではないので、sqlite3実装は自分自身1件だけを返します。
	UnWrapTyped() ([]NlogTempRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	// 一時repはラッパではないので、sqlite3実装は自分自身1件だけを返します。
	UnWrap() ([]Repository, error)
}
