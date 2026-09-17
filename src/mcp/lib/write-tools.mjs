// 書き込みツールの定義。write / readwrite の2サーバが共有する。
//
// 以前はサーバごとに逐語コピーされていて、gkill_submit_kftl / gkill_delete_kyou の
// description が接続先サーバによって違っていた。

import { ISO_DATETIME_DESC, DATE_ONLY_DESC, ENTITY_AND_PROJECTION_DATA_TYPE_VALUES } from "./constants.mjs";

export const WRITE_TOOLS = [
  {
    name: "gkill_add_kmemo",
    description:
      "Create a text memo (kmemo) in gkill — the most general-purpose record type for free-form text notes, diary entries, or any textual life-log data. " +
      "The repository where the memo is stored is determined automatically by the server based on user configuration. " +
      "Response fields: added_kmemo (full Kmemo entity with id, rep_name, content, related_time, create_time, etc.), added_kyou (parent Kyou wrapper with id, data_type, related_time). " +
      "The two overlap on purpose: added_kyou is the row shape a timeline shows (id, data_type, related_time, is_image/is_video), added_* is the entity with the fields you just wrote. Use added_*.id as the id everywhere — they are the same id. " +
      "Use the returned id as target_id for gkill_add_tag to categorize the memo, or gkill_add_text to attach additional annotations. " +
      "Typical workflow: create a memo with gkill_add_kmemo → tag it with gkill_add_tag using the returned id. " +
      "If related_time is omitted, defaults to the current timestamp. " +
      "For structured multi-record creation (e.g., memo + mood + expense in one shot), consider gkill_submit_kftl instead.",
    inputSchema: {
      type: "object",
      properties: {
        content: { type: "string", description: "Memo text content. Supports any free-form text including multi-line." },
        related_time: { type: "string", description: `When this memo relates to (not when it was created — that is auto-set). ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["content"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_urlog",
    description:
      "Create a bookmark/URL record (urlog) in gkill for saving web links with optional titles. " +
      "Useful for bookmarking articles, documentation, or any web resource as part of the life-log. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_urlog (full URLog entity with id, url, title, rep_name, related_time, etc.), added_kyou (parent Kyou wrapper). " +
      "The two overlap on purpose: added_kyou is the row shape a timeline shows (id, data_type, related_time, is_image/is_video), added_* is the entity with the fields you just wrote. Use added_*.id as the id everywhere — they are the same id. " +
      "Use the returned id as target_id for gkill_add_tag or gkill_add_text to annotate the bookmark. " +
      "NOTE: saving a bookmark makes the server fetch the URL. When title is omitted the server fills it in from the page's <title>, and it also fetches a favicon and a thumbnail. " +
      "That means adding a bookmark causes outbound traffic to the target site (and to a third-party favicon service). " +
      "Pass fetch_metadata:false and/or fetch_favicon:false to suppress those fetches — with both false the server makes no outbound request for this bookmark and stores exactly what you passed. " +
      "The stored favicon and thumbnail are not echoed back in this response — they would cost kilobytes of base64 per call and are not part of search results either.",
    inputSchema: {
      type: "object",
      properties: {
        url: { type: "string", description: "Full URL to bookmark (e.g., https://example.com/article)." },
        title: { type: "string", description: "Human-readable title for the bookmark. Optional — if omitted and fetch_metadata is true (the default), the server fetches the page and fills the title from its <title> tag (see the outbound-fetch NOTE above); with fetch_metadata:false, or when that fetch fails, the bookmark is stored with an empty title." },
        related_time: { type: "string", description: `When this bookmark relates to. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        fetch_metadata: {
          type: "boolean",
          description:
            "Default: true — the server fetches the page and fills empty title / description / thumbnail from it. " +
            "Pass false to skip that page fetch entirely (no outbound request to the target site); the bookmark then stores only what you passed. " +
            "Prefer false when the URL is sensitive, unreachable from the server, or you already supply the title.",
          default: true,
        },
        fetch_favicon: {
          type: "boolean",
          description:
            "Default: true — the server fetches a favicon for the URL's domain from a third-party favicon service. " +
            "Pass false to skip the favicon fetch. Independent of fetch_metadata.",
          default: true,
        },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["url"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_nlog",
    description:
      "Create an expense/income record (nlog) in gkill for tracking financial transactions. " +
      "Each record has a title (what was purchased or received), an amount (negative for expense/spending, positive for income/refund), and an optional shop name. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_nlog (full Nlog entity with id, title, shop, amount, rep_name, related_time, etc.), added_kyou (parent Kyou wrapper). " +
      "The two overlap on purpose: added_kyou is the row shape a timeline shows (id, data_type, related_time, is_image/is_video), added_* is the entity with the fields you just wrote. Use added_*.id as the id everywhere — they are the same id. " +
      "Use the returned id as target_id for gkill_add_tag (e.g., tag with category like \"food\", \"transport\") to organize expenses.",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string", description: "Description of the expense/income (e.g., \"lunch\", \"train ticket\", \"freelance payment\")." },
        amount: { type: "integer", description: "Monetary amount (integer only, e.g. -1500 for expense, 200 for income). Must be a valid integer — empty or non-integer values are rejected by the server." },
        shop: { type: "string", description: "Shop, store, or source name (e.g., \"Starbucks\", \"Amazon\"). Optional." },
        related_time: { type: "string", description: `When the transaction occurred. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["title", "amount"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_lantana",
    description:
      "Create a mood record (lantana) in gkill for tracking emotional state over time. " +
      "Mood is an integer from 0 (lowest/worst) to 10 (highest/best), representing a subjective self-assessment of well-being. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_lantana (full Lantana entity with id, mood, rep_name, related_time, etc.), added_kyou (parent Kyou wrapper). " +
      "The two overlap on purpose: added_kyou is the row shape a timeline shows (id, data_type, related_time, is_image/is_video), added_* is the entity with the fields you just wrote. Use added_*.id as the id everywhere — they are the same id. " +
      "Use the returned id as target_id for gkill_add_tag or gkill_add_text to add context (e.g., tag with reason like \"exercise\", annotate with notes about why the mood is high/low). " +
      "Typical usage: record mood periodically (e.g., morning, evening) to build a mood timeline.",
    inputSchema: {
      type: "object",
      properties: {
        mood: { type: "integer", description: "Mood level: 0 (lowest) to 10 (highest). Must be an integer.", minimum: 0, maximum: 10 },
        related_time: { type: "string", description: `When this mood assessment relates to. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["mood"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_timeis",
    description:
      "Create a time interval record (timeis) in gkill for tracking what you were doing during a specific period. " +
      "Each timeis has a title (the activity label) and a start/end time range. " +
      "Omit end_time to create an ongoing (open-ended) interval — it can be closed later. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_timeis (full TimeIs entity with id, title, start_time, end_time, rep_name, etc.), added_kyou (parent Kyou wrapper). " +
      "The two overlap on purpose: added_kyou is the row shape a timeline shows (id, data_type, related_time, is_image/is_video), added_* is the entity with the fields you just wrote. Use added_*.id as the id everywhere — they are the same id. " +
      "TimeIs records are used by gkill's playing view to show what was happening at any given moment. " +
      "Multiple timeis can overlap (e.g., \"work\" and \"meeting\" can run simultaneously). " +
      "Use the returned id as target_id for gkill_add_tag to categorize the activity.",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string", description: "Activity title/label (e.g., \"work\", \"meeting\", \"sleep\", \"exercise\")." },
        start_time: { type: "string", description: `When the activity started. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        end_time: { type: "string", description: `When the activity ended. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit for an ongoing interval that hasn't ended yet.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["title"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_mi",
    description:
      "Create a task (mi) in gkill's task management system. Tasks are organized into boards (like Kanban columns). " +
      "Use gkill_get_mi_board_list to discover existing board names. board_name can be any string — a non-existent board name will be created and the task is saved under that name. Pass allow_create_board:false to reject a board_name that does not exist yet (typo guard). If board_name is omitted, the account's default board is used automatically. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_mi (full Mi entity with id, title, is_checked, board_name, limit_time, estimate_start_time, estimate_end_time, rep_name, etc.), added_kyou (parent Kyou wrapper). " +
      "The two overlap on purpose: added_kyou is the row shape a timeline shows (id, data_type, related_time, is_image/is_video), added_* is the entity with the fields you just wrote. Use added_*.id as the id everywhere — they are the same id. " +
      "Tasks can have optional scheduling fields: limit_time (deadline), estimate_start_time, estimate_end_time. " +
      "Use the returned id as target_id for gkill_add_tag to categorize (e.g., \"urgent\", \"bugfix\") or gkill_add_text to add detailed notes. " +
      "Typical workflow: gkill_get_mi_board_list → pick a board → gkill_add_mi → optionally tag/annotate.",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string", description: "Task title/description. Be concise but descriptive." },
        board_name: { type: "string", description: "Board name to place the task on. Use gkill_get_mi_board_list to discover existing names. Any string is accepted — a non-existent name creates a new board (unless allow_create_board is false). If omitted, the account's default board is used." },
        allow_create_board: {
          type: "boolean",
          description:
            "Default: true — an unknown board_name silently creates a new board. " +
            "Pass false to instead reject a board_name that does not match an existing board " +
            "(checked against gkill_get_mi_board_list; exact match). Use it as a typo guard when you mean to file into an existing board.",
          default: true,
        },
        is_checked: { type: "boolean", description: "Whether the task is already completed. Default: false. Set to true to create a pre-completed task (e.g., logging past work)." },
        limit_time: { type: "string", description: `Deadline for the task. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC} — a date-only value expands to the END of that day (23:59:59 local), so "2026-08-25" means "due by the end of the 25th". Optional.` },
        estimate_start_time: { type: "string", description: `Estimated start time for scheduling. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC} — a date-only value expands to the START of that day (00:00:00 local). Optional.` },
        estimate_end_time: { type: "string", description: `Estimated end time for scheduling. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC} — a date-only value expands to the END of that day (23:59:59 local). Optional.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["title"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_kc",
    description:
      "Create a numeric record (kc) in gkill for tracking any quantitative measurement over time. " +
      "Use cases: step counts, body weight, temperature, water intake, study hours, or any custom metric. " +
      "Each record has a title (what is being measured) and a num_value (the measurement). " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_kc (full KC entity with id, title, num_value, rep_name, related_time, etc.), added_kyou (parent Kyou wrapper). " +
      "The two overlap on purpose: added_kyou is the row shape a timeline shows (id, data_type, related_time, is_image/is_video), added_* is the entity with the fields you just wrote. Use added_*.id as the id everywhere — they are the same id. " +
      "Use the returned id as target_id for gkill_add_tag to categorize (e.g., tag with \"health\", \"fitness\").",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string", description: "What is being measured (e.g., \"steps\", \"weight\", \"temperature\", \"study hours\")." },
        num_value: { type: "number", description: "Numeric measurement value. Integer or decimal (e.g., 10000, 72.5, -3)." },
        related_time: { type: "string", description: `When this measurement was taken. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["title", "num_value"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_tag",
    description:
      "Add a tag to an existing entry in gkill. Tags are the primary way to categorize and organize life-log data. " +
      "The target_id must be the ID of an existing kyou entry — obtain this from the response of any gkill_add_* tool (e.g., added_kmemo.id, added_mi.id). " +
      "Tags are free-form strings. Use gkill_get_all_tag_names to discover existing tags and maintain consistency. " +
      "You can add multiple tags to the same entry by calling this tool multiple times with the same target_id but different tag values. " +
      "The repository for the tag is determined automatically by the server. " +
      "Response fields: added_tag (full Tag entity with id, tag, target_id, rep_name, related_time, etc.). Tags have no parent Kyou wrapper of their own. " +
      "Typical workflow: create an entry (e.g., gkill_add_kmemo) → use the returned id → gkill_add_tag to categorize it.",
    inputSchema: {
      type: "object",
      properties: {
        tag: { type: "string", description: "Tag name string. Free-form text (e.g., \"work\", \"personal\", \"important\", \"recipe\")." },
        target_id: { type: "string", description: "ID of the existing kyou entry to tag. Obtain from the response of gkill_add_kmemo, gkill_add_mi, or any other gkill_add_* tool." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["tag", "target_id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_add_text",
    description:
      "Add a text annotation to an existing entry in gkill. Text annotations provide supplementary notes or details attached to a parent record. " +
      "Unlike tags (short labels), text annotations are for longer-form content such as descriptions, comments, or context. " +
      "The target_id must be the ID of an existing kyou entry — obtain this from the response of any gkill_add_* tool. " +
      "You can add multiple text annotations to the same entry by calling this tool multiple times. " +
      "The repository is determined automatically by the server. " +
      "Response fields: added_text (full Text entity with id, text, target_id, rep_name, related_time, etc.). Text annotations have no parent Kyou wrapper of their own. " +
      "Typical workflow: create an entry → gkill_add_text to attach detailed notes.",
    inputSchema: {
      type: "object",
      properties: {
        text: { type: "string", description: "Text annotation content. Supports free-form text including multi-line." },
        target_id: { type: "string", description: "ID of the existing kyou entry to annotate. Obtain from the response of any gkill_add_* tool." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["text", "target_id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_submit_kftl",
    // 文法の全文（~~ / ?? 反復・/expense の対・/end 系・行ごとの失敗規則）は help topic kftl へ移した（ADR-0622）。
    // ここに残すのは「接頭辞は行単独」の規則・接頭辞の一覧・例・応答とトランザクションの要約。
    description:
      "Submit KFTL-formatted text: gkill's line-based format that creates multiple records from one text block. " +
      "CRITICAL: text is split by newlines and each prefix MUST be alone on its own line, with the value on the NEXT line ('/mood' then '8'; '/mood 8' on one line is rejected). " +
      "Lines without a prefix are kmemo text (adjacent lines merge into one kmemo); separate records with a line that is just 、 or , (、、 also advances the time by 1 second). " +
      "Prefixes (each the ENTIRE line): /mi or ーみ (task: title, board, estimated start, estimated end, deadline on the following lines), ~~ (turn the record just above into a task; opens and closes with ~~), " +
      "/mood or ーら (0-10), /expense or ーん (shop, then title/amount pairs — NOT /nlog), /url or ーう (URL then title), /num or ーか (title then value), " +
      "/start or ーた (start a timeis), /end or ーえ (end a running timeis; the title line is REQUIRED), /timeis or ーち (title, start, end), /end? /endt /endt? (variants), " +
      "# or 。 (tag for the previous record), ? or ？ (related time), ?? (repeat block), -- or ーー (text block), ! or ！ (stop). " +
      "Example (kmemo + mood + expense): \"今日はいい天気だった\\n、\\n/mood\\n8\\n、\\n/expense\\nカフェ\\nアイスコーヒー\\n-500\\n!\". " +
      "Response: messages[] and created[] ({id, data_type, updated, related_time}) in write order; use created[].id as target_id for gkill_add_tag / gkill_add_text. " +
      "The whole submission is one transaction: any per-line error (a value-less prefix, a mood outside 0-10, an unreadable date, a missing write repository) rejects it and NOTHING is written — fix the text and resend. " +
      "Pass the same idempotency_key on a retry so a replay of an already successful submission is folded instead of written twice. " +
      "Records carry create_app \"gkill_kftl\" (same as the web notepad), so create_apps:[\"gkill_mcp_readwrite\"] does not find them. " +
      "Full syntax (~~ and ?? repeat rules, /expense pairs, the /end family, per-line failure rules): gkill_get_mcp_help topic:kftl.",
    inputSchema: {
      type: "object",
      properties: {
        kftl_text: { type: "string", description: "KFTL formatted text block. Multi-line (\\n separated). CRITICAL: Each prefix (/mood, /expense, /mi, etc.) MUST be the ENTIRE line by itself — do NOT put data values on the same line as the prefix. The data goes on the NEXT line(s). Use 、 or , on its own line to separate entities." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
        idempotency_key: {
          type: "string",
          description:
            "Optional replay guard. A failed submission writes nothing, so retrying it is always safe; the guard is for "
            + "replaying a submission that already SUCCEEDED (e.g. the response was lost): retrying with the SAME key folds "
            + "the replay into the original submission instead of writing every record again. Omit it and every retry "
            + "writes afresh. Use any stable string you can reproduce for the retry (e.g. a uuid you generate once per submission).",
        },
      },
      required: ["kftl_text"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_delete_kyou",
    description:
      "Soft-delete an existing entry by setting is_deleted=true. The entry is not physically removed — it is marked as deleted and hidden from normal queries. " +
      "Requires the entry's ID (from a previous gkill_add_* response, or from gkill_get_kyous on the read / readwrite servers) and its data_type. " +
      "Valid data_type values: kmemo (text memo), urlog (bookmark), nlog (expense), lantana (mood), timeis (time interval), mi (task), kc (numeric), tag, text, rekyou (repost), mirekyou (an entry turned into a task), notification. " +
      "The appropriate update endpoint is selected automatically based on data_type. " +
      "Response fields: updated_{data_type} (the entity with is_deleted=true), and updated_kyou (parent Kyou wrapper) only for types that have one — tag and text are attached data with no Kyou of their own, so their responses carry updated_tag / updated_text alone. " +
      "Fails with 'Entity is already deleted' when the entry is already deleted, instead of stacking another pointless version — the counterpart of gkill_restore_kyou's 'already active' guard — so it is safe to retry after an uncertain response (a timeout, say). It is NOT safe to call speculatively: an active entry WILL be deleted. " +
      "Note: this is a soft-delete. The entry stays in the database — read it back with gkill_get_kyou_history, list deleted entries with query.include_deleted_data on gkill_get_kyous, and undo with gkill_restore_kyou. " +
      "Note: idf (file) and git_commit_log entries cannot be deleted via this tool — they are managed by the file system and git repositories respectively. " +
      "Addressing: exactly one form is REQUIRED — either id + data_type (single entry) or targets[] (batch). The schema marks none of them required because it cannot express this either/or; a call with neither (or both) is rejected at runtime.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the entry to soft-delete. Obtain from gkill_add_* responses, or from gkill_get_kyous on the read / readwrite servers." },
        targets: {
          type: "array",
          items: {
            type: "object",
            properties: {
              id: { type: "string" },
              data_type: { type: "string", enum: ENTITY_AND_PROJECTION_DATA_TYPE_VALUES },
            },
            required: ["id", "data_type"],
            additionalProperties: false,
          },
          description:
            "Batch form: delete several entries in one call, instead of id + data_type. " +
            "At most 100 entries; they are processed one by one in order. " +
            "gkill_submit_kftl returns created[] in exactly this shape, so its response can be passed straight back here. " +
            "This is NOT a transaction: on partial failure the response lists every entry with ok / error, " +
            "plus succeeded_count and failed_count, so you can see how far it got. " +
            "Cannot be combined with id / data_type.",
        },

        data_type: {
          type: "string",
          description:
            "Data type of the entry to delete. Must match the actual type of the entry. " +
            "Two vocabularies exist: search results and add_* / update_* responses carry PROJECTION names (mi_create / mi_check / mi_limit / mi_start / mi_end, mirekyou_*, timeis_start / timeis_end), while this parameter is the ENTITY type (mi / mirekyou / timeis). Projection names are accepted here and folded to their entity type, so a data_type copied straight out of a response works.",
          enum: ENTITY_AND_PROJECTION_DATA_TYPE_VALUES,
        },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_kmemo",
    description:
      "Update an existing text memo (kmemo) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity by ID, merges your changes, updates metadata (update_time, update_app, update_device, update_user), and sends the update to the backend. " +
      "To obtain the entity ID: use the id from a previous gkill_add_kmemo response (added_kmemo.id), or search with gkill_get_kyous to find existing entries and their IDs. " +
      "Response fields: updated_kmemo (full Kmemo entity after update, with id, rep_name, content, related_time, create_time, update_time, etc.), updated_kyou (parent Kyou wrapper). " +
      "Typical workflow: gkill_get_kyous({query:{words:[\"keyword\"]}}) → find the entry → gkill_update_kmemo({id: found_id, content: \"updated text\"}).",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the kmemo to update. Obtain from gkill_add_kmemo response (added_kmemo.id) or gkill_get_kyous." },
        content: { type: "string", description: "New memo text content." },
        related_time: { type: "string", description: `New related time (when the memo relates to). ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_urlog",
    description:
      "Update an existing bookmark/URL record (urlog) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_urlog response, or search with gkill_get_kyous. " +
      "Response fields: updated_urlog (full URLog entity after update, with id, url, title, rep_name, related_time, etc.), updated_kyou (parent Kyou wrapper). " +
      "Unlike gkill_add_urlog, this tool never causes outbound traffic: the server re-fetches the page only when explicitly asked to (a request key MCP does not send), so there are no fetch_metadata / fetch_favicon arguments here and a bookmark saved with fetching suppressed stays un-fetched. " +
      "Use cases: correct a URL typo, add/change a title for a previously untitled bookmark, change related_time.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the urlog to update. Obtain from gkill_add_urlog response or gkill_get_kyous." },
        url: { type: "string", description: "New URL." },
        title: { type: "string", description: "New human-readable title. Omit to keep unchanged." },
        related_time: { type: "string", description: `New related time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_nlog",
    description:
      "Update an existing expense/income record (nlog) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_nlog response, or search with gkill_get_kyous. " +
      "Response fields: updated_nlog (full Nlog entity after update, with id, title, shop, amount, rep_name, related_time, etc.), updated_kyou (parent Kyou wrapper). " +
      "Use cases: correct an expense amount, change the shop name, update the description.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the nlog to update. Obtain from gkill_add_nlog response or gkill_get_kyous." },
        title: { type: "string", description: "New expense/income description." },
        amount: { type: "integer", description: "New monetary amount (integer only, e.g. -1500 for expense, 200 for income). Must be a valid integer." },
        shop: { type: "string", description: "New shop/store name. Omit to keep unchanged." },
        related_time: { type: "string", description: `New related time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_lantana",
    description:
      "Update an existing mood record (lantana) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_lantana response, or search with gkill_get_kyous. " +
      "Response fields: updated_lantana (full Lantana entity after update, with id, mood, rep_name, related_time, etc.), updated_kyou (parent Kyou wrapper). " +
      "Use cases: correct a mood value that was recorded incorrectly, adjust the related_time.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the lantana to update. Obtain from gkill_add_lantana response or gkill_get_kyous." },
        mood: { type: "integer", description: "New mood level: 0 (lowest) to 10 (highest). Must be an integer.", minimum: 0, maximum: 10 },
        related_time: { type: "string", description: `New related time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_timeis",
    description:
      "Update an existing time interval record (timeis) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_timeis response, or search with gkill_get_kyous. " +
      "Response fields: updated_timeis (full TimeIs entity after update, with id, title, start_time, end_time, rep_name, etc.), updated_kyou (parent Kyou wrapper). " +
      "Common use case: close an open-ended timeis by setting end_time (e.g., gkill_update_timeis({id, end_time: \"2026-03-31T18:00:00+09:00\"})). " +
      "Also useful for: correcting start/end times, renaming an activity, and reopening a finished one (end_time: null).",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the timeis to update. Obtain from gkill_add_timeis response or gkill_get_kyous." },
        title: { type: "string", description: "New activity title/label." },
        start_time: { type: "string", description: `New start time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        end_time: { type: ["string", "null"], description: `New end time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Set this to close an open-ended (ongoing) timeis. Omit to keep unchanged. Pass null to CLEAR it and put the interval back to in-progress — omitting the field leaves the existing end time alone, so null is the only way to reopen a timeis that was already ended.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_mi",
    description:
      "Update an existing task (mi) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_mi response, or search with gkill_get_kyous (query:{for_mi:true, include_create_mi:true}). " +
      "Response fields: updated_mi (full Mi entity after update, with id, title, is_checked, board_name, limit_time, estimate_start_time, estimate_end_time, rep_name, etc.), updated_kyou (parent Kyou wrapper). " +
      "Common use cases: mark a task as completed (is_checked:true), move to a different board (board_name), update deadline (limit_time), rename a task. " +
      "Typical workflow: gkill_get_kyous({query:{for_mi:true, mi_check_state:\"uncheck\", include_create_mi:true}}) → find the task → gkill_update_mi({id, is_checked:true}).",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the mi to update. Obtain from gkill_add_mi response or gkill_get_kyous." },
        title: { type: "string", description: "New task title." },
        board_name: { type: "string", description: "New board name to move the task to. Any string accepted — non-existent names create new boards (unless allow_create_board is false). Omit to keep the current board unchanged." },
        allow_create_board: {
          type: "boolean",
          description:
            "Default: true — an unknown board_name silently creates a new board. " +
            "Pass false to instead reject a board_name that does not match an existing board " +
            "(checked against gkill_get_mi_board_list; exact match). Only meaningful together with board_name.",
          default: true,
        },
        is_checked: { type: "boolean", description: "Set to true to mark as completed, false to reopen. Omit to keep unchanged." },
        limit_time: { type: "string", description: `New deadline. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        estimate_start_time: { type: "string", description: `New estimated start time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        estimate_end_time: { type: "string", description: `New estimated end time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_kc",
    description:
      "Update an existing numeric record (kc) in gkill using patch semantics — only specify the fields you want to change; unspecified fields are preserved as-is. " +
      "The MCP server internally fetches the current entity, merges changes, and sends the update. " +
      "To obtain the entity ID: use the id from a previous gkill_add_kc response, or search with gkill_get_kyous. " +
      "Response fields: updated_kc (full KC entity after update, with id, title, num_value, rep_name, related_time, etc.), updated_kyou (parent Kyou wrapper). " +
      "Use cases: correct a measurement value, rename the metric title, adjust related_time.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the kc to update. Obtain from gkill_add_kc response or gkill_get_kyous." },
        title: { type: "string", description: "New measurement title (e.g., \"steps\", \"weight\")." },
        num_value: { type: "number", description: "New numeric value. Integer or decimal (e.g., 10000, 72.5)." },
        related_time: { type: "string", description: `New related time. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Omit to keep unchanged.` },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_tag",
    description:
      "Update an existing tag in gkill using patch semantics. Changes the tag name while keeping the tag attached to the same target entry. " +
      "The MCP server internally fetches the current tag entity via the tag history API (get_tag_histories_by_tag_id), merges the change, and sends the update. " +
      "To obtain the tag ID: use the id from a previous gkill_add_tag response (added_tag.id). Note: tags are separate entities from the entries they're attached to — each tag has its own ID distinct from the parent entry's ID. " +
      "Response fields: updated_tag (full Tag entity after update, with id, tag, target_id, rep_name, etc.). Tags are attached data with no parent Kyou wrapper of their own, so updated_kyou is always null here. " +
      "Use case: rename a tag (e.g., fix a typo in a tag name, change \"wrk\" to \"work\"). To remove a tag entirely, use gkill_delete_kyou with data_type=\"tag\".",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the tag entity to update. This is the tag's own ID (added_tag.id), not the target entry's ID." },
        tag: { type: "string", description: "New tag name string." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_update_text",
    description:
      "Update an existing text annotation in gkill using patch semantics. Changes the text content while keeping the annotation attached to the same target entry. " +
      "The MCP server internally fetches the current text entity via the text history API (get_text_histories_by_text_id), merges the change, and sends the update. " +
      "To obtain the text ID: use the id from a previous gkill_add_text response (added_text.id). Note: text annotations are separate entities from the entries they're attached to — each has its own ID distinct from the parent entry's ID. " +
      "Response fields: updated_text (full Text entity after update, with id, text, target_id, rep_name, etc.). Text annotations are attached data with no parent Kyou wrapper of their own, so updated_kyou is always null here. " +
      "Use case: edit a note or comment attached to an existing entry. To remove a text annotation entirely, use gkill_delete_kyou with data_type=\"text\".",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the text annotation entity to update. This is the text's own ID (added_text.id), not the target entry's ID." },
        text: { type: "string", description: "New text annotation content. Supports multi-line." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      required: ["id"],
      additionalProperties: false,
    },
  },
  {
    name: "gkill_restore_kyou",
    description:
      "Undo a soft-delete: clear is_deleted on an entry so it shows up in searches again. " +
      "This is the counterpart of gkill_delete_kyou. Use gkill_get_kyou_history first to confirm what you are " +
      "about to bring back — a deleted entry is invisible to every ordinary search, so restoring blind is a guess. " +
      "gkill is append-only, so a restore adds a new version rather than removing the deleting one; " +
      "delete/restore cycles keep growing the history. " +
      "Note the restored entry lands in the account's current write repository, which is not necessarily the " +
      "repository it came from. " +
      "Fails with 'already active' when the entry is not deleted, so it is safe to retry after an uncertain response (a timeout, say). It is NOT safe to call speculatively: a deleted entry WILL be restored. " +
      "Response fields: restored_{data_type} (the entity with is_deleted=false), updated_kyou (parent Kyou wrapper, when the server returns one). " +
      "Addressing: exactly one form is REQUIRED — either id + data_type (single entry) or targets[] (batch). The schema marks none of them required because it cannot express this either/or; a call with neither (or both) is rejected at runtime.",
    inputSchema: {
      type: "object",
      properties: {
        id: { type: "string", description: "ID of the soft-deleted entry to restore." },
        targets: {
          type: "array",
          items: {
            type: "object",
            properties: {
              id: { type: "string" },
              data_type: { type: "string", enum: ENTITY_AND_PROJECTION_DATA_TYPE_VALUES },
            },
            required: ["id", "data_type"],
            additionalProperties: false,
          },
          description:
            "Batch form: restore several entries in one call, instead of id + data_type. " +
            "At most 100 entries; they are processed one by one in order. " +
            "gkill_submit_kftl returns created[] in exactly this shape, so its response can be passed straight back here. " +
            "This is NOT a transaction: on partial failure the response lists every entry with ok / error, " +
            "plus succeeded_count and failed_count, so you can see how far it got. " +
            "Cannot be combined with id / data_type.",
        },

        data_type: {
          type: "string",
          description:
            "Data type of the entry. Must match the actual type of the entry. " +
            "Two vocabularies exist: search results and add_* / update_* responses carry PROJECTION names (mi_create / mi_check / mi_limit / mi_start / mi_end, mirekyou_*, timeis_start / timeis_end), while this parameter is the ENTITY type (mi / mirekyou / timeis). Projection names are accepted here and folded to their entity type, so a data_type copied straight out of a response works.",
          enum: ENTITY_AND_PROJECTION_DATA_TYPE_VALUES,
        },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
      },
      additionalProperties: false,
    },
  },
];
