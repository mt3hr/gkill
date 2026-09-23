package mcp

// gkill_get_kyous.query の JSON Schema（旧 find-query-schema.mjs）。
//
// read / readwrite の2サーバで同一定義を使う。フィルタの活性化は
// 「値フィールドが非nullで存在すること」で決まり、旧 use_X フラグは廃止済み
// (後方互換の受理変換は normalization.go の NormalizeKyouQuery が行う)。
// 廃止済みのキー（only_latest_data / use_X）はこのスキーマには載せない。
// **properties のキー集合は NormalizeKyouQuery の受理集合（KyousQueryAllFields）から
// 廃止済みを引いたものと一致させること** —— schema_contract_test.go が固定する。
// 片方だけ改名すると「tools/list どおりに呼ぶと未知キーで拒否される」になる。

import "github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"

// FindQuerySchema は gkill_get_kyous の query のスキーマ。キーの順序は tools/list のバイト列に効く。
var FindQuerySchema = buildFindQuerySchema()

func buildFindQuerySchema() *jsonobj.Object {
	o := jsonobj.Obj(
		"type", "object",
		// 本文（推奨する絞り込み手順・ペイロード形・idf の読み方・プラグイン）は help topic の
		// search / mi / idf / plugin へ移した（ADR-0622）。ここは活性化の規則だけ。
		"description",
		"gkill find query. A filter activates when its value field is present and non-null; omit (or pass null for) fields "+
			"you don't filter by. [] means 'filter enabled but matches nothing' for tags / hide_tags / reps / rep_types / ids / "+
			"timeis_tags / period_of_time_week_of_days. Two exceptions: words / not_words: [] apply NO keyword condition "+
			"(everything passes — an empty keyword list does not narrow), and timeis_words: [] means 'only kyous covered by any "+
			"TimeIs'. Datetime fields are RFC 3339 strings with a timezone offset (or YYYY-MM-DD where noted). Payload shapes "+
			"per data_type, the Mi / TimeIs projections and how to read idf files: gkill_get_mcp_help topics search, mi and idf.",
	)
	props := jsonobj.New()
	props.Set("update_cache", jsonobj.Obj(
		"type", "boolean",
		"description",
		"Force cache refresh before query. Does NOT change any user data, but it is not free either: "+
			"it triggers a rebuild of the derived caches (I/O and wait time), so a 'read-only' call with this flag "+
			"still has an operational side effect. Leave it off unless a record you know exists is missing from results.",
	))
	props.Set("include_deleted_data", jsonobj.Obj("type", "boolean", "description", "Also return soft-deleted entries (they carry is_deleted:true). Default false. A deleted entry is indexed by the time it was DELETED, not its original related_time, and rekyou / mirekyou stay hidden even with this flag. Reading or restoring one: gkill_get_mcp_help topic:deleted."))
	props.Set("rep_types", jsonobj.Obj(
		"type", "array",
		"description",
		"Allowed rep-type names; omit or pass null for no filter, [] matches nothing. Case-sensitive canonical values come from gkill_get_rep_infos canonical_rep_types[] (files/images are \"directory\") — do not guess from ApplicationConfig labels.",
		"items", jsonobj.Obj("type", "string"),
	))
	props.Set("ids", jsonobj.Obj(
		"type", "array",
		"description",
		"Entry IDs to include (an include-list: only these entries are returned). Omit or pass null for no ID filter, [] matches nothing. Ids that match nothing are reported in warnings[] (the search cannot tell a missing id from a deleted or otherwise excluded one).",
		"items", jsonobj.Obj("type", "string"),
	))
	props.Set("words", jsonobj.Obj(
		"type", "array",
		"description",
		"Keywords to match (case-insensitive substring). Matched against each type's text fields (kmemo content; urlog url/title/description; nlog title/shop/amount; timeis title; kc title/value; mi title/board name; lantana mood value as text; idf file path and .md/.txt body; git commit message; plugin-defined text) and against attached texts; an entry whose ID starts with the keyword also matches (keywords of 7+ characters only, e.g. a pasted UUID or a short git hash). Omit or pass null for no keyword filter. [] applies NO keyword condition — everything passes, the result is NOT narrowed (unlike tags / reps / ids, where [] matches nothing); pass at least one keyword to filter.",
		"items", jsonobj.Obj("type", "string"),
	))
	props.Set("words_and", jsonobj.Obj("type", "boolean", "description", "AND logic for words (true=all must match, false=any). No effect when words is omitted or empty."))
	props.Set("not_words", jsonobj.Obj(
		"type", "array",
		"description",
		"Keywords to exclude: drops entries whose text fields (same fields as words) or attached texts contain any of them. IDs are never matched. With words omitted or empty, the result is everything minus these matches.",
		"items", jsonobj.Obj("type", "string"),
	))
	props.Set("reps", jsonobj.Obj(
		"type", "array",
		"description",
		"Allowed rep names; omit or pass null for no rep filter, [] matches nothing. Use this as an allowlist when you already know the visible repos to include. If rep_struct (from ApplicationConfig) is unavailable, infer hidden repos from unchecked rep_type leaves and keep this list aligned with visible sources only.",
		"items", jsonobj.Obj("type", "string"),
	))
	props.Set("tags", jsonobj.Obj(
		"type", "array",
		"description",
		"Allowed tag names; omit or pass null for no tag filter, [] matches nothing. For ordinary browsing, you may build a visible-tag allowlist from ApplicationConfig. If you intentionally need a hidden tag, you can pass it here directly instead of excluding it from the query.",
		"items", jsonobj.Obj("type", "string"),
	))
	props.Set("hide_tags", jsonobj.Obj(
		"type", "array",
		"description",
		"Explicit tag exclusion list. Prefer a visible-tag allowlist in tags when you need to exclude hidden tags reliably.",
		"items", jsonobj.Obj("type", "string"),
	))
	props.Set("tags_and", jsonobj.Obj("type", "boolean", "description", "AND logic for tags (true=all must match, false=any)."))
	props.Set("timeis_words", jsonobj.Obj(
		"type", "array",
		"description",
		"Keywords to match in TimeIs titles. [] means 'only Kyous covered by any TimeIs' (no keyword constraint). Omit or pass null (with timeis_not_words also absent) for no TimeIs filter.",
		"items", jsonobj.Obj("type", "string"),
	))
	props.Set("timeis_not_words", jsonobj.Obj("type", "array", "description", "Keywords to exclude from TimeIs titles.", "items", jsonobj.Obj("type", "string")))
	props.Set("timeis_words_and", jsonobj.Obj("type", "boolean", "description", "AND logic for timeis_words."))
	props.Set("timeis_tags", jsonobj.Obj(
		"type", "array",
		"description",
		"Allowed TimeIs tag names; omit or pass null for no TimeIs tag filter, [] matches nothing. When set without timeis_words/timeis_not_words, timeis_words: [] is auto-added so the TimeIs filter activates. For ordinary browsing, you may use the same visible-tag allowlist strategy as tags. If you intentionally need a hidden tag, you can pass it here directly.",
		"items", jsonobj.Obj("type", "string"),
	))
	props.Set("timeis_tags_and", jsonobj.Obj("type", "boolean", "description", "AND logic for timeis_tags."))
	props.Set("calendar_start_date", jsonobj.Obj(
		"type", "string",
		"description", "Start of the date-range filter (INCLUSIVE); set to activate. Date-only values expand to 00:00:00 local. "+ISODateTimeDesc+" or "+DateOnlyDesc,
	))
	props.Set("calendar_end_date", jsonobj.Obj(
		"type", "string",
		"description", "End of the date-range filter (INCLUSIVE - adjacent hand-made windows double-count the boundary day; prefer top-level group_by for histograms). Date-only values expand to 23:59:59 local. "+ISODateTimeDesc+" or "+DateOnlyDesc,
	))
	props.Set("map_radius", jsonobj.Obj(
		"type", "number",
		"description",
		"Search radius in METERS (500 = 500 m), greater than 0. The map filter needs all three of map_latitude, map_longitude and map_radius: "+
			"a call with only one or two of them is rejected here, because gkill would otherwise silently ignore the location condition.",
	))
	props.Set("map_latitude", jsonobj.Obj("type", "number", "description", "Center latitude in degrees (-90..90). Requires map_longitude and map_radius too — see map_radius."))
	props.Set("map_longitude", jsonobj.Obj("type", "number", "description", "Center longitude in degrees (-180..180). Requires map_latitude and map_radius too — see map_radius."))
	// 5つのinclude_*_miはMiのSQL射影そのものを選ぶスイッチで、既定(全false)では
	// 検索が0件になる。「絞り込み」ではなく「行の供給源」なので、
	// for_mi=trueのときは最低1つtrueにしないと何も返らないことを明記する。
	props.Set("include_create_mi", jsonobj.Obj("type", "boolean", "description", "Include the created-time projection of Mi tasks (data_type mi_create). Effective only when for_mi=true. The five include_*_mi flags select which projections supply rows; when for_mi is set with none of them, this one is assumed (and warnings[] says so) — set the others explicitly to look at check / deadline / estimate timestamps."))
	props.Set("include_check_mi", jsonobj.Obj("type", "boolean", "description", "Include the checked-time projection (data_type mi_check). Effective only when for_mi=true; see include_create_mi."))
	props.Set("include_limit_mi", jsonobj.Obj("type", "boolean", "description", "Include the deadline projection — only tasks with a limit_time (data_type mi_limit). Effective only when for_mi=true; see include_create_mi."))
	props.Set("include_start_mi", jsonobj.Obj("type", "boolean", "description", "Include the estimated-start projection — only tasks with an estimate_start_time (data_type mi_start). Effective only when for_mi=true; see include_create_mi."))
	props.Set("include_end_mi", jsonobj.Obj("type", "boolean", "description", "Include the estimated-end projection — only tasks with an estimate_end_time (data_type mi_end). Effective only when for_mi=true; see include_create_mi."))
	props.Set("include_end_timeis", jsonobj.Obj("type", "boolean", "description", "Also index ended TimeIs entries by their end time (data_type timeis_end). Default false, so a TimeIs is only found on the day it STARTED — an overnight sleep that began yesterday does not appear in today's calendar range unless you set this to true."))
	props.Set("playing_time", jsonobj.Obj(
		"type", "string",
		"description",
		"Set to search TimeIs entries running at that moment — a point-in-time snapshot of what was happening, unlike calendar range. "+
			"Accepts the literal \"now\" for the current time. "+ISODateTimeDesc+" or "+DateOnlyDesc,
	))
	props.Set("update_time", jsonobj.Obj(
		"type", "string",
		"description", "Filter by last update time (records updated after this time); set to activate. "+ISODateTimeDesc+" or "+DateOnlyDesc,
	))
	props.Set("is_image_only", jsonobj.Obj("type", "boolean", "description", "Return only entries that have images attached."))
	props.Set("for_mi", jsonobj.Obj("type", "boolean", "description", "Restrict the search to task entries — BOTH Mi and MiReKyou. Pair it with the include_*_mi flags that choose the timestamp projection (include_create_mi is assumed when none is given, with a note in warnings[]). Also decides which projection you see (with for_mi: mi_sort_type; without it: one collapsed representative per task). Details: gkill_get_mcp_help topic:mi."))
	props.Set("period_of_time_start_time_second", jsonobj.Obj(
		"type", "integer",
		"description", "Start of time-of-day window, seconds from 00:00:00 (0-86399); set to activate time-of-day filtering.",
	))
	props.Set("period_of_time_end_time_second", jsonobj.Obj(
		"type", "integer",
		"description", "End of time-of-day window, seconds from 00:00:00 (0-86399); set to activate time-of-day filtering.",
	))
	props.Set("period_of_time_week_of_days", jsonobj.Obj(
		"type", "array",
		"description",
		"Weekdays to include: Sunday=0 ... Saturday=6. Omit or pass null for no weekday restriction, [] matches nothing, all 7 days = no restriction.",
		"items", jsonobj.Obj("type", "integer", "minimum", 0, "maximum", 6),
	))
	props.Set("mi_board_name", jsonobj.Obj("type", "string", "description", "Filter Mi tasks by board name; omit or pass null for all boards."))
	props.Set("mi_check_state", jsonobj.Obj(
		"type", "string",
		"description", "Filter Mi tasks by check state.",
		"enum", jsonobj.Strings("all", "checked", "uncheck"),
	))
	props.Set("mi_sort_type", jsonobj.Obj(
		"type", "string",
		"description",
		"Which time of a Mi task to use. NOT only a sort order: it is the timestamp the calendar / time-of-day / "+
			"weekday filters match against and it sets the data_type of the results, so it changes WHICH tasks come back. "+
			"It only takes effect through the matching include_*_mi projection (limit_time needs include_limit_mi, and so on).",
		"enum", jsonobj.Strings("create_time", "estimate_start_time", "estimate_end_time", "limit_time"),
	))
	// only_latest_data（MCP 層が常に true へ強制する）と旧 use_X フラグはここに載せない。
	// NormalizeKyouQuery は今までどおり受理し、届いたら古いスキーマの証拠として警告する（ADR-0620）。
	o.Set("properties", props)
	// 未知キーは NormalizeKyouQuery が拒否する。true のままだと「スキーマ上は何でも通る」と読めて
	// 実際は拒否される不一致になっていた（2026-09-18 の実利用報告）。廃止済みキーは公開しないが受理は続く。
	o.Set("additionalProperties", false)
	return o
}
