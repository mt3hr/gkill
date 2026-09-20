package mcp

// gkill_get_mcp_help が返す詳しい案内。3サーバ共通（旧 help-topics.mjs）。
//
// ツール一覧（tools/list）の説明文はクライアントのセッション開始時に丸ごとコンテキストへ載り、
// 事故対策を足すたびに太っていった（2026-09-14 時点で readwrite 約 94KB、gkill_get_kyous 単体 27KB、
// gkill_submit_kftl の説明文だけで 7.5KB）。AI がタスクに要る説明だけを読めるよう、
// 説明文は「何をするか・まず使う引数・必ず確認すること・詳細の在処」の要約にとどめ、
// 本文はここへ移した（ADR-0622）。**説明文へ事故対策を書き足したくなったら、まずここへ足すこと。**
// tools/list のバイト量は tool_schema_budget.json が固定し、増加は予算テストが止める。
//
// 本文は英語（説明文と同じ）。ここに書けるツール名は、その topic を読むサーバに載っていなくてもよい
// （本文は「今これを呼べ」ではなく参照情報。tool_handlers_test.go の走査対象は説明文だけ）が、
// 実在しないツール名は help_topics_test.go が落とす。

import (
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

const searchTopic = "gkill_get_kyous searches life-log entries (kyou) and returns enriched results: each entry carries id, rep_name, " +
	"data_type, related_time, create_app / update_app (the app that wrote / last updated it), tags[], texts[], " +
	"notifications[], timeis[] (attached TimeIs when is_include_timeis is set), is_deleted (present ONLY on " +
	"soft-deleted entries, i.e. with query.include_deleted_data; absent means the entry is live) and payload " +
	"(type-specific fields). Pass include_attached_ids:true to also get tag_entities[] / text_entities[] " +
	"({id, value}: the annotation's OWN id, which is what gkill_update_text and gkill_delete_kyou(data_type:\"tag\"/\"text\") " +
	"require — the plain tags[] / texts[] strings carry no id and cannot be edited or deleted); they are off by default " +
	"because they duplicate tags[] / texts[] on every entry.\n\n" +
	"Response fields: kyous[], total_count (only on cursor-less responses), returned_count, remaining_count, has_more, " +
	"next_cursor, buckets (group_by only), partial, warnings, plugins, plugin_content.\n\n" +
	"ALWAYS inspect warnings[], even when partial is false. partial only says that some attached data (tags / texts / " +
	"notifications / TimeIs) could not be fetched. A repository that failed to load is reported in warnings[] instead, " +
	"and its records are simply absent — do not put a repository named by that warning back into query.reps, it would " +
	"narrow the search to the repositories that ARE available and look like zero results. Unknown filter values " +
	"(rep_types / tags / reps / data_types typos, ids that match nothing) are also reported in warnings[] rather than " +
	"silently matching nothing.\n\n" +
	"Query semantics: a filter activates simply by being present and non-null; omit (or pass null for) filters you " +
	"don't use. [] means 'filter enabled but matches nothing' for tags / hide_tags / reps / rep_types / ids / " +
	"timeis_tags / period_of_time_week_of_days. Two exceptions: words / not_words: [] apply NO keyword condition — " +
	"everything passes and the result is NOT narrowed (so an empty keyword list returns the same count as no keyword " +
	"at all; pass at least one word to filter) — and timeis_words: [] means 'only kyous covered by any TimeIs'. " +
	"Most used fields: calendar_start_date / calendar_end_date (both INCLUSIVE; a date-only calendar_start_date " +
	"expands to 00:00:00 local and a date-only calendar_end_date to 23:59:59 local), words, tags, for_mi. Advanced: " +
	"map_latitude / map_longitude / map_radius (all three required, radius in METERS; a partial set is rejected " +
	"because gkill would otherwise silently drop the location condition), playing_time, period_of_time_*, update_time. " +
	"The server always applies only_latest_data=true. Results come back newest first by related_time, ties by id " +
	"ascending.\n\n" +
	"Common query patterns: date range {calendar_start_date:\"2026-03-01\", calendar_end_date:\"2026-03-07\"}; " +
	"keyword {words:[\"keyword\"]}; tag {tags:[\"tagname\"]}; tasks {for_mi:true, mi_check_state:\"uncheck\", " +
	"include_create_mi:true} (for_mi needs at least one include_*_mi flag or it returns nothing — see topic mi).\n\n" +
	"Recommended filtering strategy: fetch ApplicationConfig and all tag names first, then build a visible-tag " +
	"allowlist — a tag is visible when is_force_hide=false AND check_when_inited=true in ApplicationConfig tag_struct. " +
	"Hidden tags can still be searched intentionally by passing them directly in query.tags or query.timeis_tags. " +
	"rep_types values are backend-specific and case-sensitive: ApplicationConfig display labels do not map 1:1 to " +
	"accepted values (files/images live under \"directory\", not \"idf\"), so take them from gkill_get_rep_infos " +
	"canonical_rep_types[] instead of guessing (see topic rep).\n\n" +
	"If a query fails, first retry with fewer query fields, a smaller limit and is_include_timeis=false; then add " +
	"rep_types or TimeIs expansion back step by step. is_include_timeis is expensive in a way limit does not bound: " +
	"every call reads the whole TimeIs history and the attachment ignores query.reps / rep_types / the calendar " +
	"range, so leave it off unless you need to know which stamp was running when each record was created. A stamp " +
	"with no end_time is still running by definition, so it covers every record after its start — an old stamp " +
	"someone forgot to close attaches to everything since, and that is data to clean up, not a bug.\n\n" +
	"Payload shapes (payload.kind is coarser than data_type: Mi, MiReKyou and TimeIs each surface under several " +
	"data_type values that share one kind): kmemo -> 'kmemo' (content; texts[] is a separate list of annotations, " +
	"not the body); kc -> 'kc' (title, num_value); lantana -> 'lantana' (mood 0-10); nlog -> 'nlog' (title, shop, " +
	"amount); urlog -> 'urlog' (title, url, description); git_commit_log -> 'git_commit_log' (commit_message, " +
	"addition, deletion; the entry's id IS the full commit hash); idf -> 'idf' (file_name, is_image, is_video, is_audio, is_zip, rep_name, mime_type — see " +
	"topic idf); timeis_start / timeis_end -> 'timeis' (title, start_time, end_time); mi_* -> 'mi' (title, is_checked, " +
	"board_name, create_time, limit_time, estimate_start_time, estimate_end_time); mirekyou_* -> 'mirekyou' (an " +
	"existing entry turned into a task: target_id plus the same scheduling fields, no title of its own — pass " +
	"target_id to query.ids to read the entry it points at); rekyou -> 'rekyou' (a repost: target_id only). " +
	"Plugin-provided entries have payload.kind='plugin' (see topic plugin).\n\n" +
	"plugins[] at the top level lists {rep_name, plugin_name, description} once per plugin that appears in the " +
	"response (the per-entry payload carries only rep_name / plugin_name). Note gkill_get_rep_infos also returns a " +
	"field called plugins[] with a different shape."

const paginationTopic = "Pagination: pass next_cursor from the previous response verbatim as cursor — do not construct or edit it " +
	"(cursors are composite time+id tokens). The first page carries total_count; responses with a cursor omit " +
	"total_count, so read the remaining volume from remaining_count, which decreases by exactly returned_count " +
	"from page to page. Every entry appears exactly once across the pages, including tasks (Mi), which the server " +
	"re-validates against the full query window on every cursor page so that a task already returned under one " +
	"projection cannot come back under another.\n\n" +
	"limit and max_size_mb are strict caps on what is RETURNED, not on what is SEARCHED: the backend still scans " +
	"every matching repository, so on a large account a broad query with limit:3 takes about as long as one with " +
	"limit:100. To make a query faster, narrow calendar_start_date / calendar_end_date, data_types or reps instead " +
	"of shrinking limit. The one exception to the cap: when the first entry of a page alone exceeds max_size_mb it " +
	"is returned anyway (with a warning) so that paging keeps progressing. With include_plugin_content the cap is " +
	"enforced again AFTER the plugin bodies are inlined (they are not stored in gkill, so the backend cannot measure " +
	"them): entries pushed over the budget are held back, remaining_count / has_more / next_cursor are adjusted and " +
	"warnings[] says how many were held back — they come with the next cursor page.\n\n" +
	"Counting and histograms: count_only:true returns only total_count and skips all payload construction — the " +
	"cheapest way to size a query before fetching. group_by:\"month\"|\"day\"|\"week_of_day\"|\"hour\"|\"data_type\"|" +
	"\"rep_name\"|\"create_app\"|\"update_app\"|\"url_domain\"|\"file_extension\" returns buckets:[{key,count}] plus " +
	"total_count instead of kyous[] (time keys use the server's local timezone; url_domain covers only urlog and " +
	"file_extension only idf entries; at most 1000 buckets, overflow folds into \"(other)\"). week_of_day keys are " +
	"the English day names sunday..saturday in that order and hour keys are \"00\"..\"23\"; both list every bucket, " +
	"zero counts included (the query filter period_of_time_week_of_days uses integers instead: 0=sunday..6=saturday). " +
	"Prefer group_by over hand-made windows: calendar bounds are inclusive, so adjacent windows double-count the " +
	"boundary day.\n\n" +
	"Combination rules: count_only and group_by cannot be combined with cursor (they count everything the query " +
	"matches, while a cursor resumes partway through), and count_only cannot be combined with group_by (group_by " +
	"already returns only counts). All three combinations are rejected with an explanation instead of silently " +
	"dropping one side."

const miTopic = "Mi (task) entries surface under FIVE projections, one per timestamp: mi_create (create_time), mi_check " +
	"(the time the record was last updated, which is when it was checked for a checked task), mi_limit " +
	"(limit_time, only tasks with a deadline), mi_start (estimate_start_time) and mi_end (estimate_end_time). " +
	"MiReKyou (an existing entry turned into a task) has the same five as mirekyou_*. related_time follows the " +
	"projection, so take create_time from the payload when you need the creation time.\n\n" +
	"Which projection you see depends on query.for_mi. WITHOUT it, the five collapse to one representative per " +
	"record (mi_start wins, then mi_check), so a plain date search mostly shows mi_check / mi_start; mi_create " +
	"still survives for tasks whose create_time falls in the window while their update_time does not, so it is rare " +
	"but NOT absent — never read its low count as proof that no task was created. WITH for_mi:true the search is " +
	"restricted to Mi and MiReKyou, the data_type follows query.mi_sort_type (mi_create when unset) and the five " +
	"include_*_mi flags select which projections supply rows: they do not narrow an existing result set, and with " +
	"all five false a for_mi search would return ZERO entries — so when for_mi is set without any of them this " +
	"server assumes include_create_mi:true and says so in warnings[]. Set include_check_mi / include_limit_mi / " +
	"include_start_mi / include_end_mi explicitly to look at other timestamps. mi_check_state (\"all\" / " +
	"\"checked\" / \"uncheck\") and mi_board_name narrow further; discover board names with gkill_get_mi_board_list " +
	"(an unknown mi_board_name is reported in warnings[]).\n\n" +
	"mi_sort_type is NOT only a sort order: it decides the timestamp that calendar_start_date / calendar_end_date, " +
	"the time-of-day window and the weekday filter are matched against, and it sets the data_type of the results. " +
	"It only takes effect through the matching include_*_mi projection (limit_time needs include_limit_mi, and so " +
	"on) — otherwise the projection you did enable decides the axis and the value is ignored (the response says so " +
	"in warnings).\n\n" +
	"data_types:[\"mi_create\"] without for_mi returns far fewer rows than the number of tasks actually created " +
	"(the collapse above); the response warns about it. data_types:[\"mi\"] (the entity name) expands to all five " +
	"projections and counts each task once — see topic data_types.\n\n" +
	"Typical task workflow: gkill_get_kyous({query:{for_mi:true, mi_check_state:\"uncheck\", include_create_mi:true}}) " +
	"→ find the task → gkill_update_mi({id, is_checked:true}). Fields like limit_time are ordinary datetime strings."

const dataTypesTopic = "data_type has TWO vocabularies. Search results and add_* / update_* responses carry PROJECTION names: kmemo, " +
	"kc, nlog, lantana, urlog, idf, git_commit_log, rekyou, timeis_start / timeis_end, mi_create / mi_check / " +
	"mi_limit / mi_start / mi_end, mirekyou_create / mirekyou_check / mirekyou_limit / mirekyou_start / " +
	"mirekyou_end, plus whatever data_type each plugin defines (e.g. claude_conversation — list them with " +
	"gkill_get_plugin_list). gkill_delete_kyou / gkill_restore_kyou / gkill_get_kyou_history take ENTITY names " +
	"(kmemo, kc, nlog, lantana, urlog, timeis, mi, mirekyou, rekyou, tag, text, notification); they also accept " +
	"projection names and fold them, so a data_type copied out of a response works there. The responses of " +
	"gkill_get_kyou_history, gkill_delete_kyou, gkill_restore_kyou and the update tools (gkill_update_mi and the " +
	"rest) carry the ENTITY name (a task is \"mi\" there, whichever projection a search showed it under).\n\n" +
	"The data_types filter of gkill_get_kyous is an allowlist matched against the projection names in results. " +
	"Entity names timeis / mi / mirekyou are accepted too and expand to all their projections (timeis → " +
	"timeis_start + timeis_end; mi → the five mi_*), so [\"timeis\",\"mi\",\"idf\"] counts all three kinds. " +
	"Unknown values produce warnings, not errors. This filter is how you separate Mi from MiReKyou projections and " +
	"how you filter plugin records (rep_types cannot). Plugins may reuse a built-in data_type: [\"kc\"] can return " +
	"step counts, account balances and hand-entered measurements at once, because they are different repositories " +
	"wearing the same type — add query.reps to separate them.\n\n" +
	"Numeric bounds: num_min / num_max (inclusive) apply to the numeric payload value — nlog amount, kc num_value, " +
	"lantana mood — and exclude every other kind from the result. The comparison ignores units: yen, step counts " +
	"and a 0-10 mood are measured on one axis, so num_min:7 mixes them. When the result contains more than one " +
	"of those kinds the response warns; pair the bound with data_types (and query.reps for plugin-supplied kc) " +
	"whenever the number means something specific.\n\n" +
	"idf_kinds:[\"image\"|\"video\"|\"audio\"|\"zip\"|\"other\"] narrows idf (file) entries by kind and excludes " +
	"non-idf entries; include_file_size:true adds file_size (bytes) to idf payloads. create_apps / update_apps " +
	"filter on the app that wrote / last updated an entry: known values are \"gkill\" (web UI and uploads), " +
	"\"gkill_kftl\" (the notepad AND gkill_submit_kftl, which the server stamps with the same value), " +
	"\"gkill_wear\", \"gkill_mcp_readwrite\" / \"gkill_mcp_write\" (the MCP servers), \"urlog_bookmarklet\", " +
	"\"git\", \"idf\" (files indexed from the filesystem), \"gkill_autolog\" (the automatic activity logger) and " +
	"whatever a plugin sets; the values actually present in your data are listed by group_by:\"create_app\". " +
	"Records written through gkill_submit_kftl carry \"gkill_kftl\" and are " +
	"indistinguishable from hand-typed notepad entries, so create_apps:[\"gkill_mcp_readwrite\"] does NOT find them."

const pluginTopic = "Plugin-provided entries (any data_type that is not one of the built-ins) have payload.kind='plugin' carrying " +
	"plugin_name only — the entry's own id / rep_name / data_type identify it. Their body is NOT stored in gkill, so set " +
	"include_plugin_content:true on the same gkill_get_kyous call to get it inline: each plugin payload then " +
	"gains content_status ('ok' | 'truncated' | 'skipped' | 'error'), content_text (and content_html when " +
	"plugin_content_format includes html), content_skipped_reason ('max_kyous' | 'budget' | 'deadline' | " +
	"'rep_error') when it was skipped, and content_error when the fetch failed. Only content_status='ok' means the " +
	"body is complete.\n\n" +
	"Budget: at most 20 plugin kyous per call are inlined and 200000 characters in total; the per-entry cap is " +
	"plugin_content_max_text_length (default 4000, max 200000 — raising it reduces how many entries fit the shared " +
	"total, so raise it only when fetching a small number of long records; to read one long body in full, narrow " +
	"the query to that single entry with query.ids). plugin_content_format is 'text' (default: the plugin's HTML " +
	"converted to plain text), 'html' or 'both'; prefer 'text', the HTML is mostly presentation CSS/JS. Enable " +
	"inlining only when you actually intend to read plugin bodies: it costs one extra request per plugin kyou, and " +
	"requests to the same plugin are serialized, so more entries mean a longer wait, not more parallelism.\n\n" +
	"Which plugins exist, whether each one is alive and whether its index has ever been built: gkill_get_plugin_list " +
	"(emits_kyou / provides / typed_index). A plugin that emits no kyou (a GPS-only plugin, say) never matches " +
	"query.reps or data_types — read its data with gkill_get_gps_log instead. Plugins are matched via query.reps or " +
	"data_types, never rep_types; the rep names that query.reps accepts are gkill_get_plugin_list rep_names[] " +
	"(always present; [] until the plugin's index is built) or gkill_get_rep_infos plugins[] — NOT the plugin's " +
	"rep_name, which is its manifest label (a plugin wrapping several archived Git repositories names each repository " +
	"in rep_names while its rep_name matches nothing; the response warns when you pass it). gkill_get_plugin_list " +
	"does not count entries per plugin (that would round-trip to every plugin in " +
	"series); count with count_only:true plus data_types:[<plugin data_type>] instead. The write-only server has no " +
	"gkill_get_kyous, so plugin bodies can only be read from the read or readwrite server."

const idfTopic = "IDF entries (data_type 'idf') are files: images, videos, audio, zip/cbz and others. The payload carries " +
	"file_name, rep_name, mime_type, is_image / is_video / is_audio / is_zip, plus one of three ways to reach the " +
	"bytes, chosen by transport:\n\n" +
	"1. file_path (stdio clients only): the absolute path on the machine the MCP server runs on. Read it from the " +
	"filesystem directly — no base64, no size cap. Never shown to HTTP clients.\n" +
	"2. gkill_get_idf_file(rep_name, file_name): the file body as base64 (structuredContent.file_content_base64). " +
	"For images the bytes are delivered ONLY as an MCP image content block, and structuredContent carries " +
	"image_content_attached:true instead of the base64 — that block is what puts the picture in front of the model " +
	"and lets it be used as a reference image for image generation, and this tool is its only producer. The tool is " +
	"capped by GKILL_MCP_MAX_FILE_BYTES (default 8MB); pass thumb:\"<width>x<height>\" (at most 1024 per side, e.g. " +
	"\"1024x1024\") to get a downscaled JPEG instead of the original, and is_video:true alongside thumb to grab a " +
	"frame out of a video (a video fetched whole normally blows the cap). The response echoes thumb only when a " +
	"downscaled version was returned — its absence means you got the original.\n" +
	"3. file_url / file_url_full (HTTP clients only, and only when the gkill_get_kyous call passed " +
	"include_file_urls:true): expiring public links minted by the MCP server (images: file_url is a downscaled " +
	"thumbnail, file_url_full the original, no size cap), with file_url_expires_at saying when they stop working " +
	"(default one hour). These are links to hand a human — paste them in a reply, open them in a browser, embed them " +
	"in HTML. MCP never fetches them for you, and a client that can fetch URLs on its own still ends up with bytes " +
	"outside the conversation rather than a picture it can look at. Leave include_file_urls off unless you intend to " +
	"hand a link to a human: every link is a minted token.\n\n" +
	"Files dropped into a repository directory do not appear in searches until the index is refreshed, and nothing " +
	"warns you: gkill_get_rep_infos rep_infos[].indexed_at tells you when that last happened; pass " +
	"query.update_cache:true (an operational side effect, not free) or run the update_cache CLI to refresh. Narrow " +
	"file searches with idf_kinds and add file_size with include_file_size (see topic data_types)."

const deletedTopic = "gkill is append-only: an update adds a new version of an entry and a delete adds a version with " +
	"is_deleted=true. Ordinary searches (gkill_get_kyous) only ever return the LATEST version of each entry, and by " +
	"default only entries that are not deleted.\n\n" +
	"query.include_deleted_data:true also returns soft-deleted entries (they carry is_deleted:true; live entries " +
	"carry no is_deleted field at all), but still only " +
	"their latest version. Note that a deleted entry is indexed by the time it was DELETED, not by its original " +
	"related_time, so a calendar range with this flag on also surfaces entries created on other days that merely " +
	"happened to be deleted inside the range. rekyou / mirekyou entries stay hidden even with this flag (their " +
	"repositories filter deleted rows internally), and git_commit_log has no concept of deletion.\n\n" +
	"gkill_get_kyou_history(id, data_type) reads every stored version of ONE entry, newest first, including deleted " +
	"versions — the only way to see what an entry used to say or to read back something deleted by mistake. " +
	"latest_is_deleted tells you first whether the newest version is a deletion; data_type must match the entry's " +
	"actual type (the lookup is per-type, so a wrong data_type looks exactly like a wrong id), and every version in " +
	"the response carries that entity name as its data_type. limit (default 20, max 200) caps one call; when " +
	"has_more is true pass next_offset as offset to read the older versions, however many there are. update_time " +
	"is stored at one-second resolution, so two versions written within the same second collapse into one.\n\n" +
	"gkill_delete_kyou soft-deletes (it refuses an already-deleted entry instead of stacking another version, so a " +
	"retry after an uncertain response is safe; a speculative call on an active entry WILL delete it); " +
	"gkill_restore_kyou undoes a soft delete (and refuses an already-active entry). Both live on servers that " +
	"expose write tools. idf (file) and git_commit_log entries cannot be deleted this way — they are managed by the " +
	"filesystem and git repositories. To undo a gkill_submit_kftl submission, delete only what it CREATED: " +
	"created.filter(c => !c.updated).map(({id, data_type}) => ({id, data_type})) — entries with updated:true are " +
	"pre-existing records it merely updated."

const repTopic = "Two different names describe where an entry lives. rep_type is the KIND of repository and is the vocabulary " +
	"query.rep_types accepts: the exact strings come from gkill_get_rep_infos canonical_rep_types[] (files/images " +
	"live under \"directory\", not \"idf\"; values are case-sensitive; ApplicationConfig display labels do not map " +
	"1:1). canonical_rep_types[] is the vocabulary, not an inventory: every value is listed whether or not this " +
	"account has such a repository, so zero hits after filtering by one is not an anomaly — check rep_infos[] for " +
	"what exists here. rep_name is one concrete repository (one device's kmemo store, one archived git repository) " +
	"and is the value query.reps accepts; gkill_get_all_rep_names lists them (contains / limit narrow the list).\n\n" +
	"gkill_get_rep_infos returns four lists: rep_infos[] ({rep_name, rep_type, use_to_write, indexed_at for " +
	"repositories that keep an index}; a repository appears once per rep_type it supplies, so this list can run to " +
	"several hundred entries), canonical_rep_types[], plugins[] ({rep_name, data_type, plugin_name} — each rep_name " +
	"is a valid query.reps value; plugins that emit no kyou are absent by design) and attached_data_reps[] " +
	"({rep_name, data_kind, use_to_write} — where tags, texts, notifications and GPS logs are stored). " +
	"IMPORTANT: attached_data_reps[] are NOT query.reps values; passing one to query.reps matches no kyou and " +
	"silently returns zero results.\n\n" +
	"use_to_write says whether a repository is the write target for its type. A repository can be listed and " +
	"searchable yet not writable, and then every write of that type fails with a message that does not say why " +
	"(the KFTL ~~ task-from-record line is the usual victim) — check it before writing, not after. To answer " +
	"\"where does gkill_add_tag write?\" call gkill_get_rep_infos with writable_only:true and " +
	"data_kinds:[\"tag\"]; the row filters (writable_only / rep_types / rep_names / contains / data_kinds) and " +
	"the column filter (fields) keep the response small on accounts with a long device history."

const kftlTopic = "KFTL is gkill's line-based text format that creates multiple records from a single text block, submitted " +
	"through gkill_submit_kftl.\n\n" +
	"CRITICAL parsing rules: (1) Text is split by newlines (\\n) and each line is processed independently. " +
	"(2) Prefixes MUST be on their own line with NOTHING else on that line; the prefix line and the data value MUST " +
	"be on SEPARATE lines. For example, '/mood' must be alone on one line and '8' on the next line. Writing " +
	"'/mood 8' on one line is rejected per-line, and so is a prefix with no value line after it. (3) Lines without a " +
	"recognized prefix are treated as kmemo (text memo) content; adjacent non-prefixed lines are merged into a " +
	"single kmemo. (4) To create SEPARATE records, insert a separator line (、 or ,) between them; without separators, " +
	"consecutive lines merge into one kmemo. 、、 or ,, separates AND increments the time by 1 second.\n\n" +
	"Prefix lines (each must be the ENTIRE line):\n" +
	"/mi or ーみ → task in up to five positional lines: title, then board name, estimated start, estimated end, " +
	"deadline (same order as the ~~ block). Write the three datetimes BARE — a leading ? on those lines is rejected " +
	"per-line.\n" +
	"~~ or ～～ → turn the record written just ABOVE into a task (repost task). Opens AND closes with the same ~~ " +
	"marker, and is meaningless on its own — the record it tasks must come first. There is NO title line (the " +
	"original record is shown as-is). Inside the block: board name, estimated start, estimated end, deadline (all " +
	"optional; write the datetimes BARE — a leading ? on one of those lines is rejected per-line, because ?? there " +
	"would silently parse as a broken date). Lines starting with # inside the block become tags on the TASK itself " +
	"and may appear before or after the board name. Use /mi for a brand new task. ~~ can ONLY task a record created " +
	"earlier in THIS SAME submission (its target is the id the previous line just minted) — it cannot reference a " +
	"record that already exists in gkill, and there is no syntax that can. It also needs a mirekyou repository " +
	"configured for the account; without one the whole submission fails and nothing is written.\n" +
	"/mood or ーら → next line is the mood value (0-10).\n" +
	"/expense or ーん → next lines: shop name, then (title/description, amount) pairs repeating — one expense record " +
	"per pair. A #tag line or a -- text block written after an amount line attaches to that one payment only; write " +
	"them after the amount, never before /expense. (IMPORTANT: the prefix is /expense, NOT /nlog.)\n" +
	"/url or ーう → next line is the URL, and the line AFTER that is its title.\n" +
	"/num or ーか → next line is the title, then the line after that is the numeric value.\n" +
	"/start or ーた → next line is the timeis start label.\n" +
	"/end or ーえ → end a running timeis. The NEXT LINE IS REQUIRED and must be the exact title of the running " +
	"timeis to close; omitting it fails with \"打刻終了タイトルを指定してください\".\n" +
	"/timeis or ーち → timeis in three fixed lines: title, then start datetime, then end datetime.\n" +
	"/end? or ーいえ → same as /end but tolerates \"no such running timeis\". The title line is STILL REQUIRED — " +
	"the if-exists part only forgives a missing target, not a missing title.\n" +
	"/endt or ーたえ → end a running timeis by tag. The NEXT LINE IS REQUIRED and holds the tag name(s), separated " +
	"by 、 or , and written WITHOUT the # / 。 prefix.\n" +
	"/endt? or ーいたえ → same as /endt but tolerates \"no such running timeis\"; the tag line is still required.\n" +
	"# or 。 → tag (attach to the previous record). Matched by PREFIX, so a Markdown heading line like \"# Title\" " +
	"becomes a tag.\n" +
	"? or ？ → related time. Also matched by prefix, so any line starting with ? is parsed as a datetime and fails " +
	"if it is not one. A line that is exactly ?? is the repeat block below, not a related time.\n" +
	"?? or ？？ → repeat block: make the record written just above it repeat. Opens AND closes with the same ?? " +
	"marker, and the marker must be the WHOLE line (\"?? fri 3\" is rejected). Inside, up to four positional lines: " +
	"(1) the rule — daily / a weekday such as fri or mon,wed,fri / 6w fri (every 6th Friday) / monthly 15 / 2nd fri / " +
	"last fri, (2) how many times, OR an end date, (3) whether to add even when an equivalent record already exists: " +
	"yes or no, DEFAULT NO, (4) the start point, in any datetime format ? accepts, default now. Lines 3 and 4 can be " +
	"omitted by closing early. Every datetime field of the repeated record is shifted by the SAME number of days, so " +
	"the gaps between estimated start / end / deadline are preserved; the time of day comes from the record's first " +
	"filled datetime field. A task with NO datetime field is rejected — there is nowhere to put the repeated dates. " +
	"Candidates are strictly AFTER the start point, and days that do not exist are SKIPPED rather than rounded " +
	"(monthly 31 skips February, 2nd fri is fine but 5th fri skips months without one). Because line 3 defaults to " +
	"no, submitting the SAME text twice does not create duplicates. Repeating is refused for /start (it would leave " +
	"N never-closed running stamps that then cover every later record) and for /end, /end?, /endt, /endt? (they only " +
	"ever look at the ONE timeis running right now). Inside an /expense block the repeat covers the WHOLE block, so " +
	"2 payments x 3 times = 6 records. At most 1000 records per submission; over that the whole submission is " +
	"rejected rather than truncated.\n" +
	"-- or ーー → text block start/end.\n" +
	"! or ！ → stop processing. On the FIRST line it does nothing (processing continues). The marker line itself " +
	"is NOT a value line: a prefix followed only by ! (e.g. \"/timeis\\n!\") is a per-line error, not an empty record.\n" +
	"(no prefix) → kmemo text content.\n\n" +
	"Example (creates 3 records: kmemo + mood + expense): " +
	"\"今日はいい天気だった\\n、\\n/mood\\n8\\n、\\n/expense\\nカフェ\\nアイスコーヒー\\n-500\\n!\"\n\n" +
	"Response fields: messages[] (server processing messages), created[] ({id, data_type, updated, related_time}) " +
	"— one entry per record actually written, in the order they were written; related_time is stored at one-second " +
	"resolution — and replayed (see idempotency below). Ending a timeis reports the existing record with updated:true " +
	"rather than a new id. Use created[].id as target_id for gkill_add_tag / gkill_add_text. created[] is NOT the " +
	"targets[] shape of gkill_delete_kyou: to undo a submission pass " +
	"created.filter(c => !c.updated).map(({id, data_type}) => ({id, data_type})) — the two extra fields are rejected " +
	"there, and entries with updated:true are pre-existing records the submission merely updated (a timeis it ended), " +
	"so deleting them is not an undo.\n\n" +
	"Failures: a record with no content (a blank kmemo, a task or bookmark with no title, an expense with only a shop " +
	"name, a numeric record with no value), a tag / related time with nothing to attach to, or an unreadable " +
	"schedule date is a per-line INPUT ERROR that rejects the whole submission — it is never silently skipped. " +
	"The whole submission is one database transaction: errors are reported one per bad line (errors[]) and NOTHING is " +
	"written, so the response carries created:[] (an empty array) whether the failure was a bad VALUE caught while " +
	"parsing (a mood outside 0-10, a prefix with no argument) or a failure while writing (a missing write repository, " +
	"a database error — the error names the line it stopped at). Simply fix the text and submit again.\n\n" +
	"Idempotency: pass the same idempotency_key on a retry of an ALREADY SUCCESSFUL submission (e.g. the response was " +
	"lost). The same key with the SAME kftl_text writes nothing and returns the ORIGINAL created[] with replayed:true, " +
	"so the ids of the first submission can be recovered without searching. The same key with DIFFERENT text is " +
	"rejected with 409 (ERR000423) and writes nothing — a key is not a session, use a fresh one for new content. Keys " +
	"are remembered for 10 minutes; a failed submission is never remembered, so retrying it always writes afresh.\n\n" +
	"Provenance: records written through this tool carry create_app=\"gkill_kftl\" (the same value the web notepad " +
	"writes) and create_device set to the SERVER's device name, not \"mcp\". MCP-submitted KFTL is therefore " +
	"indistinguishable from hand-typed notepad KFTL and create_apps:[\"gkill_mcp_readwrite\"] does NOT find them."

// HelpTopic は topic 名と本文。
type HelpTopic struct {
	Name  string
	Title string
	Text  string
}

// HelpTopics は topic の順序つき一覧（index は BuildHelpIndexText が組み立てる）。
var HelpTopics = []HelpTopic{
	{Name: "search", Title: "gkill_get_kyous: query semantics, response fields, warnings, payload shapes", Text: searchTopic},
	{Name: "pagination", Title: "Paging (cursor / remaining_count), limit is not speed, count_only and group_by", Text: paginationTopic},
	{Name: "mi", Title: "Mi tasks: the five projections, for_mi, include_*_mi, mi_sort_type", Text: miTopic},
	{Name: "data_types", Title: "data_type vocabularies, the data_types filter, num_min / num_max, idf_kinds, create_apps", Text: dataTypesTopic},
	{Name: "plugin", Title: "Plugin entries and include_plugin_content (budget, format, content_status)", Text: pluginTopic},
	{Name: "idf", Title: "Files and images: file_path / gkill_get_idf_file / file_url, thumbnails, index freshness", Text: idfTopic},
	{Name: "deleted", Title: "Deleted entries and version history: include_deleted_data, gkill_get_kyou_history, restore", Text: deletedTopic},
	{Name: "rep", Title: "rep_types vs rep names, gkill_get_rep_infos, attached_data_reps, use_to_write", Text: repTopic},
	{Name: "kftl", Title: "KFTL text format for gkill_submit_kftl: every prefix, ~~ and ?? blocks, failures, provenance", Text: kftlTopic},
}

// HelpIndexTopic は topic を省略したときの応答。topic 名は inputSchema の enum にもなる。
const HelpIndexTopic = "index"

// HelpTopicNames は index + 全 topic 名（enum の順序）。
var HelpTopicNames = func() []string {
	out := []string{HelpIndexTopic}
	for _, topic := range HelpTopics {
		out = append(out, topic.Name)
	}
	return out
}()

// HelpTopicByName は topic 名で引く。
func HelpTopicByName(name string) (HelpTopic, bool) {
	for _, topic := range HelpTopics {
		if topic.Name == name {
			return topic, true
		}
	}
	return HelpTopic{}, false
}

// ListHelpTopics は {topic, title} の一覧（index の本体）。
func ListHelpTopics() []any {
	out := make([]any, 0, len(HelpTopics))
	for _, topic := range HelpTopics {
		out = append(out, jsonobj.Obj("topic", topic.Name, "title", topic.Title))
	}
	return out
}

func buildHelpIndexText() string {
	lines := []string{
		"The tool descriptions in this list are summaries. Read the topic you need before a first search, a first " +
			"KFTL submission, or when a response carries warnings you do not understand. Topics:",
	}
	for _, topic := range HelpTopics {
		lines = append(lines, "- "+topic.Name+": "+topic.Title)
	}
	lines = append(lines, "Call gkill_get_mcp_help with topic:<name> to read one. gkill_status reports the account and the "+
		"generation of the tool list you hold (schema_revision).")
	return strings.Join(lines, "\n")
}

// BuildHelpPayload は gkill_get_mcp_help の応答。topic 省略・"index" は一覧。
// 未知の topic は正規化器（NormalizeMcpHelpArgs）が先に弾くので、ここでは index へ倒すだけ。
func BuildHelpPayload(topic string) *jsonobj.Object {
	name := topic
	if name == "" {
		name = HelpIndexTopic
	}
	entry, ok := HelpTopicByName(name)
	if name == HelpIndexTopic || !ok {
		return jsonobj.Obj(
			"topic", HelpIndexTopic,
			"title", "Help index",
			"text", buildHelpIndexText(),
			"topics", ListHelpTopics(),
		)
	}
	return jsonobj.Obj("topic", name, "title", entry.Title, "text", entry.Text)
}
