package gkill_plugin

import "time"

// PluginRequest はgkill本体からプラグインプロセスに送るリクエスト（改行区切りJSON）。
type PluginRequest struct {
	// ID はリクエストとレスポンスを対応付けるUUID。
	ID string `json:"id"`

	// Command は実行するコマンド名。
	// find_kyous / get_kyou / get_rep_name / get_content_html / get_config_html / post_config / ping / close
	Command string `json:"command"`

	// Query は find_kyous コマンドで使用する検索条件。
	Query *PluginQuery `json:"query,omitempty"`

	// KyouID は get_kyou / get_content_html コマンドで使用するKyouのID。
	KyouID string `json:"kyou_id,omitempty"`

	// UpdateTime は get_kyou コマンドで特定バージョンを取得する場合に使用。
	UpdateTime *time.Time `json:"update_time,omitempty"`

	// FormData は post_config コマンドで使用するフォームデータ（key→value）。
	FormData map[string]string `json:"form_data,omitempty"`
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

	// Errors はエラーメッセージのリスト。空のとき成功。
	Errors []string `json:"errors"`
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
	Tags []string `json:"tags"`

	// Texts はタイムラインやリストに表示するテキストのリスト。
	Texts []string `json:"texts"`
}
