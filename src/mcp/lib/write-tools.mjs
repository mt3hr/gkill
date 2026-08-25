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
      "The stored favicon and thumbnail are not echoed back in this response — they would cost kilobytes of base64 per call and are not part of search results either.",
    inputSchema: {
      type: "object",
      properties: {
        url: { type: "string", description: "Full URL to bookmark (e.g., https://example.com/article)." },
        title: { type: "string", description: "Human-readable title for the bookmark. Optional — if omitted, only the URL is stored." },
        related_time: { type: "string", description: `When this bookmark relates to. ${ISO_DATETIME_DESC} or ${DATE_ONLY_DESC}. Defaults to now.` },
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
      "TimeIs records are used by gkill's plaing view to show what was happening at any given moment. " +
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
      "Use gkill_get_mi_board_list to discover existing board names. board_name can be any string — a non-existent board name will be created and the task is saved under that name. If board_name is omitted, the account's default board is used automatically. " +
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
        board_name: { type: "string", description: "Board name to place the task on. Use gkill_get_mi_board_list to discover existing names. Any string is accepted — a non-existent name creates a new board. If omitted, the account's default board is used." },
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
    description:
      "Submit KFTL-formatted text for batch processing. KFTL is gkill's line-based text format that creates multiple records from a single text block. " +
      "CRITICAL parsing rules: " +
      "(1) Text is split by newlines (\\n). Each line is processed independently. " +
      "(2) Prefixes MUST be on their own line with NOTHING else on that line. The prefix line and the data value MUST be on SEPARATE lines. " +
      "For example, '/mood' must be alone on one line, and '8' on the next line. Writing '/mood 8' on one line is rejected per-line, and so is a prefix with no value line after it. " +
      "(3) Lines without a recognized prefix are treated as kmemo (text memo) content. Adjacent non-prefixed lines are merged into a single kmemo. " +
      "(4) To create SEPARATE records, insert a separator line (、 or ,) between them. Without separators, consecutive lines merge into one kmemo. " +
      "Supported prefix lines (must be the ENTIRE line, not part of a line): " +
      "/mi or ーみ → task in up to five positional lines: title, then board name, estimated start, estimated end, deadline (same order as the ~~ block), " +
      "~~ or ～～ → turn the record written just ABOVE into a task (repost task). Opens AND closes with the same ~~ marker, and is meaningless on its own — the record it tasks must come first. There is NO title line (the original record is shown as-is). Inside the block: board name, estimated start, estimated end, deadline (all optional, no ? needed before a date). Lines starting with # inside the block become tags on the TASK itself and may appear before or after the board name. Use /mi for a brand new task. ~~ can ONLY task a record created earlier in THIS SAME submission (its target is the id the previous line just minted) — it cannot reference a record that already exists in gkill, and there is no syntax that can. It also needs a mirekyou repository configured for the account; without one the submission fails after the records above it were already written, " +
      "/mood or ーら → next line is mood value (0-10), " +
      "/expense or ーん → next lines: shop name, then (title/description, amount) pairs repeating — one expense record per pair. " +
      "A #tag line or a -- text block written after an amount line attaches to that one payment only; write them after the amount, never before /expense. " +
      "(IMPORTANT: the prefix is /expense, NOT /nlog), " +
      "/url or ーう → next line is the URL, and the line AFTER that is its title, " +
      "/num or ーか → next line is title, then the line after that is the numeric value, " +
      "/start or ーた → next line is timeis start label, " +
      "/end or ーえ → end a running timeis. The NEXT LINE IS REQUIRED and must be the exact title of the running timeis to close; omitting it fails with \"打刻終了タイトルを指定してください\", " +
      "/timeis or ーち → timeis in three fixed lines: title, then start datetime, then end datetime, " +
      "/end? or ーいえ → same as /end but tolerates \"no such running timeis\". The title line is STILL REQUIRED — the if-exists part only forgives a missing target, not a missing title, " +
      "/endt or ーたえ → end a running timeis by tag. The NEXT LINE IS REQUIRED and holds the tag name(s), separated by 、 or , and written WITHOUT the # / 。 prefix, " +
      "/endt? or ーいたえ → same as /endt but tolerates \"no such running timeis\"; the tag line is still required, " +
      "# or 。 → tag (attach to previous record). Matched by PREFIX, so a Markdown heading line like \"# Title\" becomes a tag, " +
      "? or ？ → related time. Also matched by prefix, so any line starting with ? is parsed as a datetime and fails if it is not one, " +
      "-- or ーー → text block start/end, " +
      "! or ！ → stop processing. On the FIRST line it does nothing (processing continues), " +
      "(no prefix) → kmemo text content. " +
      "Separator lines: 、 or , → separate into a new entity; 、、 or ,, → separate + increment time by 1 second. " +
      "Example (creates 3 records: kmemo + mood + expense): " +
      "\"今日はいい天気だった\\n、\\n/mood\\n8\\n、\\n/expense\\nカフェ\\nアイスコーヒー\\n-500\\n!\" " +
      "Response fields: messages[] (server processing messages) and created[] ({id, data_type, updated}) — one entry per record actually written, in the order they were written. Lines that write nothing (a blank kmemo, a blank task, a blank expense) produce no entry, and ending a timeis reports the existing record with updated:true rather than a new id. Use created[].id as target_id for gkill_add_tag / gkill_add_text. " +
      "On failure the errors are reported one per bad line. Which failures can leave records behind differs: a bad VALUE (a mood outside 0-10, a prefix with no argument) is caught while parsing, before a single byte is written, so created[] comes back empty and nothing was saved. A failure while actually writing (a missing write repository, a database error) happens after earlier lines are already in, and created[] then lists exactly those — KFTL is not a database transaction, so they stay, and the error names the line it stopped at. Pass the same idempotency_key when you retry so the replay does not write the earlier records a second time. "
      + "Provenance: records written through this tool carry create_app=\"gkill_kftl\" (the same value the web notepad writes) and create_device set to the SERVER's device name, not \"mcp\". There is currently no field that separates MCP-submitted KFTL from hand-typed notepad KFTL, so create_apps:[\"gkill_mcp_readwrite\"] does NOT find them.",
    inputSchema: {
      type: "object",
      properties: {
        kftl_text: { type: "string", description: "KFTL formatted text block. Multi-line (\\n separated). CRITICAL: Each prefix (/mood, /expense, /mi, etc.) MUST be the ENTIRE line by itself — do NOT put data values on the same line as the prefix. The data goes on the NEXT line(s). Use 、 or , on its own line to separate entities." },
        locale_name: { type: "string", description: "Locale for server messages, e.g. ja/en. Defaults to server default (ja)." },
        idempotency_key: {
          type: "string",
          description:
            "Optional replay guard. KFTL is not a database transaction: when a submission fails partway, the records "
            + "already written stay. Retrying with the SAME key folds the replay into the original submission instead of "
            + "writing those records again. Omit it and every retry writes afresh. Use any stable string you can reproduce "
            + "for the retry (e.g. a uuid you generate once per submission).",
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
      "Fails with 'Entity is already deleted' when the entry is already deleted, instead of stacking another pointless version — the counterpart of gkill_restore_kyou's 'already active' guard — so it is safe to call speculatively. " +
      "Note: this is a soft-delete. The entry stays in the database — read it back with gkill_get_kyou_history, list deleted entries with query.include_deleted_data on gkill_get_kyous, and undo with gkill_restore_kyou. " +
      "Note: idf (file) and git_commit_log entries cannot be deleted via this tool — they are managed by the file system and git repositories respectively.",
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
        board_name: { type: "string", description: "New board name to move the task to. Any string accepted — non-existent names create new boards. Omit to keep the current board unchanged." },
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
      "Fails with 'already active' when the entry is not deleted, so it is safe to call speculatively. " +
      "Response fields: restored_{data_type} (the entity with is_deleted=false), updated_kyou (parent Kyou wrapper, when the server returns one).",
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
