package reps

import (
	"context"
	"net/http"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// IDFKyouRepository はファイル（idf_kyou）のリポジトリが満たす契約です。
// Repository の共通契約に、対象ファイルを指す IDFKyou 実体を直接扱うメソッドと、
// ファイル配信・派生キャッシュ（サムネイル / 互換動画 / ZIP展開）の操作を足したものです。
//
// 他のデータ型と違い、実体はDBの中ではなく監視対象ディレクトリ上のファイルです。
// DBが持つのは対象ファイルの相対パスとID・時刻だけで、
// IsImage / IsVideo / IsAudio / IsZip・FileURL・ContentPath は読み出しのたびにファイル名から組み立てます。
type IDFKyouRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	// id が非空のときは監視対象ディレクトリと対象ファイルの相対パスを繋いだ、実ファイルのパスを返します。
	// **id に対応するデータが無いときはエラーを返します**（GetKyou と違い (nil, nil) 相当にはなりません）。
	// 集約は id を含むリポジトリが1つも無いときエラーを返します。
	GetPath(ctx context.Context, id string) (string, error)

	// UpdateCache の契約は Repository.UpdateCache を参照。
	// 自動IDFが有効なリポジトリでは、ここで IDF が走って未登録ファイルを取り込みます。
	UpdateCache(ctx context.Context) error
	// LastUpdateCacheChanged は直近の UpdateCache の時点で下層に変更があったかを返します。
	// キャッシュ実装はこれが false のときフルリビルドをスキップします。
	// ただしIDFは実DBファイルの更新を観測しておらず、監視対象ディレクトリの中身も変わりうるため、
	// 素の実装もキャッシュ実装も常に true を返します（リポジトリをローカルにキャッシュする実装だけが例外）。
	LastUpdateCacheChanged() bool

	// GetLatestDataRepositoryAddress の契約は Repository.GetLatestDataRepositoryAddress を参照。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// GetRepName の契約は Repository.GetRepName を参照。
	// 単一リポジトリでは監視対象ディレクトリ名がそのままリポジトリ名になり、集約は固定で "IDFKyouReps" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// FindIDFKyou は検索条件に一致するIDFKyouを、対象ファイルの情報込みで返します。
	//
	// 単一リポジトリは RelatedTime の降順で返しますが、集約はマップ経由で集めるため順序を保証しません。
	// 集約は ID（query.OnlyLatestData が false なら ID に UpdateTime の Unix 秒を連結したキー）
	// ごとに UpdateTime が最も新しい1件だけを残します。
	// query.UpdateCache が true のときは検索前にキャッシュを更新します（自動IDFなら取り込みも走ります）。
	//
	// キーワード検索だけはSQLではなくGo側で突き合わせます。
	// 対象は実ファイルのパス文字列と、拡張子が .md / .txt のときはそのファイルの中身です。
	// そのため該当ファイルが読めないとエラーになります。
	FindIDFKyou(ctx context.Context, query *find.FindQuery) ([]IDFKyou, error)

	// GetIDFKyou は id に対応するIDFKyouを1件返します。
	// updateTime が nil なら最新バージョン、非nilならそのバージョンを返します。
	// 見つからない場合は (nil, nil) を返します（エラーではありません）。
	GetIDFKyou(ctx context.Context, id string, updateTime *time.Time) (*IDFKyou, error)

	// GetIDFKyouByTargetFile は監視対象ディレクトリからの相対パスでIDFKyouを逆引きします。
	//
	// 対象ファイルはOS依存の区切り文字で格納されている可能性があるため、"/" 区切りと "\" 区切りの
	// 両方で照合します。返るのは最新バージョンで、それが削除済みの場合は対象外です。
	// 見つからない場合は (nil, nil) を返します（エラーではありません）。
	GetIDFKyouByTargetFile(ctx context.Context, targetFile string) (*IDFKyou, error)

	// GetIDFKyouHistories は id の全バージョンを返します。
	// 集約は UpdateTime の降順に整列して返します。
	GetIDFKyouHistories(ctx context.Context, id string) ([]IDFKyou, error)

	// IDF は監視対象ディレクトリを走査し、まだ登録されていないファイルをIDFKyouとして取り込みます。
	//
	// 登録済みの対象ファイルは飛ばすので、何度呼んでも結果は変わりません。
	// 除外指定に載っているファイル名は取り込みません。
	// ファイル名がDUID形式（時刻_UUID）ならそこからIDと関連時刻を取り出し、
	// そうでなければ新しいIDを振ってファイルの更新時刻を関連時刻にします。
	// 集約（IDFKyouRepositories）は未実装で常にエラーを返します。
	IDF(ctx context.Context) error

	// AddIDFKyouInfo はIDFKyouを1件追記します。
	// 追記専用なので、更新は同一IDで新しい UpdateTime の版を、削除は IsDeleted=true の版を追加します。
	// idfKyou.RepName がこのリポジトリ自身のときは対象リポジトリ名を空で格納します。
	// これはフォルダ名を変えても（DVNF後も）対象ファイルを解決できるようにするためです。
	// 集約（IDFKyouRepositories）は未実装で常にエラーを返すため、書き込みは書き込み先リポジトリに対して行ってください。
	AddIDFKyouInfo(ctx context.Context, idfKyou IDFKyou) error

	// HandleFileServe は /files/{リポジトリ名}/ 以下のリクエストに対して対象ファイルを配信します。
	// 呼び出し側がリポジトリ名までのプレフィックスを取り除いてから渡します。
	//
	// thumb クエリがあればサムネイル（動画ならポスター画像）を、無ければ互換動画キャッシュ経由で本体を返します。
	// ディレクトリそのものへのアクセスは404にします。
	// 集約（IDFKyouRepositories）は配信先を特定できないため常に404を返します。
	HandleFileServe(w http.ResponseWriter, r *http.Request)

	// GenerateThumbCache は配下の画像・動画のサムネイルキャッシュをまとめて生成します。CLIからの事前生成用です。
	// 1件ごとの失敗はログに出して読み飛ばすので、途中で失敗してもエラーは返しません。
	GenerateThumbCache(ctx context.Context) error

	// ClearThumbCache は userID のサムネイルキャッシュディレクトリを削除します。
	// リポジトリ名は利用者間で一意ではないため、生成側と同じく利用者IDで階層を分けて消します。
	// 削除に失敗しても単一リポジトリはエラーを返しません。
	ClearThumbCache(userID string) error

	// GenerateVideoCache は配下の動画のブラウザ互換動画キャッシュ（HEVC→H.264 MP4など）をまとめて生成します。
	// CLIからの事前生成用です。
	// 互換動画の生成器が用意されていないリポジトリでは何もしません。
	// 1件ごとの失敗はログに出して読み飛ばすので、途中で失敗してもエラーは返しません。
	GenerateVideoCache(ctx context.Context) error

	// ClearVideoCache は userID の互換動画キャッシュディレクトリを削除します。
	// 削除の契約は ClearThumbCache と同じです。
	ClearVideoCache(userID string) error

	// ClearZipCache は userID のZIP展開キャッシュディレクトリを削除します。
	// 削除の契約は ClearThumbCache と同じです。
	ClearZipCache(userID string) error

	// UnWrapTyped の契約は Repository.UnWrap を参照。戻り値が IDFKyouRepository である点だけが異なります。
	// ファイル配信はリーフ実装しか行えないため、配信先リポジトリの特定にはこれで平坦化してから名前で突き合わせます。
	UnWrapTyped() ([]IDFKyouRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
