package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// GitCommitLogRepository はgitコミットログのリポジトリが満たす契約です。
//
// 実体はローカルのgitリポジトリで、gkillからの書き込み口はありません（Add*Infoがありません）。
// コミットは不変なのでバージョンは常に1つだけで、IsDeletedは常にfalse、
// RelatedTime / CreateTime / UpdateTime はいずれもコミッタ日時、DataTypeは "git_commit_log" 固定です。
//
// キャッシュ実装はキャッシュ構築中だけ下層のgitリポジトリへ読みを逃がします。
// 集約（GitCommitLogRepositories）にはそのための逐次版（XxxSequential）があるので、
// threads.Goのスロットを保持したまま呼ぶときはそちらを使ってください。
type GitCommitLogRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	// git logを走査してKyouを組み立てます。
	// 削除済みコミットは存在しないため、query.IsDeleted が true なら必ず0件になります。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。id はコミットハッシュです。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	// コミットは不変なので、見つかれば1件、見つからなければ0件です。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	// コミットは個別のファイルを持たないため、idの有無にかかわらずgitリポジトリのパスを返します。
	GetPath(ctx context.Context, id string) (string, error)

	// UpdateCache の契約は Repository.UpdateCache を参照。
	// 素のgit実装は全refのハッシュを前回と突き合わせ、LastUpdateCacheChanged の判定材料を更新します。
	// キャッシュ実装は変更があったコミットだけを差分でキャッシュDBへ取り込みます。
	UpdateCache(ctx context.Context) error
	// LastUpdateCacheChanged は直近のUpdateCacheでrefが動いたかを返します。
	// キャッシュ実装はこれがfalseなら取り込みをスキップします。
	LastUpdateCacheChanged() bool

	// GetLatestDataRepositoryAddress の契約は Repository.GetLatestDataRepositoryAddress を参照。
	// TargetIDはコミットハッシュ、DataUpdateTimeはコミッタ日時で、IsDeletedは常にfalseです。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetRepName の契約は Repository.GetRepName を参照。gitリポジトリのディレクトリ名になります。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// FindGitCommitLog は検索条件に一致するコミットを、追加・削除行数込みで返します。
	//
	// FindKyousと違って行数集計（go-gitのStatsContext）を伴うぶん重く、
	// 素のgit実装は集計をワーカープールで並列に回します。
	// 並び順は保証しません。1件も一致しなければ (nil, nil) を返します。
	FindGitCommitLog(ctx context.Context, query *find.FindQuery) ([]GitCommitLog, error)

	// FindGitCommitLogByIDs は指定したコミットハッシュのぶんだけを行数込みで返します。
	//
	// キャッシュの差分更新（増えたコミットだけを取り込む）用です。
	// ids が空なら (nil, nil) を返します。
	// 見つからなかったidは黙って落ちるだけでエラーにはならず、並び順も保証しません。
	FindGitCommitLogByIDs(ctx context.Context, ids []string) ([]GitCommitLog, error)

	// GetGitCommitLog はコミットハッシュ id のGitCommitLogを1件返します。
	// updateTime を指定するとコミッタ日時が一致するものだけに絞ります。
	// 見つからない場合は (nil, nil) を返します（エラーではありません）。
	GetGitCommitLog(ctx context.Context, id string, updateTime *time.Time) (*GitCommitLog, error)

	// UnWrapTyped の契約は Repository.UnWrap を参照。GitCommitLogRepository型で返す版です。
	UnWrapTyped() ([]GitCommitLogRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
