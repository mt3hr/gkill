package fakegkill

// 偽 gkill の固定応答。JSON 文字列のまま持ち、Node の採取と Go の再生で同じバイト列を返す。
// 利用者IDは testuser 系、パスや値はサンプル（実データを入れない）。

// kyousPage1 は get_kyous_mcp の1ページ目（cursor 無し）。プラグイン・IDF 画像・IDF 文書・
// kmemo・mi・壊れたプラグインの順。limit で先頭から切る。
const kyousPage1 = `[
{"id":"plug-1","data_type":"claude_conversation","rep_name":"PluginRep","create_app":"gkill_plugin_claudeai","update_app":"gkill_plugin_claudeai","related_time":"2026-09-15T10:00:00+09:00","update_time":"2026-09-15T10:00:00+09:00","tags":["ai"],"payload":{"kind":"plugin","plugin_name":"claudeai"}},
{"id":"k1","data_type":"kmemo","rep_name":"Kmemo_fakepc","create_app":"gkill","update_app":"gkill","related_time":"2026-09-15T09:00:00+09:00","update_time":"2026-09-15T09:00:00+09:00","tags":["日記","朝"],"texts":["補足テキスト"],"payload":{"kind":"kmemo","content":"朝のメモ \"引用\" と \\ バックスラッシュ 🍣"}},
{"id":"k2","data_type":"idf","rep_name":"Files","create_app":"gkill","update_app":"gkill","related_time":"2026-09-14T08:30:00+09:00","update_time":"2026-09-14T08:30:00+09:00","payload":{"kind":"idf","file_name":"photo.png","is_image":true,"is_video":false,"is_audio":false,"is_zip":false,"rep_name":"Files","mime_type":"image/png","file_path":"/data/testuser/gkill/Files/photo.png"}},
{"id":"k3","data_type":"idf","rep_name":"Files","create_app":"gkill","update_app":"gkill","related_time":"2026-09-13T08:30:00+09:00","update_time":"2026-09-13T08:30:00+09:00","payload":{"kind":"idf","file_name":"docs/doc.pdf","is_image":false,"is_video":false,"is_audio":false,"is_zip":false,"rep_name":"Files","mime_type":"application/pdf","file_path":"/data/testuser/gkill/Files/docs/doc.pdf"}},
{"id":"mi-1","data_type":"mi_create","rep_name":"Mi_fakepc","create_app":"gkill","update_app":"gkill","related_time":"2026-09-12T12:00:00+09:00","update_time":"2026-09-12T12:00:00+09:00","payload":{"kind":"mi","title":"買い物","is_checked":false,"board_name":"Inbox","create_time":"2026-09-12T12:00:00+09:00","limit_time":"2026-09-30T00:00:00+09:00"}},
{"id":"plug-err","data_type":"broken_plugin_record","rep_name":"PluginErrRep","create_app":"gkill_plugin_broken","update_app":"gkill_plugin_broken","related_time":"2026-09-11T10:00:00+09:00","update_time":"2026-09-11T10:00:00+09:00","payload":{"kind":"plugin","plugin_name":"broken"}}
]`

// kyousPage2 は cursor 付きの2ページ目。
const kyousPage2 = `[
{"id":"plug-2","data_type":"claude_conversation","rep_name":"PluginRep","create_app":"gkill_plugin_claudeai","update_app":"gkill_plugin_claudeai","related_time":"2026-09-10T10:00:00+09:00","update_time":"2026-09-10T10:00:00+09:00","payload":{"kind":"plugin","plugin_name":"claudeai"}},
{"id":"k5","data_type":"kmemo","rep_name":"Kmemo_fakepc","create_app":"gkill","update_app":"gkill","related_time":"2026-09-09T09:00:00+09:00","update_time":"2026-09-09T09:00:00+09:00","payload":{"kind":"kmemo","content":"2ページ目のメモ"}}
]`

const kyousPlugins = `[{"rep_name":"PluginRep","plugin_name":"claudeai","description":"Claude.ai の会話ログ（サンプル）"},{"rep_name":"PluginErrRep","plugin_name":"broken","description":"本文取得が失敗するプラグイン（サンプル）"}]`

const kyousBuckets = `[{"key":"2026-09-01","count":3},{"key":"2026-09-02","count":5}]`

// pluginContentHTML は get_plugin_content_html の本文（kyou_id ごと）。
var pluginContentHTML = map[string]string{
	"plug-1": `<html><head><title>会話</title><style>body{color:red}</style></head><body><h1>タイトル</h1><p>こんにちは <b>世界</b> &amp; &lt;tag&gt;</p><ul><li>一つ</li><li>二つ</li></ul><script>alert(1)</script></body></html>`,
	"plug-2": `<html><body><p>` + longParagraph + `</p></body></html>`,
}

// longParagraph は 6,000 字超の本文（切り詰めの検査用）。
const longParagraph = "長い本文です。" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの" + "あいうえおかきくけこさしすせそたちつてとなにぬねの"

const pluginList = `[
{"name":"claudeai","version":"1.2.0","description":"Claude.ai の会話ログ（サンプル）","data_type":"claude_conversation","rep_name":"PluginRep","rep_names":["PluginRep"],"emits_kyou":true,"provides":["typed_data"],"is_alive":true,"process_running":true,"typed_index":{"ok":true,"state":"ready","record_count":120,"oldest":"2025-01-01T00:00:00+09:00","newest":"2026-09-15T10:00:00+09:00","truncated":false,"built_at":"2026-09-19T00:00:00+09:00"}},
{"name":"broken","version":"0.1.0","description":"本文取得が失敗するプラグイン（サンプル）","data_type":"broken_plugin_record","rep_name":"PluginErrRep","rep_names":["PluginErrRep"],"emits_kyou":true,"is_alive":false,"process_running":false,"last_error":"exit status 1: /data/testuser/plugins/broken failed"},
{"name":"gpsonly","version":"2.0.0","description":"GPS だけを提供するプラグイン（サンプル）","data_type":"","rep_name":"","rep_names":[],"emits_kyou":false,"is_alive":true,"process_running":true,"gps_index":{"point_count":5000,"oldest":"2025-01-01T00:00:00+09:00","newest":"2026-09-15T10:00:00+09:00","fetched_at":"2026-09-19T00:00:00+09:00"}}
]`

const miBoards = `["Inbox","Work","買い物"]`

const tagNames = `["日記","朝","ai","work","Work2","仕事","買い物"]`

const repNames = `["Kmemo_fakepc","Files","Mi_fakepc","PluginRep","PluginErrRep","Tag_fakepc","Text_fakepc","GPSLogs_fakepc","Nlog_fakepc","URLog_fakepc"]`

// gpsLogs は時刻降順。同一時刻の点が3つ並ぶ（カーソルの同時刻カウントの検査用）。
const gpsLogs = `[
{"related_time":"2026-09-15T10:00:05+09:00","longitude":139.70,"latitude":35.66},
{"related_time":"2026-09-15T10:00:00+09:00","longitude":139.71,"latitude":35.67},
{"related_time":"2026-09-15T10:00:00+09:00","longitude":139.72,"latitude":35.68},
{"related_time":"2026-09-15T10:00:00+09:00","longitude":139.73,"latitude":35.69},
{"related_time":"2026-09-14T23:59:59+09:00","longitude":139.74,"latitude":35.70},
{"related_time":"2026-09-14T08:00:00+09:00","longitude":139.75,"latitude":35.71},
{"related_time":"2026-09-13T08:00:00+09:00","longitude":139.76,"latitude":35.72}
]`

const repInfos = `{
"rep_infos":[
{"rep_name":"Kmemo_fakepc","rep_type":"kmemo","indexed_at":"2026-09-19T00:00:00+09:00","use_to_write":true},
{"rep_name":"Files","rep_type":"directory","use_to_write":true},
{"rep_name":"Mi_fakepc","rep_type":"mi","use_to_write":true},
{"rep_name":"Nlog_fakepc","rep_type":"nlog","use_to_write":false},
{"rep_name":"URLog_fakepc","rep_type":"urlog","use_to_write":false}
],
"canonical_rep_types":["kmemo","directory","mi","nlog","urlog","timeis","lantana","kc","tag","text","notification","gpslog","git_commit_log","rekyou","mirekyou"],
"plugins":[
{"rep_name":"PluginRep","data_type":"claude_conversation","plugin_name":"claudeai"},
{"rep_name":"PluginErrRep","data_type":"broken_plugin_record","plugin_name":"broken"}
],
"attached_data_reps":[
{"rep_name":"Tag_fakepc","data_kind":"tag","use_to_write":true},
{"rep_name":"Tag_oldphone","data_kind":"tag","use_to_write":false},
{"rep_name":"Text_fakepc","data_kind":"text","use_to_write":true},
{"rep_name":"Notification_fakepc","data_kind":"notification","use_to_write":true},
{"rep_name":"GPSLogs_fakepc","data_kind":"gpslog","use_to_write":true}
]
}`

const applicationConfig = `{
"user_id":"testuser","device":"fakepc","use_dark_theme":true,"google_map_api_key":"","rykv_image_list_column_number":3,"rykv_hot_reload":false,
"mi_default_board":"Inbox","rykv_default_period":-1,"mi_default_period":-1,"is_show_share_footer":false,"default_page":"rykv",
"version":"9.9.9","commit_hash":"0123456789abcdef","build_time":"2026-09-19T00:00:00+09:00",
"show_tags_in_list":true,
"kftl_template_struct":{"name":"__root__","id":"t-root","title":"","template":"","description":"","children":[
{"id":"t1","title":"日記","template":"ーにっき\n","description":"1日の終わりに書く日記の雛形","children":null,"is_dir":false,"check_when_inited":false,"is_open":true,"key":"t1"}
],"is_dir":true,"key":"__root__"},
"tag_struct":{"name":"__root__","id":"tg-root","tag_name":"","check_when_inited":false,"is_force_hide":false,"description":"","children":[
{"id":"tg-life","tag_name":"生活","name":"生活","description":"日々の生活に関するタグの入れ物","children":[
{"id":"tg-1","tag_name":"日記","name":"日記","description":"その日の出来事の記録。夜にまとめて書く","children":null,"is_dir":false,"check_when_inited":true,"is_force_hide":false,"is_open":false,"key":"tg-1","indeterminate":false},
{"id":"tg-2","tag_name":"朝","name":"朝","description":"","children":[],"is_dir":false,"check_when_inited":false,"is_force_hide":true,"is_open":false,"key":"tg-2"}
],"is_dir":true,"check_when_inited":false,"is_force_hide":false,"is_open":true,"key":"tg-root"},
{"id":"tg-3","tag_name":"work","name":"仕事のタグ","children":null,"is_dir":false,"check_when_inited":true,"is_force_hide":false,"is_open":false,"key":"tg-3"}
],"is_dir":true,"key":"__root__"},
"mi_board_struct":{"name":"__root__","id":"mb-root","board_name":"","check_when_inited":false,"children":[
{"id":"mb-1","board_name":"Inbox","name":"Inbox","description":"未分類のタスク置き場。週に1回振り分ける","children":null,"is_dir":false,"check_when_inited":true,"is_open":false,"key":"mb-1"},
{"id":"mb-2","board_name":"Work","name":"Work","children":null,"is_dir":false,"check_when_inited":true,"key":"mb-2"}
],"is_dir":true,"key":"__root__"},
"rep_struct":{"name":"__root__","id":"rs-root","rep_name":"","check_when_inited":false,"ignore_check_rep_rykv":false,"children":[
{"id":"rs-dev","rep_name":"端末","name":"端末","children":[
{"id":"rs-1","rep_name":"Kmemo_fakepc","name":"Kmemo_fakepc","description":"PCで書いたメモ","children":null,"is_dir":false,"check_when_inited":true,"ignore_check_rep_rykv":false,"is_open":false,"key":"rs-1"},
{"id":"rs-2","rep_name":"Files","name":"写真","children":null,"is_dir":false,"check_when_inited":true,"ignore_check_rep_rykv":true,"is_open":false,"key":"rs-2"}
],"is_dir":true,"check_when_inited":false,"ignore_check_rep_rykv":false,"is_open":true,"key":"rs-root"}
],"is_dir":true,"key":"__root__"},
"rep_type_struct":{"name":"__root__","id":"rt-root","rep_type_name":"","check_when_inited":false,"children":[
{"id":"rt-1","rep_type_name":"kmemo","name":"kmemo","children":null,"is_dir":false,"check_when_inited":true,"is_open":false,"key":"rt-1"}
],"is_dir":true,"key":"__root__"},
"device_struct":{"name":"__root__","id":"dv-root","device_name":"","check_when_inited":false,"children":[
{"id":"dv-1","device_name":"fakepc","name":"fakepc","description":"開発用のPC","children":null,"is_dir":false,"check_when_inited":true,"is_open":false,"key":"dv-1"}
],"is_dir":true,"key":"__root__"},
"dnote_json_data":{"ignored":true},"ryuu_json_data":null,"dashboard_json_data":null,"playing_timeis_json_data":null
}`

// histories は型別 get エンドポイントの応答（id ごと）。先頭が最新版。
var histories = map[string]string{
	"kmemo/km-1": `[
{"id":"km-1","rep_name":"Kmemo_fakepc","related_time":"2026-09-15T09:00:00+09:00","content":"最新の内容","data_type":"kmemo","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-16T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false},
{"id":"km-1","rep_name":"Kmemo_fakepc","related_time":"2026-09-15T09:00:00+09:00","content":"古い内容","data_type":"kmemo","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"kmemo/km-del": `[
{"id":"km-del","rep_name":"Kmemo_fakepc","related_time":"2026-09-15T09:00:00+09:00","content":"消したメモ","data_type":"kmemo","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-17T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":true},
{"id":"km-del","rep_name":"Kmemo_fakepc","related_time":"2026-09-15T09:00:00+09:00","content":"消したメモ","data_type":"kmemo","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	// 更新時刻がテストの固定現在時刻と同じ秒（nextUpdateTime が +1 秒する経路）
	"kmemo/km-same-second": `[
{"id":"km-same-second","rep_name":"Kmemo_fakepc","related_time":"2026-09-15T09:00:00+09:00","content":"同じ秒","data_type":"kmemo","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-21T03:30:45+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"kmemo/km-many": `[
{"id":"km-many","rep_name":"Kmemo_fakepc","related_time":"2026-09-01T09:00:00+09:00","content":"v5","data_type":"kmemo","create_time":"2026-09-01T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-05T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false},
{"id":"km-many","rep_name":"Kmemo_fakepc","related_time":"2026-09-01T09:00:00+09:00","content":"v4","data_type":"kmemo","create_time":"2026-09-01T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-04T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false},
{"id":"km-many","rep_name":"Kmemo_fakepc","related_time":"2026-09-01T09:00:00+09:00","content":"v3","data_type":"kmemo","create_time":"2026-09-01T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-03T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false},
{"id":"km-many","rep_name":"Kmemo_fakepc","related_time":"2026-09-01T09:00:00+09:00","content":"v2","data_type":"kmemo","create_time":"2026-09-01T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-02T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false},
{"id":"km-many","rep_name":"Kmemo_fakepc","related_time":"2026-09-01T09:00:00+09:00","content":"v1","data_type":"kmemo","create_time":"2026-09-01T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-01T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"urlog/url-1": `[
{"id":"url-1","rep_name":"URLog_fakepc","related_time":"2026-09-15T09:00:00+09:00","url":"https://example.com/a","title":"例","description":"","favicon_image":"","thumbnail_image":"","data_type":"urlog","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"nlog/nlog-1": `[
{"id":"nlog-1","rep_name":"Nlog_fakepc","related_time":"2026-09-15T09:00:00+09:00","shop":"スーパー","title":"食費","amount":1200,"data_type":"nlog","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"lantana/lan-1": `[
{"id":"lan-1","rep_name":"Lantana_fakepc","related_time":"2026-09-15T09:00:00+09:00","mood":7,"data_type":"lantana","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"timeis/ti-1": `[
{"id":"ti-1","rep_name":"TimeIs_fakepc","title":"作業中","start_time":"2026-09-15T09:00:00+09:00","end_time":null,"data_type":"timeis_start","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"timeis/ti-ended": `[
{"id":"ti-ended","rep_name":"TimeIs_fakepc","title":"終了済み","start_time":"2026-09-15T09:00:00+09:00","end_time":"2026-09-15T10:00:00+09:00","data_type":"timeis_end","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T10:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"mi/mi-1": `[
{"id":"mi-1","rep_name":"Mi_fakepc","title":"買い物","is_checked":false,"board_name":"Inbox","limit_time":"2026-09-30T00:00:00+09:00","estimate_start_time":null,"estimate_end_time":null,"data_type":"mi_create","create_time":"2026-09-12T12:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-12T12:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"kc/kc-1": `[
{"id":"kc-1","rep_name":"KC_fakepc","related_time":"2026-09-15T09:00:00+09:00","title":"体重","num_value":62.5,"data_type":"kc","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"tag/tag-1": `[
{"id":"tag-1","rep_name":"Tag_fakepc","target_id":"km-1","tag":"日記","related_time":"2026-09-15T09:00:00+09:00","data_type":"tag","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"text/text-1": `[
{"id":"text-1","rep_name":"Text_fakepc","target_id":"km-1","text":"補足","related_time":"2026-09-15T09:00:00+09:00","data_type":"text","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"rekyou/rk-1": `[
{"id":"rk-1","rep_name":"ReKyou_fakepc","target_id":"km-1","related_time":"2026-09-15T09:00:00+09:00","data_type":"rekyou","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"mirekyou/mrk-1": `[
{"id":"mrk-1","rep_name":"MiReKyou_fakepc","target_id":"km-1","is_checked":false,"board_name":"Inbox","limit_time":null,"estimate_start_time":null,"estimate_end_time":null,"data_type":"mirekyou_create","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
	"notification/nt-1": `[
{"id":"nt-1","rep_name":"Notification_fakepc","target_id":"km-1","content":"通知","notification_time":"2026-09-20T09:00:00+09:00","is_notificated":false,"data_type":"notification","create_time":"2026-09-15T09:00:00+09:00","create_app":"gkill","create_device":"fakepc","create_user":"testuser","update_time":"2026-09-15T09:00:00+09:00","update_app":"gkill","update_device":"fakepc","update_user":"testuser","is_deleted":false}
]`,
}

const kftlCreated = `[{"id":"kftl-1","data_type":"kmemo","updated":false,"related_time":"2026-09-20T10:00:00+09:00"},{"id":"kftl-2","data_type":"mi","updated":false,"related_time":"2026-09-20T10:00:00+09:00"},{"id":"kftl-3","data_type":"timeis","updated":true,"related_time":"2026-09-20T10:00:00+09:00"}]`
