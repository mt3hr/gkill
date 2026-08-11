package gkill_plugin

import (
	"encoding/json"
	"time"
)

// PluginRequest はgkill本体からプラグインプロセスに送るリクエスト（改行区切りJSON）。
type PluginRequest struct {
	// ID はリクエストとレスポンスを対応付けるUUID。
	ID string `json:"id"`

	// Command は実行するコマンド名。
	// find_kyous / get_kyou / get_rep_name / get_content_html / get_config_html / post_config / get_gps_logs / ping / close
	Command string `json:"command"`

	// Query は find_kyous コマンドで使用する検索条件。
	Query *PluginQuery `json:"query,omitempty"`

	// KyouID は get_kyou / get_content_html コマンドで使用するKyouのID。
	KyouID string `json:"kyou_id,omitempty"`

	// UpdateTime は get_kyou コマンドで特定バージョンを取得する場合に使用。
	UpdateTime *time.Time `json:"update_time,omitempty"`

	// FormData は post_config コマンドで使用するフォームデータ（key→value）。
	FormData map[string]string `json:"form_data,omitempty"`

	// GPSLogQuery は get_gps_logs コマンドで使用する取得条件。
	GPSLogQuery *PluginGPSLogQuery `json:"gps_log_query,omitempty"`
}

// PluginGPSLogQuery はプラグインへのGPSログ取得条件。
// 期間の意味は GPSLogRepository.GetGPSLogs と同じで、両端を含む。
// gkillは呼ぶ前にnil解決と入れ替えを済ませてから渡す。
type PluginGPSLogQuery struct {
	// StartTime は取得期間の開始時刻。nilなら下限なし。
	StartTime *time.Time `json:"start_time,omitempty"`

	// EndTime は取得期間の終了時刻。nilなら上限なし。
	EndTime *time.Time `json:"end_time,omitempty"`

	// Offset は返し始める位置（0起点）。ページングに使う。
	Offset int `json:"offset"`

	// Limit は1レスポンスで返す最大点数。0のときはプラグイン側の既定に任せる。
	// 親の bufio.Scanner が1レスポンス32MBまでなので、必ず有限にすること。
	Limit int `json:"limit"`
}

// PluginQuery はプラグインへの検索条件。FindQueryのサブセット。
type PluginQuery struct {
	// Words は全文検索キーワード（AND/OR はWordsAndで制御）。
	Words []string `json:"words"`

	// NotWords は除外キーワード。
	NotWords []string `json:"not_words"`

	// WordsAnd は true のとき全Words AND 検索、false のとき OR。
	WordsAnd bool `json:"words_and"`

	// Tags は絞り込みタグ。
	Tags []string `json:"tags"`

	// NotTags は除外タグ。
	NotTags []string `json:"not_tags"`

	// TagsAnd は true のとき全Tags AND 検索、false のとき OR。
	TagsAnd bool `json:"tags_and"`

	// CalendarStartDate は期間フィルタの開始日時。
	CalendarStartDate *time.Time `json:"calendar_start_date,omitempty"`

	// CalendarEndDate は期間フィルタの終了日時。
	CalendarEndDate *time.Time `json:"calendar_end_date,omitempty"`

	// IsDeleted は true のとき削除済みデータを対象にする。
	IsDeleted bool `json:"is_deleted"`

	// OnlyLatestData は true のとき各IDの最新バージョンのみ返す。
	OnlyLatestData bool `json:"only_latest_data"`

	// Limit は返す最大件数。0 は無制限。
	Limit int `json:"limit"`
}

// PluginResponse はプラグインプロセスからgkill本体に返すレスポンス（改行区切りJSON）。
type PluginResponse struct {
	// ID は対応するPluginRequestのID。
	ID string `json:"id"`

	// Kyous は find_kyous コマンドのレスポンス。
	Kyous []PluginKyou `json:"kyous,omitempty"`

	// Kyou は get_kyou コマンドのレスポンス。
	Kyou *PluginKyou `json:"kyou,omitempty"`

	// RepName は get_rep_name コマンドのレスポンス。
	RepName string `json:"rep_name,omitempty"`

	// HTML は get_content_html / get_config_html コマンドのレスポンス。
	HTML string `json:"html,omitempty"`

	// Pong は ping コマンドのレスポンス。
	Pong bool `json:"pong,omitempty"`

	// GPSLogs は get_gps_logs コマンドのレスポンス。RelatedTimeの昇順で返すこと。
	GPSLogs []PluginGPSLog `json:"gps_logs,omitempty"`

	// HasMoreGPSLogs は get_gps_logs で、Offset+len(GPSLogs) 以降にまだ点があることを表す。
	// gkillはOffsetを進めて続きを取りに行く。
	HasMoreGPSLogs bool `json:"has_more_gps_logs,omitempty"`

	// Errors はエラーメッセージのリスト。空のとき成功。
	Errors []string `json:"errors"`
}

// PluginGPSLog はプラグインが返すGPSログの1点。
// gkill本体の reps.GPSLog に対応する。
// GPSログはKyouではないのでID・更新時刻・削除フラグを持たない。
type PluginGPSLog struct {
	// RelatedTime はこの点を観測した日時。
	RelatedTime time.Time `json:"related_time"`

	// Longitude は経度（度）。
	Longitude float64 `json:"longitude"`

	// Latitude は緯度（度）。
	Latitude float64 `json:"latitude"`
}

// PluginKyou はプラグインが返す記録データ。
// gkill本体のKyou構造体に対応するが、プラグインとの疎結合のため独立した型として定義する。
type PluginKyou struct {
	// IsDeleted は削除済みフラグ。
	IsDeleted bool `json:"is_deleted"`

	// ID は記録の一意識別子。UUIDまたはSNSのポストID等。
	ID string `json:"id"`

	// RepName はリポジトリ表示名（manifest.jsonのrep_nameと一致させること）。
	RepName string `json:"rep_name"`

	// RelatedTime はこの記録が示す日時（ツイート投稿時刻等）。
	RelatedTime time.Time `json:"related_time"`

	// DataType はデータ種別（manifest.jsonのdata_typeと一致させること）。
	DataType string `json:"data_type"`

	// CreateTime はgkill上でのレコード作成時刻。
	CreateTime time.Time `json:"create_time"`

	// CreateApp はレコードを作成したアプリ名。
	CreateApp string `json:"create_app"`

	// CreateDevice はレコードを作成したデバイス名。
	CreateDevice string `json:"create_device"`

	// CreateUser はレコードを作成したユーザID。
	CreateUser string `json:"create_user"`

	// UpdateTime は最終更新時刻。
	UpdateTime time.Time `json:"update_time"`

	// UpdateApp は最終更新アプリ名。
	UpdateApp string `json:"update_app"`

	// UpdateDevice は最終更新デバイス名。
	UpdateDevice string `json:"update_device"`

	// UpdateUser は最終更新ユーザID。
	UpdateUser string `json:"update_user"`

	// ImageSource は画像のURL（https://...）またはdata URI（data:image/...）。
	ImageSource string `json:"image_source"`

	// Tags はこの記録に付与するタグ名のリスト。
	// manifest.jsonのprovidesに"tag"を書いたときだけgkillのタグとして扱われる。
	Tags []string `json:"tags"`

	// Texts はタイムラインやリストに表示するテキストのリスト。
	// manifest.jsonのprovidesに"text"を書いたときだけgkillのテキストとして扱われる。
	Texts []string `json:"texts"`

	// Typed はこの記録の型別データ。nilなら従来どおりメタ情報だけのKyouになる。
	// manifest.jsonのprovidesに書いた種別だけが採用される。
	Typed *PluginTypedData `json:"typed,omitempty"`

	// Notifications はこの記録に付ける通知。
	// Tags / Texts と違い発火時刻を持つので構造体で受ける。
	// manifest.jsonのprovidesに"notification"を書いたときだけ採用される。
	Notifications []PluginNotification `json:"notifications,omitempty"`
}

// PluginTypedData はPluginKyouに載せる型別データ。
//
// 非nilにしてよいのは高々1つ。2つ以上が非nilのときgkillは
// Kmemo→KC→URLog→Nlog→Lantana→TimeIs→Mi の順で最初の1つだけを採用し、
// 残りは警告ログに落とす（プラグイン側の実装ミスを静かに握り潰さないため）。
//
// 各構造体が持つのはその型固有の列だけで、ID・各種時刻・RepName・DataTypeは持たない。
// これらは親のPluginKyouからそのままコピーする。
// クライアントは「KyouのUpdateTimeと型別データのUpdateTimeが秒精度で一致する版」を
// 選んで表示するため、別々の更新時刻を持たせられるようにすると
// 1秒ずれただけで型別ビューが空になる罠をプラグイン作者に押し付けることになる。
type PluginTypedData struct {
	// Kmemo はテキストメモ。data_typeは"kmemo"にすること。
	Kmemo *PluginKmemo `json:"kmemo,omitempty"`

	// KC は数値記録。data_typeは"kc"にすること。
	KC *PluginKC `json:"kc,omitempty"`

	// URLog はブックマーク。data_typeは"urlog"にすること。
	URLog *PluginURLog `json:"urlog,omitempty"`

	// Nlog は支出記録。data_typeは"nlog"にすること。
	Nlog *PluginNlog `json:"nlog,omitempty"`

	// Lantana は気分値。data_typeは"lantana"にすること。
	Lantana *PluginLantana `json:"lantana,omitempty"`

	// TimeIs は時間計測。data_typeは"timeis_start"（終了行を出すなら"timeis_end"）にすること。
	TimeIs *PluginTimeIs `json:"timeis,omitempty"`

	// Mi はタスク。data_typeは"mi_create"/"mi_check"/"mi_limit"/"mi_start"/"mi_end"のいずれかにすること。
	Mi *PluginMi `json:"mi,omitempty"`
}

// PluginKmemo はKmemoの本文。
type PluginKmemo struct {
	// Content は本文。
	Content string `json:"content"`
}

// PluginKC はKCのタイトルと数値。
type PluginKC struct {
	// Title は表示名。空にはできない。
	Title string `json:"title"`

	// NumValue は数値。JSONの数値でも文字列でも受け付ける。
	NumValue json.Number `json:"num_value"`
}

// PluginURLog はURLogのブックマーク情報。
// FaviconImage / ThumbnailImage はいずれもbase64（data URIのプレフィックスは付けない）。
type PluginURLog struct {
	URL            string `json:"url"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FaviconImage   string `json:"favicon_image,omitempty"`
	ThumbnailImage string `json:"thumbnail_image,omitempty"`
}

// PluginNlog はNlogの支出情報。
type PluginNlog struct {
	Shop   string      `json:"shop"`
	Title  string      `json:"title"`
	Amount json.Number `json:"amount"`
}

// PluginLantana はLantanaの気分値（0〜10）。
type PluginLantana struct {
	Mood int `json:"mood"`
}

// PluginTimeIs はTimeIsの計測情報。EndTimeがnilなら計測中。
type PluginTimeIs struct {
	Title     string     `json:"title"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

// PluginMi はMiのタスク情報。
type PluginMi struct {
	Title             string     `json:"title"`
	IsChecked         bool       `json:"is_checked"`
	BoardName         string     `json:"board_name"`
	LimitTime         *time.Time `json:"limit_time,omitempty"`
	EstimateStartTime *time.Time `json:"estimate_start_time,omitempty"`
	EstimateEndTime   *time.Time `json:"estimate_end_time,omitempty"`
}

// PluginNotification はKyouに付ける通知。
// ID・作成/更新時刻は親のPluginKyouから導出するのでここには持たない。
type PluginNotification struct {
	// Content は通知本文。
	Content string `json:"content"`

	// NotificationTime は発火予定時刻。
	NotificationTime time.Time `json:"notification_time"`

	// IsNotificated は通知済みフラグ。
	IsNotificated bool `json:"is_notificated"`
}
