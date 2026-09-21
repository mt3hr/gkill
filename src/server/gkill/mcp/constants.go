package mcp

// 検証・正規化で共有する定数（旧 constants.mjs）。

import (
	"math"
	"os"
	"regexp"
	"strconv"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// 受理するのは RFC 3339（オフセット必須）。ISO-8601 の他の形（秒なし・基本形式・オフセットなし）は弾くので、
// 説明も「ISO-8601」ではなく受理する形を言う（2026-09-18 の実利用報告: オフセット無しを弾いた文言が「ISO-8601で」だけだった）。
const ISODateTimeDesc = "RFC 3339 datetime WITH a timezone offset, e.g. 2026-02-25T10:30:00+09:00 (Z is accepted; a value without an offset is rejected)"
const DateOnlyDesc = "YYYY-MM-DD date string"
const DefaultKyousLimit = 20
const DefaultKyousMaxSizeMB = 0.25
const DefaultKyousIncludeTimeIs = false

// MaxIDFFileBytes は gkill_get_idf_file が返すファイルの上限。base64はJSON-RPCレスポンスに素で載るため、
// 上限がないと大きな動画などで応答が破裂する。超えた場合はローカルパス経由の取得を案内する。
// 起動時の環境変数 GKILL_MCP_MAX_FILE_BYTES（設定ファイルは bootstrap が上書きする）。
var MaxIDFFileBytes = maxIDFFileBytesFromEnv()

const defaultMaxIDFFileBytes = 8 * 1024 * 1024

// maxIDFFileBytesFromEnv は Math.max(1, Number(env) || 8MiB) を写す。
func maxIDFFileBytesFromEnv() int64 {
	return ParseByteLimit(os.Getenv("GKILL_MCP_MAX_FILE_BYTES"), defaultMaxIDFFileBytes, 1)
}

// ParseByteLimit は JS の Math.max(floor, Number(value) || fallback)。数値でない・0・NaN は fallback。
func ParseByteLimit(value string, fallback, floor int64) int64 {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(f) || f == 0 || math.IsInf(f, 0) {
		f = float64(fallback)
	}
	if f < float64(floor) {
		return floor
	}
	return int64(f)
}

var RFC3339Regex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$`)
var DateOnlyRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

var KyousTopLevelFields = NewStringSet(
	"query",
	"locale_name",
	"limit",
	"cursor",
	"max_size_mb",
	"is_include_timeis",
	// include_id / include_rep_name は公開スキーマ（read_tools.go）には無い。
	// v2 で廃止したが、古い一覧を握ったセッションが送り続けるので受理だけ残す
	// （届いたら DetectStaleSchemaSignals が古さの証拠として警告する。ADR-0620）。
	"include_id",
	"include_rep_name",
	"include_plugin_content",
	"plugin_content_max_text_length",
	"plugin_content_format",
	"count_only",
	"group_by",
	"data_types",
	"create_apps",
	"update_apps",
	"num_min",
	"num_max",
	"idf_kinds",
	"include_file_size",
	// 2026-09-19（ADR-0629 / ADR-0630）。どちらも既定 false のオプトイン。
	"include_attached_ids",
	"include_file_urls",
)

// MiProjectionFlagFields は for_mi の射影を選ぶ5フラグ。for_mi:true で1つも立っていない検索は必ず0件なので、
// MCP の入口で include_create_mi を補う（NormalizeKyouArgs。ADR-0627）。
var MiProjectionFlagFields = []string{
	"include_create_mi",
	"include_check_mi",
	"include_limit_mi",
	"include_start_mi",
	"include_end_mi",
}

// KyousGroupByValues は v2 (ADR-0604) の集計・絞り込みの列挙値。サーバ側(get_kyous_mcp_helpers.go)と揃えること。
var KyousGroupByValues = NewStringSet(
	"month",
	"day",
	"week_of_day",
	"hour",
	"data_type",
	"rep_name",
	"create_app",
	"update_app",
	"url_domain",
	"file_extension",
)

var KyousIdfKindValues = NewStringSet("image", "video", "audio", "zip", "other")

// MaxCursorLength: カーソルは v2 から不透明文字列（複合形式 {RFC3339Nano}::{ID}）。
// MCP 側は解釈せず素通しする。長さだけ制限して事故を防ぐ。
const MaxCursorLength = 512

// AppConfigFields は gkill_get_application_config の fields 射影の許可値。
var AppConfigFields = NewStringSet(
	// 接続先の識別。gkill が返す user_id / device をそのまま通す。
	// read サーバと readwrite サーバが別アカウントを向いていても、AI からは
	// 区別する手段が無く「同じAPIなのに件数が違う」と誤診されていた
	// （実利用レビュー）。環境変数の値（GKILL_BASE_URL 等）は
	// 端末の情報なので出さない（ADR-0707）。出すのは gkill 由来のこの2つだけ。
	"user_id",
	"device",
	"tag_struct",
	"mi_board_struct",
	"rep_struct",
	"rep_type_struct",
	"device_struct",
	"kftl_template_struct",
	"mi_default_board",
	"show_tags_in_list",
	// 6 ツリーの description（利用者が設定画面で書く運用メモ）だけを平坦な一覧で返す仮想欄。
	// 応答には fields で明示したときだけ載る（既定の全量には含めない。ツリー側に同じ文が載るため）。
	"descriptions",
)

// MaxDeleteTargets は gkill_delete_kyou / gkill_restore_kyou の一括指定の上限。
// 1件につき取得+更新の2往復なので、上限が無いと1リクエストで数百往復になりうる。
const MaxDeleteTargets = 100

// RepInfosFields は gkill_get_rep_infos の fields 射影の許可値。
// 本番では rep_infos[] だけで数百件になる（rep は rep_type ごとに重複して載るので
// rep 数より膨らむ）。一方「正準値と対応表だけ欲しい」呼び出しが多く、
// 一番よく使う形が一番大きい応答になっていた（実利用レビュー）。
var RepInfosFields = NewStringSet(
	"rep_infos",
	"canonical_rep_types",
	"plugins",
	"attached_data_reps",
)

// AttachedDataKinds は attached_data_reps の data_kind。歴代端末ぶんの Tag_ / Text_ / Notification_ / GPSLogs_ が
// 並ぶので本番では約120件になり、fields で丸ごと落とすか丸ごと取るかの2択だった
// （実利用レビュー）。生成側は handle_get_rep_infos_mcp.go の
// appendAttachedDataRep が渡す4値。
var AttachedDataKinds = NewStringSet("tag", "text", "notification", "gpslog")

// AppConfigUIStateKeys は struct ツリーから既定で剥がす UI 状態キー（ツリーエディタの一時状態。
// 実測で応答が 193.6k→93.0k 字に減る）。check_when_inited / is_force_hide は
// 可視タグ判定に必要なので**絶対に剥がさない**。
var AppConfigUIStateKeys = NewStringSet(
	"is_checked",
	"indeterminate",
	"key",
	"seq",
	"seq_in_parent",
	"is_open_default",
	"parent_folder_id",
	"id",
)

// gkill_get_gps_log のページング（MCP 側実装。gkillは全件を返す）。
const DefaultGpsLimit = 500
const MaxGpsLimit = 5000

var GpsGroupByValues = NewStringSet("day")

// gkill_get_all_rep_names の絞り込み（MCP 側実装。gkillは全件を返す）。
// rep が数百ある環境では「その名前の rep があるか」を確かめるだけで
// 全件を読むことになっていた。
const DefaultRepNamesLimit = 200
const MaxRepNamesLimit = 2000

// タグ名も rep 名と同じ理由で絞り込める必要がある。タグは数百〜数千に育つので、
// 「autolog 系のタグはあるか」を確かめるために全件を受け取るのは無駄が大きい。
// rep 名側にだけ contains / limit があり、タグ名だけ全件返しだった。
const DefaultTagNamesLimit = 200
const MaxTagNamesLimit = 2000

var KyousQueryBooleanFields = NewStringSet(
	"update_cache",
	"include_deleted_data",
	"words_and",
	"tags_and",
	"timeis_words_and",
	"timeis_tags_and",
	"include_create_mi",
	"include_check_mi",
	"include_limit_mi",
	"include_start_mi",
	"include_end_mi",
	"include_end_timeis",
	"is_image_only",
	"for_mi",
	"only_latest_data",
)

// LegacyUseFlagKeys は廃止された旧 use_X フラグ (フィルタの活性化は値フィールドの非null存在で決まる)。
// 後方互換のために受理だけする: use_X:false は対応グループの値キーを落とし、
// use_X:true は捨てる。normalized クエリには決して積まれない。
var LegacyUseFlagKeys = NewStringSet(
	"use_tags",
	"use_reps",
	"use_rep_types",
	"use_ids",
	"use_include_id",
	"use_words",
	"use_timeis",
	"use_timeis_tags",
	"use_calendar",
	"use_map",
	"use_playing",
	"use_update_time",
	"use_mi_board_name",
	"use_period_of_time",
	// 値キーを束ねない (フラグだけ落とす) が、受理しないと未知キーとして throw してしまい
	// 「後方互換で受け付ける」という約束が破れるので、Go の移行実装と同じ16キーを揃える
	"use_mi_sort_type",
	"use_mi_check_state",
)

var KyousQueryStringArrayFields = NewStringSet(
	"rep_types",
	"ids",
	"words",
	"not_words",
	"reps",
	"tags",
	"hide_tags",
	"timeis_words",
	"timeis_not_words",
	"timeis_tags",
)

var KyousQueryNumberFields = NewStringSet("map_radius", "map_latitude", "map_longitude")

// IntegerFieldSpec は整数の検索条件の範囲。
type IntegerFieldSpec struct {
	Name string
	Min  int64
	Max  int64
}

var KyousQueryIntegerFields = []IntegerFieldSpec{
	{Name: "period_of_time_start_time_second", Min: 0, Max: 86399},
	{Name: "period_of_time_end_time_second", Min: 0, Max: 86399},
}

// DateTimeFieldSpec は日時の検索条件の展開規則。
type DateTimeFieldSpec struct {
	Name          string
	AllowDateOnly bool
	EndOfDay      bool
}

var KyousQueryDateTimeFields = []DateTimeFieldSpec{
	{Name: "calendar_start_date", AllowDateOnly: true, EndOfDay: false},
	{Name: "calendar_end_date", AllowDateOnly: true, EndOfDay: true},
	{Name: "playing_time", AllowDateOnly: true, EndOfDay: false},
	{Name: "update_time", AllowDateOnly: true, EndOfDay: false},
}

func kyousQueryIntegerField(name string) (IntegerFieldSpec, bool) {
	for _, spec := range KyousQueryIntegerFields {
		if spec.Name == name {
			return spec, true
		}
	}
	return IntegerFieldSpec{}, false
}

func kyousQueryDateTimeField(name string) (DateTimeFieldSpec, bool) {
	for _, spec := range KyousQueryDateTimeFields {
		if spec.Name == name {
			return spec, true
		}
	}
	return DateTimeFieldSpec{}, false
}

// KyousQueryAllFields は NormalizeKyouQuery が受け付けるクエリキー全体。
// 個別の集合から導出するので、フィールドを足すときはここを触らなくてよい。
var KyousQueryAllFields = buildKyousQueryAllFields()

func buildKyousQueryAllFields() *StringSet {
	s := NewStringSet()
	for _, k := range KyousQueryBooleanFields.Values() {
		s.Add(k)
	}
	for _, k := range KyousQueryStringArrayFields.Values() {
		s.Add(k)
	}
	for _, k := range KyousQueryNumberFields.Values() {
		s.Add(k)
	}
	for _, spec := range KyousQueryIntegerFields {
		s.Add(spec.Name)
	}
	for _, spec := range KyousQueryDateTimeFields {
		s.Add(spec.Name)
	}
	// 上の集合に属さない、個別に検証しているキー
	s.Add("period_of_time_week_of_days")
	s.Add("mi_board_name")
	s.Add("mi_check_state")
	s.Add("mi_sort_type")
	return s
}

// gkill_get_kyous の plugin_content_format 引数。既定は text。
// プラグインのコンテンツHTMLは表示用のCSS/JSでほとんどが埋まっているため、
// 生HTMLを既定で返すとトークンを浪費するだけになる。
var PluginContentFormats = NewStringSet("text", "html", "both")

const DefaultPluginContentFormat = "text"

// ここから下は gkill_get_kyous の include_plugin_content 用。
// プラグインKyouの本文はgkillに保存されておらず、1件ずつプラグインプロセスに
// 問い合わせるしかない。レスポンスに直接埋めるので、単発取得より1件あたりの
// 上限を小さく取り、さらに合計・件数・時間にも上限を設ける。
const DefaultIncludePluginContent = false

// 1件あたりのテキスト上限。既定値は「20件並べても常識的なサイズに収まる」値。
const DefaultInlinePluginContentMaxTextLength = 4000

// 1件だけを対象にして全文を取りたいケースがあるので、上限は大きめに許す。
const MaxPluginContentMaxTextLength = 200000

// 1回のget_kyousで本文を埋めるKyouの最大件数。
const MaxInlinePluginContentKyous = 20

// 埋め込むテキストの合計上限。1件を最大長で取っても収まる値にしてある。
// buildToolResultがペイロードを2回直列化するため、実際の転送量はこの約2倍になる。
const InlinePluginContentTotalTextLength = 200000

// 並列に叩くプラグイン (rep_name) の数。同一プラグイン内は必ず直列にする。
const InlinePluginContentRepConcurrency = 4

// 本文取得全体の打ち切り時間（ミリ秒）。これを過ぎたら新しいリクエストを「始めない」だけで、
// 実行中のリクエストはabortしない。現在のgkillはabortされてもプラグインプロセスを
// 回収しないが、MCPサーバは古いgkillにも接続しうる (古い実装ではabortがプロセスkillになる)。
const InlinePluginContentDeadlineMS = 30000

// htmlToTextに渡す前にHTMLを切り詰める上限。stripTagsが1文字ずつ走査するため、
// 巨大なHTMLをそのまま流すと変換だけで時間を食う。
const MaxInlinePluginContentHTMLLength = 400000

var MiCheckStates = NewStringSet("all", "checked", "uncheck")
var MiSortTypes = NewStringSet("create_time", "estimate_start_time", "estimate_end_time", "limit_time")

// ---------------------------------------------------------------------------
// 型別エンティティの取得/更新エンドポイント
// ---------------------------------------------------------------------------
//
// **削除の語彙はここだけに置くこと。**
// 以前は gkill_delete_kyou の enum (write_tools.go)、DeleteDataTypes
// (write_normalization.go)、DELETE_ENDPOINT_MAP、GET_ENDPOINT_MAP の4箇所に
// 同じ9値が別々に書かれており、しかもサーバ2本ぶんに複製されていた。
// 1つ足し忘れると「スキーマは受理するのにディスパッチで落ちる」
// (あるいはその逆で、正規化を素通りしてエンドポイント未定義で落ちる) になる。
//
// gkill に専用の削除APIは無い。現在値を取って is_deleted を立て、
// 同じ型の更新APIへ送り直す patch 方式なので、種別ごとに取得と更新の対が要る。
// 削除・復活・版履歴の3ツールが同じ対応表を使う。
//
// **型非依存の /api/get_kyou を使ってはいけない。**
// Repositories.GetKyouHistoriesByRepName は冒頭で UnWrap() を呼び、
// キャッシュrepを丸ごとバイパスして 11rep→約940rep に膨れる（実測20.7秒）。
// 型別の XxxRepositories.GetXxxHistoriesByRepName はキャッシュrepを直接回るので安全。

// EntityTarget は型別エンドポイントの対応。
type EntityTarget struct {
	DataType       string
	GetEndpoint    string
	HistoriesKey   string
	UpdateEndpoint string
	RequestKey     string
	ResponseKey    string
}

// EntityTargets は data_type の順に並ぶ（Object.keys の順序が「must be one of」の文言に出る）。
var EntityTargets = []EntityTarget{
	{DataType: "kmemo", GetEndpoint: "/api/get_kmemo", HistoriesKey: "kmemo_histories", UpdateEndpoint: "/api/update_kmemo", RequestKey: "kmemo", ResponseKey: "updated_kmemo"},
	{DataType: "urlog", GetEndpoint: "/api/get_urlog", HistoriesKey: "urlog_histories", UpdateEndpoint: "/api/update_urlog", RequestKey: "urlog", ResponseKey: "updated_urlog"},
	{DataType: "nlog", GetEndpoint: "/api/get_nlog", HistoriesKey: "nlog_histories", UpdateEndpoint: "/api/update_nlog", RequestKey: "nlog", ResponseKey: "updated_nlog"},
	{DataType: "lantana", GetEndpoint: "/api/get_lantana", HistoriesKey: "lantana_histories", UpdateEndpoint: "/api/update_lantana", RequestKey: "lantana", ResponseKey: "updated_lantana"},
	{DataType: "timeis", GetEndpoint: "/api/get_timeis", HistoriesKey: "timeis_histories", UpdateEndpoint: "/api/update_timeis", RequestKey: "timeis", ResponseKey: "updated_timeis"},
	{DataType: "mi", GetEndpoint: "/api/get_mi", HistoriesKey: "mi_histories", UpdateEndpoint: "/api/update_mi", RequestKey: "mi", ResponseKey: "updated_mi"},
	{DataType: "kc", GetEndpoint: "/api/get_kc", HistoriesKey: "kc_histories", UpdateEndpoint: "/api/update_kc", RequestKey: "kc", ResponseKey: "updated_kc"},
	{DataType: "tag", GetEndpoint: "/api/get_tag_histories_by_tag_id", HistoriesKey: "tag_histories", UpdateEndpoint: "/api/update_tag", RequestKey: "tag", ResponseKey: "updated_tag"},
	{DataType: "text", GetEndpoint: "/api/get_text_histories_by_text_id", HistoriesKey: "text_histories", UpdateEndpoint: "/api/update_text", RequestKey: "text", ResponseKey: "updated_text"},
	// rekyou / mirekyou / notification は rep が実在するのに MCP から作成も削除もできなかった。
	// 作成(add)は依然として無いが、Web クライアントや他端末が作ったものを
	// 消す・履歴を見る・戻すことはできるべきなので対応表へ入れる。
	{DataType: "rekyou", GetEndpoint: "/api/get_rekyou", HistoriesKey: "rekyou_histories", UpdateEndpoint: "/api/update_rekyou", RequestKey: "rekyou", ResponseKey: "updated_rekyou"},
	{DataType: "mirekyou", GetEndpoint: "/api/get_mirekyou", HistoriesKey: "mirekyou_histories", UpdateEndpoint: "/api/update_mirekyou", RequestKey: "mirekyou", ResponseKey: "updated_mirekyou"},
	{DataType: "notification", GetEndpoint: "/api/get_gkill_notification_histories_by_notification_id", HistoriesKey: "notification_histories", UpdateEndpoint: "/api/update_gkill_notification", RequestKey: "notification", ResponseKey: "updated_notification"},
}

// EntityTargetOf は data_type の対応表の行を返す。
func EntityTargetOf(dataType string) (EntityTarget, bool) {
	for _, target := range EntityTargets {
		if target.DataType == dataType {
			return target, true
		}
	}
	return EntityTarget{}, false
}

// gkill_get_kyou_history が1回に返す版の既定件数。
// 履歴は編集のたびに1件伸びるので必ず上限を掛ける。
const DefaultKyouHistoryLimit = 20
const MaxKyouHistoryLimit = 200

// EntityDataTypeValues は data_type として受理する値。削除・復活・版履歴のスキーマ enum と正規化がこれを見る。
var EntityDataTypeValues = func() []string {
	out := make([]string, 0, len(EntityTargets))
	for _, target := range EntityTargets {
		out = append(out, target.DataType)
	}
	return out
}()

// ProjectionToEntityDataType は射影名 → エンティティ種別の対応。**語彙が2つある**ことへの唯一の橋。
//
// gkill_get_kyous の DTO が返す data_type は「射影名」（mi_create / timeis_start …）で、
// delete / restore / history が受理するのは EntityTargets のキー＝「エンティティ種別」
// （mi / timeis …）。この違いはどこにも書かれておらず、
// **gkill_add_mi の応答 data_type:"mi_create" をそのまま gkill_delete_kyou へ渡すと落ちる**
// （KFTL の created[] だけはエンティティ語彙なので通る、という三者三様だった。
// 実利用レビュー）。
// 応答をそのまま次のツールへ渡せるよう、射影名も受理して正規化する。
var ProjectionToEntityDataType = []struct{ Projection, Entity string }{
	{"mi_create", "mi"},
	{"mi_check", "mi"},
	{"mi_limit", "mi"},
	{"mi_start", "mi"},
	{"mi_end", "mi"},
	{"mirekyou_create", "mirekyou"},
	{"mirekyou_check", "mirekyou"},
	{"mirekyou_limit", "mirekyou"},
	{"mirekyou_start", "mirekyou"},
	{"mirekyou_end", "mirekyou"},
	{"timeis_start", "timeis"},
	{"timeis_end", "timeis"},
	// idf は Kyou 側の data_type。エンティティとしての削除/履歴は未対応なので写さない。
}

// ProjectionDataTypeNames は ProjectionToEntityDataType のキー一覧。
var ProjectionDataTypeNames = func() []string {
	out := make([]string, 0, len(ProjectionToEntityDataType))
	for _, pair := range ProjectionToEntityDataType {
		out = append(out, pair.Projection)
	}
	return out
}()

// CrossServerToolMentions は「そのサーバに載っていなくても説明文に出てよい」ツール名。
//
// read / write は同じデータを別のサーバで扱うので、読み取り側の説明が
// 書き込み側のツールに触れること自体は正しい —— 「この id は gkill_update_text へ
// 渡すためのものだ」は、その場で呼べという案内ではなく、id を捨てるなという話。
//
// 禁じたいのはその逆で、**「今これを呼べ」と書いておいて、そのサーバに無い**こと。
// read の履歴説明が gkill_restore_kyou を「使え」と書いていたのが実例で、
// AI は載っていないツールを探しにいく。区別は機械では付かないので、
// 参照情報としての言及だけをここへ明示する。**案内を足すときは表ではなく
// ツールの搭載側を直すこと。**
var CrossServerToolMentions = NewStringSet(
	"gkill_restore_kyou",
	"gkill_update_text",
	"gkill_delete_kyou",
	"gkill_submit_kftl",
	"gkill_add_mi",
	"gkill_update_mi",
	"gkill_add_tag",
)

// EntityAndProjectionDataTypeValues は data_type を受け取る口の enum。
//
// 入口は ToEntityDataType を通すので射影名(mi_start / timeis_start …)でも動くのに、
// enum を畳んだ後の語彙だけにしていると、JSON Schema を送信前に検証する
// クライアントが mi_start をサーバへ届く前に弾く。説明文が「応答の data_type を
// そのまま渡せる」と約束しているので正面から食い違っていた(実利用レビュー)。
// 畳んだ後の語彙(EntityDataTypeValues)は検証側で使うので、両方を残す。
var EntityAndProjectionDataTypeValues = func() []string {
	out := make([]string, 0, len(EntityDataTypeValues)+len(ProjectionDataTypeNames))
	out = append(out, EntityDataTypeValues...)
	out = append(out, ProjectionDataTypeNames...)
	return out
}()

// ToEntityDataType は射影名を受け取ったらエンティティ種別へ寄せる。
// エンティティ種別・未知の値はそのまま返す（判定は呼び出し側の許可リストに任せる）。
func ToEntityDataType(dataType string) string {
	for _, pair := range ProjectionToEntityDataType {
		if pair.Projection == dataType {
			return pair.Entity
		}
	}
	return dataType
}

// removedToolHints は消したツールの案内。MCP のツール一覧は**クライアントのセッション寿命で固定**されるので、
// サーバから消しても既存セッションは呼び続ける (2巡目の指摘で live コネクタから再現)。
// しかもそのクライアントが握っている古い説明文は「パスを優先しろ」と、
// まさにこの消えたツールへ誘導している。名前だけ返すと行き止まりになる。
var removedToolHints = map[string]string{
	"gkill_get_idf_file_path": "removed on 2026-08-24: it could never be reached from an HTTP client, and on stdio it was a strict " +
		"subset of the file_path that gkill_get_kyous already puts into IDF payloads. On stdio read " +
		"payload.file_path directly; otherwise call gkill_get_idf_file, which is the only way to get an " +
		`image in front of the model (add thumb:"1024x1024" to stay under the size cap).`,
	"gkill_get_plugin_content": "removed: pass include_plugin_content:true to gkill_get_kyous instead. It inlines plugin bodies into " +
		"the same response rather than costing one round trip per entry.",
}

// UnknownToolMessage は未知ツールのエラー文。消したツールなら代替まで案内する。
func UnknownToolMessage(name string) string {
	if hint, ok := removedToolHints[name]; ok {
		return "Unknown tool: " + name + " — " + hint
	}
	return "Unknown tool: " + name
}

// EnumJoin は StringSet の値を ", " で連結する（"must be one of: …" 用）。
func EnumJoin(s *StringSet) string {
	return s.Join(", ")
}

// stringsToAny は []string を JSON 配列にする（スキーマの enum 用）。
func stringsToAny(items []string) []any {
	return jsonobj.Strings(items...)
}
