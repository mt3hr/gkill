package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// URLogRepository はブックマーク（urlog）のリポジトリが満たす契約です。
// Repository の共通契約に、URL・タイトル・説明・ファビコン・サムネイルを持つ URLog 実体を
// 直接扱うメソッドを足したものです。
// Kyou 系のメソッドはメタ情報しか返さないので、URLや画像が要るときは URLog 系のメソッドを使ってください。
type URLogRepository interface {
	// FindKyous の契約は Repository.FindKyous を参照。
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	// GetKyou の契約は Repository.GetKyou を参照。
	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	// GetKyouHistories の契約は Repository.GetKyouHistories を参照。
	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	// GetPath の契約は Repository.GetPath を参照。
	// urlog はブックマークをSQLiteの中に持つため、id が非空でも返るのは同じDBファイルのパス（絶対パス）です。
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

	// GetRepName の契約は Repository.GetRepName を参照。集約は固定で "URLogReps" を返します。
	GetRepName(ctx context.Context) (string, error)

	// Close の契約は Repository.Close を参照。
	Close(ctx context.Context) error

	// FindURLog は検索条件に一致するURLogをURL・タイトル・説明込みで返します。
	//
	// 単一リポジトリは RelatedTime の降順で返しますが、集約はマップ経由で集めるため順序を保証しません。
	// 集約は ID（query.OnlyLatestData が false なら ID に UpdateTime の Unix 秒を連結したキー）
	// ごとに UpdateTime が最も新しい1件だけを残します。
	// query.UpdateCache が true のときは検索前にキャッシュを更新します。
	// キーワード検索の対象列は URL・TITLE・DESCRIPTION です。
	//
	// query.ExcludeURLogThumbnailImage が true のとき、ThumbnailImage は空文字で返ります。
	// サムネイルはbase64で1行あたり平均400KBに達しキャッシュ表（既定でインメモリDB）に載せられないため、
	// キャッシュ実装は false のとき検索そのものを下層の実DBへ回します。
	FindURLog(ctx context.Context, query *find.FindQuery) ([]URLog, error)

	// GetURLog は id に対応するURLogを1件返します。
	// updateTime が nil なら最新バージョン、非nilならそのバージョンを返します。
	// 見つからない場合は (nil, nil) を返します（エラーではありません）。
	//
	// キャッシュ実装はサムネイルをキャッシュ表に持たないので、RepName からその版を持つリポジトリを
	// 特定して読み直します。持ち主が見つからないときや実DBにその版が無いときは
	// ThumbnailImage が空文字のままになります（エラーではありません）。
	GetURLog(ctx context.Context, id string, updateTime *time.Time) (*URLog, error)

	// GetURLogHistories は id の全バージョンを返します。
	// 集約は UpdateTime の降順に整列して返します。
	// サムネイルの扱いは GetURLog と同じです。
	GetURLogHistories(ctx context.Context, id string) ([]URLog, error)

	// AddURLogInfo はURLogを1件追記します。
	// 追記専用なので、更新は同一IDで新しい UpdateTime の版を、削除は IsDeleted=true の版を追加します。
	// urlog.URL が空白のみのときはエラーです。
	// 集約（URLogRepositories）は未実装で常にエラーを返すため、書き込みは書き込み先リポジトリに対して行ってください。
	AddURLogInfo(ctx context.Context, urlog URLog) error

	// UnWrapTyped の契約は Repository.UnWrap を参照。戻り値が URLogRepository である点だけが異なります。
	UnWrapTyped() ([]URLogRepository, error)

	// UnWrap の契約は Repository.UnWrap を参照。
	UnWrap() ([]Repository, error)
}
