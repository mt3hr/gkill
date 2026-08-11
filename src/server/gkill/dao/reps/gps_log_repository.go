package reps

import (
	"context"
	"time"
)

// GPSLogRepository はGPSログのリポジトリが満たす契約です。
//
// GPSログはKyouではないので Repository は実装せず、ID・更新時刻・削除フラグを持ちません。
// 実体は日付ごとのGPXファイル（yyyyMMdd.gpx）が並ぶディレクトリで、書き込み口はありません。
// 返るのはトラックポイント1点1点で、どのファイル由来かの情報は落ちます。
type GPSLogRepository interface {
	// GetAllGPSLogs はリポジトリ内の全GPXファイルを読み、全トラックポイントを返します。
	// 集約（GPSLogRepositories）は全リポジトリぶんを連結し RelatedTime の降順で返します。
	// 1点も無ければ空スライスで、エラーではありません。
	GetAllGPSLogs(ctx context.Context) ([]GPSLog, error)

	// GetGPSLogs は指定期間のトラックポイントを返します。
	//
	// startTime / endTime はどちらもnil可です。両方nilなら GetAllGPSLogs と同じ全件、
	// 片方だけnilなら残る一方と同じ時刻として扱い、逆順で渡されたらその場で入れ替えます。
	// 期間の判定は他の時刻フィルタと同じく両端を含みます
	// (同一時刻を2つ渡すと、ちょうどその時刻の点だけが返ります)。
	//
	// 走査するのは日付から引いたファイル（yyyyMMdd.gpx）だけです。
	// 時差やファイル境界のずれを拾うため前後1日ぶんを余分に読み、
	// 存在しない日付のファイルは読み飛ばすので、0件でもエラーにはなりません。
	GetGPSLogs(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]GPSLog, error)

	// GetPath の契約は Repository.GetPath を参照。
	// GPSログはトラックポイント単位のパスを持たないため、idの有無にかかわらず
	// GPXファイルを置いたディレクトリのパスを返します。
	GetPath(ctx context.Context, id string) (string, error)

	// GetRepName の契約は Repository.GetRepName を参照。ディレクトリ名になります。
	GetRepName(ctx context.Context) (string, error)

	// UpdateCache の契約は Repository.UpdateCache を参照。
	// GPSログはキャッシュを持たず、毎回GPXファイルを読み直すため何もしません。
	UpdateCache(ctx context.Context) error

	// UnWrapTyped の契約は Repository.UnWrap を参照。GPSLogRepository型で返す版です。
	UnWrapTyped() ([]GPSLogRepository, error)
}

// ReadOnlyGPSLogRepository は書き込み口を持たないGPSログリポジトリの目印です。
//
// GPXファイルのアップロード先の候補から外すためだけに使います。
// 画面の選択肢は端末のrep設定から作られるのでプラグイン由来のrepは出てきませんが、
// サーバ側はリクエストのrep名を GPSLogReps 全件と突き合わせるだけなので、
// 名前を指定されるとプラグインフォルダにGPXを書き込んでしまいます。
type ReadOnlyGPSLogRepository interface {
	GPSLogRepository

	// IsReadOnlyGPSLogRepository はこのリポジトリがアップロード先に選べないことを表します。
	// 実装は空で構いません（型で区別するための目印です）。
	IsReadOnlyGPSLogRepository()
}
