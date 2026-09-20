package mcp

// 読み取りツールの定義。read / readwrite の2サーバが共有する（旧 read-tools.mjs）。
// 書き込み専用サーバも rep名 / 板名 / タグ名などをここから取る。
//
// 以前はサーバごとに逐語コピーされていて、同じツールの description が
// 接続先サーバによって違うという状態になっていた（locale_name の説明、
// gkill_get_mi_board_list / gkill_get_all_tag_names / gkill_get_all_rep_names の本文）。
//
// 定義は *jsonobj.Object で持つ。キーの順序（name → description → inputSchema、
// properties の並び）が tools/list のバイト列と schema_revision に効くので、struct や map ではなく
// 順序つきの値で書く。verify_docs は `Name: "gkill_…"` の行を数えるので、その形を崩さない。

import (
	"fmt"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// tool はツール定義1件（name / description / inputSchema の順）。
func tool(name, description string, inputSchema *jsonobj.Object) *jsonobj.Object {
	return jsonobj.Obj("name", name, "description", description, "inputSchema", inputSchema)
}

// schema は {type:"object", properties, [required], additionalProperties:false}。
func schema(properties *jsonobj.Object, required []string) *jsonobj.Object {
	o := jsonobj.Obj("type", "object", "properties", properties)
	if required != nil {
		o.Set("required", jsonobj.Strings(required...))
	}
	o.Set("additionalProperties", false)
	return o
}

const localeNameDesc = "Locale for server messages, e.g. ja/en. Defaults to server default (ja)."

// ReadTools は読み取りツール 11 本（順序は固定。gkill_status が先頭、gkill_get_mcp_help がその次）。
var ReadTools = []*jsonobj.Object{
	// gkill_status は3サーバ全部に載る（write 専用サーバは WriteServerReadToolNames で選ぶ）。
	// description の末尾には Server が起動時に「そのサーバの」schema_revision を
	// 焼き込む（status_tool.go の StampSchemaRevision）。ここに書く文はその印の前に来る。
	// 引数は取らない —— 引数を足すと StaleSchemaArgKindsByTool の対象になり、
	// 「古さを確かめるツール自身が古いスキーマで壊れる」ことになる。
	tool(
		"gkill_status",
		"Report what this MCP server is: server_kind (read / write / readwrite), the gkill account it is connected to "+
			"(account.user_id / account.device), the gkill build (gkill.version / commit_hash / build_time), transport (stdio / http), "+
			"started_at / uptime_seconds, tool_count and schema_revision. "+
			"schema_revision identifies the generation of the tool list a client holds. This description ends with the revision of "+
			"the list you were given; if the response's schema_revision differs, your client fetched the tools before the server "+
			"changed them (tool lists are fetched once per client session) — reconnect the MCP client before trusting any argument "+
			"name or description in this list. "+
			"Call it first when a search or write behaves unexpectedly: a different account or a stale tool list explains most of them. "+
			"Takes no arguments.",
		schema(jsonobj.New(), nil),
	),
	// ツール一覧の説明文は要約で、本文はここから topic ごとに取り出す（help_topics.go、ADR-0622）。
	// gkill_status と同じく3サーバ全部に載る（write 専用サーバは WriteServerReadToolNames で選ぶ）。
	// topic は string なので古スキーマ救済表には載せない。enum なのは schema_contract のスモークが
	// enum[0]（index）で呼ぶため。
	tool(
		"gkill_get_mcp_help",
		"Return the detailed guide for one topic of this MCP server. The tool descriptions in this list are summaries; "+
			"read the relevant topic before a first search (search / pagination / mi / data_types), before reading files (idf) "+
			"or plugin bodies (plugin), before writing KFTL text (kftl), when looking for deleted entries (deleted) or repository "+
			"names (rep), and whenever a response carries warnings you do not understand. Omit topic (or pass \"index\") for "+
			"the list of topics. Static text — no round trip to gkill.",
		schema(jsonobj.Obj(
			"topic", jsonobj.Obj(
				"type", "string",
				"enum", jsonobj.Strings(HelpTopicNames...),
				"description", "Topic to read. Default: index (the list of topics with one line each).",
			),
		), nil),
	),
	tool(
		"gkill_get_kyous",
		// 説明文は要約にとどめる。応答フィールドの一覧・クエリの意味論・射影・ペイロード形などの
		// 本文は help_topics.go（gkill_get_mcp_help）へ移した（ADR-0622）。
		// tool_handlers_test.go が固定する3句（partial と独立に warnings を見る）はここに残す。
		"Search life-log entries (kyou) and return them newest first with tags, texts, notifications and the type-specific payload inline. "+
			"Start with query.calendar_start_date / calendar_end_date, query.words, query.tags, limit and data_types; tasks need query.for_mi plus include_create_mi (see gkill_get_mcp_help topic:mi). "+
			"ALWAYS inspect warnings[] even when partial is false: a repository may have failed to load, so its records are absent — do not put a repository named by that warning back into query.reps. Unknown filter values and ids that match nothing are reported there too. "+
			"limit / max_size_mb cap what is RETURNED, not what is SEARCHED — narrow the calendar range, data_types or reps to make a query faster. "+
			"Page by passing next_cursor back as cursor; cursor pages omit total_count, so read remaining_count. count_only / group_by give counts and histograms without payloads. "+
			"Every entry carries id and rep_name; pass include_attached_ids:true when you need tag_entities[] / text_entities[] (the annotation ids that gkill_update_text / gkill_delete_kyou need). "+
			"Details: gkill_get_mcp_help topics search (response fields, query semantics, payload shapes), pagination, mi, data_types, plugin, idf, deleted.",
		schema(buildKyousProperties(), nil),
	),
	tool(
		"gkill_get_mi_board_list",
		"Get the Mi (task) board names that are actually IN USE. Boards are like Kanban columns that organize tasks. "+
			"These are the distinct board_name values of live (latest, not deleted) Mi and MiReKyou records — not a configured list, "+
			"so an account with no tasks returns [] and a board whose last task was deleted disappears. "+
			"The default board (ApplicationConfig.mi_default_board, usually \"Inbox\") is NOT included unless a task actually sits there, "+
			"and the web UI's \"all boards\" entry is a client-side pseudo-board that never appears here. "+
			"Use this to discover existing board names for Mi queries (query.mi_board_name), and call it before gkill_add_mi / gkill_update_mi. Any string can be used as board_name — non-existent names create new boards. "+
			"Response fields: boards[] (array of board name strings).",
		schema(jsonobj.Obj(
			"locale_name", jsonobj.Obj("type", "string", "description", localeNameDesc),
		), nil),
	),
	tool(
		"gkill_get_all_tag_names",
		"Get tag names defined in gkill. Use this to discover available tags for filtering in "+
			"gkill_get_kyous via query.tags or query.timeis_tags. Accounts accumulate hundreds of tags, "+
			"so narrow with contains when you only need to confirm one exists, or to list one family "+
			"such as the autolog tags. Response fields: tag_names[], total_count (before limit), "+
			"returned_count, truncated. Only tags whose target still exists are listed.",
		schema(jsonobj.Obj(
			"locale_name", jsonobj.Obj("type", "string", "description", localeNameDesc),
			"contains", jsonobj.Obj(
				"type", "string",
				"description",
				"Case-insensitive substring filter on the tag name. Omit for no filter. "+
					"Matching is done on the full list, so total_count reflects the filter.",
			),
			"limit", jsonobj.Obj(
				"type", "integer",
				"default", DefaultTagNamesLimit,
				"minimum", 1,
				"maximum", MaxTagNamesLimit,
				"description",
				fmt.Sprintf("Max names to return after filtering (1-%d). Default: %d. ", MaxTagNamesLimit, DefaultTagNamesLimit)+
					"When more matched, truncated is true and total_count tells you how many there were.",
			),
		), nil),
	),
	tool(
		"gkill_get_all_rep_names",
		"Get repository names configured in gkill. Use this to discover rep names for filtering in "+
			"gkill_get_kyous via query.reps. Deployments can hold hundreds of repositories, so narrow with "+
			"contains when you only need to confirm one name exists. Response fields: rep_names[], total_count "+
			"(before limit), returned_count, truncated. "+
			"Only repositories that supply kyou appear here — a plugin that emits no kyou (for example a "+
			"GPS-only plugin) is absent by design, and its manifest rep_name is not a query.reps value. "+
			"For canonical rep_types values, per-repository index freshness and where attached data lives, "+
			"call gkill_get_rep_infos instead.",
		schema(jsonobj.Obj(
			"locale_name", jsonobj.Obj("type", "string", "description", localeNameDesc),
			"contains", jsonobj.Obj(
				"type", "string",
				"description",
				"Case-insensitive substring filter on the repository name. Omit for no filter. "+
					"Matching is done on the full list, so total_count reflects the filter.",
			),
			"limit", jsonobj.Obj(
				"type", "integer",
				"default", DefaultRepNamesLimit,
				"minimum", 1,
				"maximum", MaxRepNamesLimit,
				"description",
				fmt.Sprintf("Max names to return after filtering (1-%d). Default: %d. ", MaxRepNamesLimit, DefaultRepNamesLimit)+
					"When more matched, truncated is true and total_count tells you how many there were.",
			),
		), nil),
	),
	tool(
		"gkill_get_gps_log",
		"Get GPS log points in a date range (deduplicated, newest first). Supports cursor pagination and aggregation: "+
			"use count_only to size a range, group_by:\"day\" for daily coverage buckets, and limit/cursor to page through points. "+
			"Response fields: gps_logs[], total_count (cursor-less responses), returned_count, remaining_count, has_more, next_cursor, buckets (group_by only). Read-only.",
		schema(jsonobj.Obj(
			"start_date", jsonobj.Obj(
				"type", "string",
				"description", "Required "+ISODateTimeDesc+" or "+DateOnlyDesc,
			),
			"end_date", jsonobj.Obj(
				"type", "string",
				"description", "Required "+ISODateTimeDesc+" or "+DateOnlyDesc,
			),
			"limit", jsonobj.Obj(
				"type", "integer",
				"default", DefaultGpsLimit,
				"minimum", 1,
				"maximum", MaxGpsLimit,
				"description", fmt.Sprintf("Max GPS points per page (1-%d). Default: %d.", MaxGpsLimit, DefaultGpsLimit),
			),
			"cursor", jsonobj.Obj(
				"type", "string",
				"description",
				"Opaque pagination cursor. Pass next_cursor from the previous response verbatim — do not "+
					"construct or edit it. It is not an ISO-8601 datetime, and it is not interchangeable with "+
					"the gkill_get_kyous cursor.",
			),
			"count_only", jsonobj.Obj(
				"type", "boolean",
				"default", false,
				"description", "Return only total_count. Cannot be combined with cursor.",
			),
			"group_by", jsonobj.Obj(
				"type", "string",
				"enum", jsonobj.Strings("day"),
				"description", "Return buckets:[{key:\"YYYY-MM-DD\",count}] — daily coverage instead of points. Cannot be combined with cursor.",
			),
			"locale_name", jsonobj.Obj("type", "string", "description", localeNameDesc),
		), []string{"start_date", "end_date"}),
	),
	tool(
		"gkill_get_application_config",
		"Get application configuration including tag hierarchy, task board structure, repository structure, and KFTL templates. "+
			"It also answers WHICH ACCOUNT this MCP server is connected to: user_id and device. "+
			"Different gkill MCP servers (read / write / readwrite) can be configured against different accounts, so a record "+
			"written through one may be invisible to another — check user_id before concluding that a search or an id lookup is broken. "+
			"fields:[\"user_id\",\"device\"] is the cheap way to ask. "+
			"Recommended first call: use this before gkill_get_kyous to understand the data organization, visible tags, and board names. "+
			"Response fields: tag_struct (tag parent-child hierarchy with check_when_inited, is_force_hide, children), mi_board_struct (task board hierarchy), rep_struct (repository hierarchy — this is the tree the web settings screen saves, so it is null until someone has pressed Apply there at least once; an account used only through MCP or the CLI will always see null, and that is not an error. For the actual list of repositories, call gkill_get_rep_infos instead), rep_type_struct (repository type hierarchy), device_struct (device hierarchy), kftl_template_struct (KFTL templates), mi_default_board (default board name, e.g. \"Inbox\"), show_tags_in_list (boolean). "+
			"Note that display labels in this config may not map 1:1 to accepted rep_types query values — canonical query values come from gkill_get_rep_infos. "+
			"The full config is large; narrow with fields (e.g. fields:[\"tag_struct\"]) and contains, keep compact on (default: default-valued node fields are omitted), and the response is capped by max_size_mb (over-sized struct fields are replaced by {omitted_bytes} with a warning).",
		schema(jsonobj.Obj(
			"locale_name", jsonobj.Obj(
				"type", "string",
				"description", "Locale, e.g. ja/en.",
			),
			"fields", jsonobj.Obj(
				"type", "array",
				"items", jsonobj.Obj(
					"type", "string",
					"enum", jsonobj.Strings("user_id", "device", "tag_struct", "mi_board_struct", "rep_struct", "rep_type_struct", "device_struct", "kftl_template_struct", "mi_default_board", "show_tags_in_list"),
				),
				"description", "Return only these fields. Default: all.",
			),
			"include_ui_state", jsonobj.Obj(
				"type", "boolean",
				"default", false,
				"description",
				"When false (default), transient tree-editor keys (is_checked, indeterminate, key, seq, seq_in_parent, "+
					"is_open_default, parent_folder_id, id) are stripped from struct nodes. check_when_inited and is_force_hide are always kept.",
			),
			"compact", jsonobj.Obj(
				"type", "boolean",
				"default", true,
				"description",
				"When true (default), struct nodes omit their default-valued fields: an absent children means no children, "+
					"an absent is_dir means false (a leaf), an absent ignore_check_rep_rykv means false, and an absent name means "+
					"the name equals the node's identity field (rep_name / tag / device / rep_type / board_name). "+
					"check_when_inited and is_force_hide are never omitted. Pass false for the raw tree.",
			),
			"contains", jsonobj.Obj(
				"type", "string",
				"description",
				"Keep only tree leaves whose name / rep_name / tag / device / rep_type / board_name contains this text "+
					"(case-insensitive); folders left without a matching leaf are dropped. Omit for the whole tree.",
			),
			"max_size_mb", jsonobj.Obj(
				"type", "number",
				"default", DefaultKyousMaxSizeMB,
				"description",
				fmt.Sprintf("Cap on the response size in MB (default %s). Trees cannot be paged, so when the ", jsonobj.MarshalString(DefaultKyousMaxSizeMB))+
					"response would exceed it the largest struct fields are replaced by {omitted_bytes} until it fits, and "+
					"warnings[] says which — narrow with fields / contains, or raise the cap.",
			),
		), nil),
	),
	tool(
		"gkill_get_rep_infos",
		// 列を削る fields に加えて、行を削る writable_only / rep_types / rep_names / contains を持つ。
		// 本番では fields で rep_infos[] を落としても、Archived Git の rep 名・歴代端末の GPSLogs_ / Tag_ / Text_
		// だけで数百行あり、「gkill_add_tag はどこへ書くか」を知るために全部読むことになっていた
		// （2026-09-18 の実利用報告）。絞り込みは MCP 側（gkill は4配列を丸ごと返す）。
		"List repositories: rep_infos[] ({rep_name, rep_type, use_to_write, indexed_at} — the kyou repositories; rep_name is a "+
			"query.reps value, rep_type a query.rep_types value, use_to_write whether it is the write target for its type, "+
			"indexed_at when its index was last refreshed), canonical_rep_types[] (the exact strings query.rep_types accepts — "+
			"the vocabulary, not an inventory), plugins[] ({rep_name, data_type, plugin_name} — values for query.reps / data_types; "+
			"plugins that emit no kyou are absent by design) and attached_data_reps[] ({rep_name, data_kind, use_to_write} — where "+
			"tags, texts, notifications and GPS logs are stored; NOT query.reps values, passing one there silently matches nothing). "+
			"rep_infos[] and attached_data_reps[] run to hundreds of rows on an account with a long device history, so narrow: "+
			"\"where does gkill_add_tag write?\" is writable_only:true, data_kinds:[\"tag\"]; the vocabulary alone is "+
			"fields:[\"canonical_rep_types\"]. Details: gkill_get_mcp_help topic:rep.",
		schema(jsonobj.Obj(
			"locale_name", jsonobj.Obj("type", "string", "description", "Locale, e.g. ja/en."),
			"fields", jsonobj.Obj(
				"type", "array",
				"items", jsonobj.Obj(
					"type", "string",
					"enum", jsonobj.Strings("rep_infos", "canonical_rep_types", "plugins", "attached_data_reps"),
				),
				"description",
				"Return only these top-level fields. Default: all. "+
					"fields:[\"canonical_rep_types\",\"plugins\",\"attached_data_reps\"] skips the large rep_infos[] list.",
			),
			"writable_only", jsonobj.Obj(
				"type", "boolean",
				"default", false,
				"description",
				"Keep only rows with use_to_write:true in rep_infos[] and attached_data_reps[] (the current write target per "+
					"type / per attached-data kind). plugins[] becomes [] because plugins are never write targets.",
			),
			"rep_types", jsonobj.Obj(
				"type", "array",
				"items", jsonobj.Obj("type", "string"),
				"description",
				"Keep only rep_infos[] rows with one of these rep_type values. Checked against canonical_rep_types[]; an unknown "+
					"value is rejected rather than silently matching nothing. No effect on the other lists.",
			),
			"rep_names", jsonobj.Obj(
				"type", "array",
				"items", jsonobj.Obj("type", "string"),
				"description",
				"Keep only rows whose rep_name is exactly one of these (case-sensitive), across rep_infos[] / plugins[] / attached_data_reps[].",
			),
			"contains", jsonobj.Obj(
				"type", "string",
				"description",
				"Keep only rows whose rep_name contains this text (case-insensitive), across rep_infos[] / plugins[] / "+
					"attached_data_reps[] — the same rule as gkill_get_all_rep_names.",
			),
			"data_kinds", jsonobj.Obj(
				"type", "array",
				"items", jsonobj.Obj(
					"type", "string",
					"enum", jsonobj.Strings("tag", "text", "notification", "gpslog"),
				),
				"description",
				"Narrow attached_data_reps[] to these data_kind values (e.g. data_kinds:[\"tag\"] to see where gkill_add_tag writes). "+
					"Unknown values are rejected rather than silently matching nothing. No effect on the other lists.",
			),
		), nil),
	),
	tool(
		"gkill_get_idf_file",
		"Retrieve the content of an IDF (file/image/video/audio) entry found by gkill_get_kyous, given the rep_name and "+
			"file_name from its payload, as base64. For images the bytes come back ONLY as an MCP image content block "+
			"(structuredContent carries image_content_attached:true instead of the base64) — that block is what puts the "+
			"picture in front of the model and lets it serve as a reference image, and this tool is its only producer. "+
			"Capped by GKILL_MCP_MAX_FILE_BYTES (default 8MB): pass thumb to downscale (and is_video:true to grab a frame "+
			"from a video). On stdio clients prefer the payload's file_path (no base64, no cap); file_url is a link to hand "+
			"a human, never fetched for you. Response fields: file_name, mime_type, file_size_bytes, is_image, thumb (echoed "+
			"only when downscaled), file_content_base64. Details: gkill_get_mcp_help topic:idf.",
		schema(jsonobj.Obj(
			"rep_name", jsonobj.Obj(
				"type", "string",
				"description", "Repository name from the IDF payload's rep_name field.",
			),
			"file_name", jsonobj.Obj(
				"type", "string",
				"description", "File name from the IDF payload's file_name field.",
			),
			"thumb", jsonobj.Obj(
				"type", "string",
				"description",
				"Return a downscaled JPEG instead of the original, sized \"<width>x<height>\" (e.g. \"1024x1024\"). "+
					"At most 1024 per side. Reach for this when you only need to look at the picture, or when the "+
					"original exceeds the size cap. Files that are neither images nor (with is_video) videos come "+
					"back unchanged.",
			),
			"is_video", jsonobj.Obj(
				"type", "boolean",
				"description",
				"Extract a frame from a video and return that as the thumbnail. Requires thumb. Without it a "+
					"video is fetched whole, which normally blows the size cap.",
			),
			"locale_name", jsonobj.Obj(
				"type", "string",
				"description", "Locale, e.g. ja/en.",
			),
		), []string{"rep_name", "file_name"}),
	),
	tool(
		"gkill_get_kyou_history",
		"Read every stored version of ONE entry, including versions that are soft-deleted. "+
			"gkill is append-only: an update adds a new version and a delete adds a version with is_deleted=true. "+
			"Ordinary searches (gkill_get_kyous) only ever return the LATEST version of an entry, and by default only "+
			"entries that are not deleted. query.include_deleted_data:true does surface deleted entries there, but still "+
			"only their latest version — no query flag reaches earlier versions. This tool is the only way to see what an "+
			"entry used to say, or to read back something you deleted by mistake. "+
			"Requires both the id and its data_type: the lookup is per-type, and there is no safe type-agnostic fallback. "+
			"Response fields: id, data_type, latest_is_deleted (whether the newest version is deleted — check this first "+
			"when an entry has vanished from search results), version_count, returned_count, has_more, and versions[] "+
			"newest first, each with update_time, is_deleted, update_app, update_device, update_user, rep_name and the "+
			"type-specific fields. On a server that exposes write tools, gkill_restore_kyou brings a deleted entry back "+
			"(this read-only server does not have it). "+
			"Caveat: update_time is stored at one-second resolution, so two versions written within the same second "+
			"collapse into one and a history may be missing a version.",
		schema(jsonobj.Obj(
			"id", jsonobj.Obj(
				"type", "string",
				"description", "ID of the entry. Obtain from gkill_add_* responses or from gkill_get_kyous results.",
			),
			"data_type", jsonobj.Obj(
				"type", "string",
				"description",
				"Data type of the entry. Must match the actual type — a mismatch reports the entry as not found. "+
					"Two vocabularies exist: search results and add_* / update_* responses carry PROJECTION names (mi_create / mi_check / mi_limit / mi_start / mi_end, mirekyou_*, timeis_start / timeis_end), while this parameter is the ENTITY type (mi / mirekyou / timeis). Projection names are accepted here and folded to their entity type, so a data_type copied straight out of a response works.",
				"enum", stringsToAny(EntityAndProjectionDataTypeValues),
			),
			"limit", jsonobj.Obj(
				"type", "integer",
				"description", fmt.Sprintf("Max versions to return, newest first (1-%d). Default: %d. Histories are unbounded — every edit appends one; read further with offset.", MaxKyouHistoryLimit, DefaultKyouHistoryLimit),
				"default", DefaultKyouHistoryLimit,
				"minimum", 1,
				"maximum", MaxKyouHistoryLimit,
			),
			"offset", jsonobj.Obj(
				"type", "integer",
				"description",
				"Number of versions to skip from the newest. When has_more is true, pass the response's next_offset here to "+
					"read the older versions. Default: 0.",
				"default", 0,
				"minimum", 0,
			),
			"locale_name", jsonobj.Obj("type", "string", "description", "Locale, e.g. ja/en."),
		), []string{"id", "data_type"}),
	),
}

// buildKyousProperties は gkill_get_kyous のトップレベル引数（順序固定）。
func buildKyousProperties() *jsonobj.Object {
	props := jsonobj.New()
	props.Set("query", FindQuerySchema)
	props.Set("locale_name", jsonobj.Obj(
		"type", "string",
		"description", "Locale, e.g. ja/en.",
	))
	props.Set("limit", jsonobj.Obj(
		"type", "integer",
		"description", fmt.Sprintf("Max number of entries to return (1-1000). Default: %d.", DefaultKyousLimit),
		"default", DefaultKyousLimit,
		"minimum", 1,
		"maximum", 1000,
	))
	props.Set("cursor", jsonobj.Obj(
		"type", "string",
		"description",
		"Opaque pagination cursor. Pass next_cursor from the previous response verbatim — do not construct or edit it "+
			"(v2 cursors are composite time+ID tokens; plain ISO-8601 datetimes from older clients are still accepted). "+
			"Responses with a cursor omit total_count (use remaining_count); the first page carries total_count.",
	))
	props.Set("max_size_mb", jsonobj.Obj(
		"type", "number",
		"description", fmt.Sprintf("Max size of kyous[] in MB (strict; only a first entry that alone exceeds it is returned anyway, with a warning). Enforced again after plugin bodies are inlined with include_plugin_content — entries pushed over it move to the next cursor page. Default: %s.", jsonobj.MarshalString(DefaultKyousMaxSizeMB)),
		"default", DefaultKyousMaxSizeMB,
	))
	props.Set("is_include_timeis", jsonobj.Obj(
		"type", "boolean",
		"description", fmt.Sprintf("Attach the TimeIs (playing) entries that were running when each record was created, as timeis[] ({id, title, tags, start_time, end_time — absent while still running}). Default: %t. Expensive in a way limit does not bound (every call reads the whole TimeIs history and ignores query.reps / the calendar range), so leave it off unless you need it. Does not filter TimeIs-type kyous out of the results. Details: gkill_get_mcp_help topic:search.", DefaultKyousIncludeTimeIs),
		"default", DefaultKyousIncludeTimeIs,
	))
	// include_id / include_rep_name（v2 で廃止。id / rep_name は常時付与）はここに載せない。
	// 受理は NormalizeKyouArgs が続けるが、公開スキーマに載せると AI が
	// 「false なら id が消えるのか」と考える余地を作るだけになる（ADR-0620）。
	props.Set("count_only", jsonobj.Obj(
		"type", "boolean",
		"default", false,
		"description",
		"Return only total_count (no kyous[], no payloads) — the cheapest way to size a query. "+
			"Cannot be combined with cursor or group_by (group_by already returns counts only).",
	))
	props.Set("group_by", jsonobj.Obj(
		"type", "string",
		"enum", jsonobj.Strings("month", "day", "week_of_day", "hour", "data_type", "rep_name", "create_app", "update_app", "url_domain", "file_extension"),
		"description",
		"Aggregate server-side and return buckets:[{key,count}] plus total_count instead of kyous[]. "+
			"Time keys use the server's local timezone; url_domain covers only urlog and file_extension only idf entries; "+
			"at most 1000 buckets. Prefer this over hand-made windows (inclusive calendar bounds double-count the boundary day). "+
			"Cannot be combined with cursor or count_only.",
	))
	props.Set("data_types", jsonobj.Obj(
		"type", "array",
		"items", jsonobj.Obj("type", "string"),
		"description",
		"Allowlist of data_type strings as they appear in results (e.g. [\"nlog\"], plugin types like [\"claude_conversation\"]). "+
			"The entity names timeis / mi / mirekyou are accepted too and expand to all their projections. "+
			"Unknown values produce warnings, not errors. Filtering on Mi projection names without query.for_mi returns "+
			"far fewer rows than the number of tasks (they collapse to one representative) — the response warns. "+
			"Details: gkill_get_mcp_help topic:data_types. null/omitted = no filter, [] = match nothing.",
	))
	props.Set("create_apps", jsonobj.Obj(
		"type", "array",
		"items", jsonobj.Obj("type", "string"),
		"description",
		"Allowlist of the app that WROTE each entry (matched against create_app): \"gkill\" (web UI and uploads), "+
			"\"gkill_kftl\" (the notepad AND gkill_submit_kftl), \"gkill_wear\", \"gkill_mcp_readwrite\" / \"gkill_mcp_write\" "+
			"(these MCP servers), \"urlog_bookmarklet\", \"git\", \"idf\" (indexed files), \"gkill_autolog\", or a plugin's value; "+
			"group_by:\"create_app\" lists the values actually present. null/omitted = no filter, [] = match nothing.",
	))
	props.Set("update_apps", jsonobj.Obj(
		"type", "array",
		"items", jsonobj.Obj("type", "string"),
		"description",
		"Same as create_apps but for the app that last UPDATED the entry (matched against update_app). "+
			"Use this to find entries edited through a particular client rather than created by it. "+
			"null/omitted = no filter, [] = match nothing.",
	))
	props.Set("num_min", jsonobj.Obj(
		"type", "number",
		"description",
		"Lower bound (inclusive) on the numeric payload value: nlog amount, kc num_value, lantana mood; other kinds "+
			"are excluded. The comparison ignores units (yen, step counts and a 0-10 mood share one axis), so pair it "+
			"with data_types — the response warns when kinds mix.",
	))
	props.Set("num_max", jsonobj.Obj(
		"type", "number",
		"description", "Upper bound (inclusive). See num_min.",
	))
	props.Set("idf_kinds", jsonobj.Obj(
		"type", "array",
		"items", jsonobj.Obj("type", "string", "enum", jsonobj.Strings("image", "video", "audio", "zip", "other")),
		"description",
		"Filter idf (file) entries by kind. When set, non-idf entries are excluded from results. "+
			"null/omitted = no filter, [] = match nothing.",
	))
	props.Set("include_file_size", jsonobj.Obj(
		"type", "boolean",
		"default", false,
		"description",
		"Add file_size (bytes, from the filesystem) to idf payloads on the returned page. "+
			"Missing when the file cannot be stat-ed.",
	))
	props.Set("include_plugin_content", jsonobj.Obj(
		"type", "boolean",
		"description",
		"Inline the body of plugin kyous (payload.kind='plugin') into this response instead of one follow-up call "+
			"per entry. Default: false. Adds content_status ('ok' | 'truncated' | 'skipped' | 'error') and content_text "+
			"per plugin payload; only 'ok' means the body is complete. "+
			fmt.Sprintf("At most %d plugin kyous and %d characters per call. ", MaxInlinePluginContentKyous, InlinePluginContentTotalTextLength)+
			"Details: gkill_get_mcp_help topic:plugin.",
		"default", false,
	))
	props.Set("plugin_content_max_text_length", jsonobj.Obj(
		"type", "integer",
		"description",
		"Max characters of inlined text per plugin kyou. Only used when include_plugin_content is true. "+
			fmt.Sprintf("Default: %d, max: %d. ", DefaultInlinePluginContentMaxTextLength, MaxPluginContentMaxTextLength)+
			"Longer bodies are cut and content_status becomes 'truncated'. Raising this reduces how many entries fit "+
			"the shared total-text budget, so raise it only when fetching a small number of long records.",
		"default", DefaultInlinePluginContentMaxTextLength,
	))
	props.Set("plugin_content_format", jsonobj.Obj(
		"type", "string",
		"description",
		"Format of the inlined plugin body. Only used when include_plugin_content is true. "+
			"'text' (default) converts the plugin's HTML to plain text into content_text, 'html' puts the raw HTML "+
			"into content_html, 'both' fills both. Prefer 'text': plugin content HTML is mostly presentation CSS/JS.",
		"enum", jsonobj.Strings("text", "html", "both"),
		"default", DefaultPluginContentFormat,
	))
	props.Set("include_attached_ids", jsonobj.Obj(
		"type", "boolean",
		"description",
		"Also return tag_entities[] / text_entities[] ({id, value}) per entry — the annotation's OWN id, needed only "+
			"to edit or delete a tag / text (gkill_update_text, gkill_delete_kyou data_type:\"tag\"/\"text\"). Default: false, "+
			"because they duplicate tags[] / texts[] on every entry.",
		"default", false,
	))
	props.Set("include_file_urls", jsonobj.Obj(
		"type", "boolean",
		"description",
		"HTTP clients only: mint expiring public links (file_url / file_url_full, with file_url_expires_at) for the idf "+
			"entries in this page, to hand to a human. Default: false — leave it off unless you intend to hand someone a link; "+
			"the model cannot fetch them (use gkill_get_idf_file to look at a file). No effect on stdio clients.",
		"default", false,
	))
	return props
}

// findTool は name のツール定義を返す。無ければ nil。
func findTool(tools []*jsonobj.Object, name string) *jsonobj.Object {
	for _, tool := range tools {
		if n, _ := tool.String("name"); n == name {
			return tool
		}
	}
	return nil
}
