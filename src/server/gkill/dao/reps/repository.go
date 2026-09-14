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
// documents/adr/0201-append-only-dao.md
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

// RepNamesProvider は、1つのリーフ実装が複数の rep 名を名乗るリポジトリが満たす任意の契約です。
//
// プラグイン1本が複数の Git リポジトリを代表するとき、Kyou.RepName はリポジトリごとに
// 別の名前になります。GetRepName が返す1つの名前だけを見て名前を列挙・照合すると、
// サイドバーの rep 一覧に載らず、rep 名の絞り込み（find_filter.go の Step4）で検索対象からも
// 外れて、エラーも警告も無いまま0件になります。名前を**列挙する側**は GetRepName ではなく
// RepNamesOf を使ってください。GetRepName は引き続き「この実装を代表する1つの名前」で、
// MatchReps のキーやログにはそちらを使います。
type RepNamesProvider interface {
	// GetRepNames はこの実装の記録が名乗る rep 名の全集合を返します。
	// 空スライスは「いまは名乗る名前が無い」で、その間は rep 名の絞り込みで選ばれません。
	GetRepNames(ctx context.Context) ([]string, error)
}

// RepNamesOf は rep が名乗る rep 名の全集合を返します。
// RepNamesProvider を実装していれば GetRepNames を、そうでなければ GetRepName の1つを返します。
// GetAllRepNames と find_filter.go の rep 名照合はこれを通し、GetRepName を直接見ないこと。
func RepNamesOf(ctx context.Context, rep Repository) ([]string, error) {
	if provider, ok := rep.(RepNamesProvider); ok {
		return provider.GetRepNames(ctx)
	}
	repName, err := rep.GetRepName(ctx)
	if err != nil {
		return nil, err
	}
	return []string{repName}, nil
}
