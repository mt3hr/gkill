package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// Repository は全データ型のリポジトリが満たす共通契約です。
//
// 「更新も削除も INSERT」を選んだ理由と却下案:
// documents/adr/0010-append-only-dao.md
//
// 実装は4層あります（SQLiteを持つデータ型の場合）:
//
//	*_repository.go              interface定義（この契約の型別拡張）
//	*_repository_sqlite3_impl.go 実DBを読む素の実装（リーフ）
//	*_repository_cached_*.go     キャッシュDBを読むラッパ
//	*_repository_temp_*.go       トランザクション用の一時置き場
//
// これに加えて、複数リポジトリを束ねる集約型（XxxRepositories）が同じ契約を実装します。
// 集約は各リポジトリを threads.Go で並列に呼ぶため、**リポジトリ実装の中から集約を呼ぶときは
// 必ず逐次版（XxxSequential）を使ってください**。並列版をネストすると
// threads.Go の有界セマフォが枯渇して恒久ハングします。
//
// git_commit_log（local_dir実装）・gps_log（gpx_dir実装）・plugin（サブプロセス実装）は
// 外部ソースを直接読むため4層すべてを持ちません。
type Repository interface {
	// FindKyous は検索条件に一致するKyouを返します。
	//
	// 戻り値のキーは、単一リポジトリでは Kyou.ID で、値はその ID の全バージョンです。
	// 集約（Repositories）では query.OnlyLatestData が false のとき
	// ID に UpdateTime の Unix 秒を連結したキーになり、バージョンごとに別エントリになります。
	//
	// 並び順は保証しません。最終的な順序は呼び出し側（api.FindFilter）が決めます。
	//
	// query.UpdateCache が true のときは検索前にキャッシュを更新します。
	// 集約実装は並列dispatchの前に UpdateCache を逐次実行し、
	// クローンした query（UpdateCache=false）を各リポジトリへ配ります。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou は id に対応するKyouを1件返します。
	//
	// updateTime が nil なら最新バージョン、非nilならそのバージョンを返します。
	// **見つからない場合は (nil, nil) を返します**（エラーではありません）。
	// 呼び出し側は戻り値の nil チェックで存在判定してください。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories は id の全バージョンを返します。
	// gkill は追記専用（Append-Only）なので、更新のたびにバージョンが増えます。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath は id が空文字ならリポジトリ自身のファイルパス（多くはDBファイル）を、
	// id が非空ならそのデータの実体パスを返します。
	// IDFKyou のようにファイルを指すデータ型では後者が対象ファイルの絶対パスになります。
	GetPath(ctx context.Context, id string) (string, error)

	// GetRepName はこのリポジトリの表示名を返します。
	// 名前は利用者間で一意ではありません（同名の別ユーザのリポジトリが存在しえます）。
	GetRepName(ctx context.Context) (string, error)

	// UpdateCache はキャッシュを最新の状態にします。
	//
	// キャッシュ付き実装では、下層の LastUpdateCacheChanged が false のときは
	// フルリビルドをスキップします。素のsqlite3実装はキャッシュを持ちませんが、
	// DBファイルの更新時刻とサイズを観測して LastUpdateCacheChanged の判定材料を更新します。
	UpdateCache(ctx context.Context) error

	// GetLatestDataRepositoryAddress は「どのIDの最新版がどのリポジトリにあるか」の一覧を返します。
	// ReKyou/MiReKyou のターゲット解決と、最新版所在キャッシュの構築に使います。
	// updateCache が true のときは収集し直します。
	GetLatestDataRepositoryAddress(ctx context.Context, updateCache bool) ([]gkill_cache.LatestDataRepositoryAddress, error)

	// Close はDB接続などのリソースを解放します。
	Close(ctx context.Context) error

	// UnWrap は集約・キャッシュのラッパを剥がして、実データを持つリーフ実装の一覧を返します。
	// リポジトリ名でのフィルタや、IDFの実ファイル配信で具象実装を列挙するのに使います。
	UnWrap() ([]Repository, error)
}
