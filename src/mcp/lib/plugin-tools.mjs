// プラグイン関連のMCPツール定義とハンドラ、および
// gkill_get_kyous のレスポンスにプラグインKyouの本文を埋め込む処理。
//
// read / write / readwrite の3サーバから共有する。サーバごとにAPI呼び出しの
// メソッド名が違う (callRead / callWrite / callApi) ため、呼び出し口は
// call(pathname, body) 形式のコールバックで受け取る。
//
// 同一プラグインへ並列に投げない理由（stdio が1本しかない）:
// documents/adr/0602-mcp-inline-plugin-content.md

import { GkillApiError } from "./errors.mjs";
import { normalizeLocaleOnlyArgs } from "./normalization.mjs";
import { htmlToText } from "./html-text.mjs";
import {
  DEFAULT_PLUGIN_CONTENT_FORMAT,
  DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  MAX_INLINE_PLUGIN_CONTENT_KYOUS,
  INLINE_PLUGIN_CONTENT_TOTAL_TEXT_LENGTH,
  INLINE_PLUGIN_CONTENT_REP_CONCURRENCY,
  INLINE_PLUGIN_CONTENT_DEADLINE_MS,
  MAX_INLINE_PLUGIN_CONTENT_HTML_LENGTH,
} from "./constants.mjs";

export const GET_PLUGIN_LIST_ENDPOINT = "/api/get_plugin_list";
export const GET_PLUGIN_CONTENT_HTML_ENDPOINT = "/api/get_plugin_content_html";

export const PLUGIN_TOOL_NAMES = ["gkill_get_plugin_list"];

// content_error に載せるメッセージの上限。スタックトレース等でレスポンスが
// 膨らむのを防ぐ。
const MAX_PLUGIN_CONTENT_ERROR_LENGTH = 200;

export const PLUGIN_TOOLS = [
  {
    name: "gkill_get_plugin_list",
    description:
      "List the gkill plugins installed for the current user. Plugins are external programs that feed their own " +
      "data into gkill — for example Claude Code / Claude.ai / ChatGPT conversation logs, Fitbit daily metrics, " +
      "Google location history. " +
      "IMPORTANT — plugins do not all play the same role, and emits_kyou tells you which one you are looking at. " +
      "When emits_kyou is true the plugin supplies kyou: filter gkill_get_kyous with query.reps using an entry of " +
      "rep_names[] (ALWAYS present: the rep names its kyou actually carry — a plugin that wraps several repositories, " +
      "such as Git repositories archived as zip, lists each repository there, and rep_names is [] until its index is " +
      "built) — rep_name itself is the plugin's manifest label and is NOT a query.reps value unless it also appears " +
      "in rep_names — or " +
      "the top-level data_types (its data_type), and pass include_plugin_content:true to get their bodies in the " +
      "same call. query.rep_types does NOT work for plugins — they are not in the canonical rep-type vocabulary " +
      "(a plugin whose provides names a typed kind such as kc or git_commit_log is the exception: its records also " +
      "answer to that rep_types value). " +
      "When emits_kyou is false the plugin supplies no kyou at all, and its data_type / rep_name are NOT query " +
      "values: passing them matches nothing. Read that plugin's data through the route matching provides — today " +
      "provides:[\"gpslog\"] means gkill_get_gps_log. " +
      "provides lists what the plugin supplies beyond kyou metadata (kmemo, kc, urlog, nlog, lantana, timeis, mi, " +
      "git_commit_log, tag, text, notification, gpslog); an absent provides means it supplies plain kyou only. " +
      "Response fields: plugins[] with name, version, description, data_type, rep_name (manifest label), rep_names " +
      "(the query.reps values; always present for kyou-emitting plugins), emits_kyou, provides, " +
      "is_alive (responds to a ping), " +
      "process_running (started; read without side effects), has_last_error, typed_index, and gps_index. " +
      "has_last_error is true when the plugin process wrote something to stderr — that is the signal to look at " +
      "when is_alive is true but no records come back. The text itself is deliberately NOT returned: it is the " +
      "plugin's raw stderr and carries the directory layout of the user's own machine. When it is true and the " +
      "text is needed, ask the person running gkill to read it from the server console, and never copy it into " +
      "documents or commit messages. " +
      "typed_index is the KYOU index and is present only for plugins that declare a non-gpslog provides; it carries " +
      "state (\"ok\" / \"failed\" / \"never_built\"), record_count (unique kyou ids), oldest / newest (how far the " +
      "plugin has actually ingested), truncated, built_at, and — when a build failed — has_last_build_error and " +
      "last_attempt_at (the failure text is withheld for the same reason as last_error). Index build failures " +
      "never reach has_last_error: they happen inside gkill (timeouts, a busy plugin, malformed JSON), so " +
      "has_last_build_error is the one to read for those. last_attempt_at matters because " +
      "rebuilds back off after a failure and then produce no error at all. " +
      "gps_index is separate (GPS points are not kyou) and appears for gpslog plugins once their points have been " +
      "loaded: point_count, oldest / newest, fetched_at. It covers ONLY the points this plugin supplied, counted " +
      "before deduplication — gkill_get_gps_log spans every GPS repository (including native ones) and dedupes, " +
      "so the two numbers are expected to differ, sometimes by an order of magnitude. Do not read a stale newest " +
      "here as proof that GPS recording stopped. Its absence means nothing has requested GPS logs yet, not " +
      "that the plugin is broken — call gkill_get_gps_log with count_only:true to size the real total.",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
      },
      additionalProperties: false,
    },
  },
];

// プラグインの診断文をAIへ返さないときに足す1行。
//
// last_error はプラグインプロセスの生stderr、typed_index.last_build_error は起動失敗の
// エラー文で、どちらも「利用者の端末のどこに何が置いてあるか」を含む。gkill 側で
// ユーザー名は伏せているが、AIの文脈へ入れば資料やコミットメッセージへ引き写される経路が
// できてしまう。「何か書かれている」ことだけ has_* で伝えれば、
// 「is_alive=true なのに0件」の診断（外部監査 D2）は成立する。
// 経緯: documents/adr/0707-redact-environment-specific-strings.md
export const PLUGIN_DIAGNOSTICS_WITHHELD_WARNING =
  "plugin diagnostics are withheld from this response: last_error (raw plugin stderr) and " +
  "typed_index.last_build_error describe the directory layout of the user's own machine, so only " +
  "has_last_error / has_last_build_error are returned. When one is true and the text is needed, ask the " +
  "person running gkill to read it from the server console — do not copy it into documents or commit messages. " +
  "The most common cause of has_last_error with is_alive:true is that the plugin's configured import path " +
  "matches nothing — a real account ran for 20 months at zero records that way — so ask for that path first.";

// withoutPluginDiagnostics はプラグイン1件から診断文を落とし、
// 非空だったときだけ has_last_error / has_last_build_error を立てる。
function withoutPluginDiagnostics(plugin) {
  if (plugin === null || typeof plugin !== "object" || Array.isArray(plugin)) {
    return plugin;
  }
  const { last_error: lastError, typed_index: typedIndex, ...rest } = plugin;
  const stripped = { ...rest };
  if (typeof lastError === "string" && lastError !== "") {
    stripped.has_last_error = true;
  }
  if (typedIndex !== undefined) {
    if (typedIndex !== null && typeof typedIndex === "object" && !Array.isArray(typedIndex)) {
      const { last_build_error: lastBuildError, ...typedRest } = typedIndex;
      stripped.typed_index = { ...typedRest };
      if (typeof lastBuildError === "string" && lastBuildError !== "") {
        stripped.typed_index.has_last_build_error = true;
      }
    } else {
      stripped.typed_index = typedIndex;
    }
  }
  return stripped;
}

// hasWithheldDiagnostics は診断文を実際に落としたかどうかを返す。
// 落としていないのに警告を出すと常時ノイズになる（ADR-0609 と同じ理由）。
function hasWithheldDiagnostics(plugin) {
  if (plugin === null || typeof plugin !== "object") {
    return false;
  }
  if (plugin.has_last_error === true) {
    return true;
  }
  return (
    plugin.typed_index !== null &&
    typeof plugin.typed_index === "object" &&
    plugin.typed_index.has_last_build_error === true
  );
}

// pluginsWithoutIngestCount は「記録を出すのに取り込み件数を名乗れない」プラグインの名前を返す。
//
// provides を宣言していないプラグインには typed_index が付かない。すると
// gkill_get_plugin_list からは「取り込み0件」と「正常」の区別が付かない ——
// is_alive:true / process_running:true のまま1件も取り込めていない状態が、
// 一覧の上では完全に正常に見える（実測 2026-08-25: 会話ログ系4本のうち1本が
// has_last_error:true で全期間0件だったのに、別途カウントを打つまで分からなかった）。
//
// 件数そのものはここでは出さない。数えるにはプラグイン本体へ問い合わせることになり、
// gkill_get_plugin_list が全プラグインへ直列に往復する形になる（プラグインのハンドラは
// 数十msで返す前提。ADR-0301）。代わりに「数えられない」ことと数え方を名指しする。
function pluginsWithoutIngestCount(plugins) {
  return plugins
    .filter(
      (plugin) =>
        plugin !== null &&
        typeof plugin === "object" &&
        plugin.emits_kyou === true &&
        plugin.typed_index === undefined &&
        typeof plugin.data_type === "string" &&
        plugin.data_type !== "",
    )
    .map((plugin) => plugin.data_type);
}

// summarizePluginToolPayload はプラグインツールの結果の1行サマリを返す。
// 対象外のツール名には null を返すので、呼び出し側は既存のsummarizeにフォールバックできる。
export function summarizePluginToolPayload(name, payload) {
  switch (name) {
    case "gkill_get_plugin_list": {
      const summary = `Fetched ${Array.isArray(payload.plugins) ? payload.plugins.length : 0} plugins.`;
      // 本文の warnings を読まない経路でも気づけるようにする。
      if (Array.isArray(payload.warnings) && payload.warnings.length !== 0) {
        return `${summary} (some plugins reported diagnostics; the text is withheld — see warnings)`;
      }
      return summary;
    }
    default:
      return null;
  }
}

export function isPluginToolName(name) {
  return PLUGIN_TOOL_NAMES.includes(name);
}

/**
 * handlePluginToolCall はプラグイン関連ツールを処理する。
 *
 * @param {(pathname: string, body: object) => Promise<object>} call サーバ固有のAPI呼び出し。
 * @param {string} name ツール名。
 * @param {unknown} args ツール引数。
 * @returns {Promise<object>} ツールのペイロード。
 */
export async function handlePluginToolCall(call, name, args) {
  switch (name) {
    case "gkill_get_plugin_list": {
      const normalized = normalizeLocaleOnlyArgs(args);
      const response = await call(GET_PLUGIN_LIST_ENDPOINT, normalized);
      const plugins = (Array.isArray(response.plugins) ? response.plugins : []).map(withoutPluginDiagnostics);
      const warnings = [];
      // 実際に落としたときだけ警告を足す。落としていないのに出すと常時ノイズになる。
      if (plugins.some(hasWithheldDiagnostics)) {
        warnings.push(PLUGIN_DIAGNOSTICS_WITHHELD_WARNING);
      }
      const uncounted = pluginsWithoutIngestCount(plugins);
      if (uncounted.length !== 0) {
        warnings.push(
          `these plugins feed records but report no ingest count (they declare no provides, so they have no typed_index): ` +
            `${uncounted.join(", ")}. is_alive:true does not mean anything was ingested — a plugin can answer pings ` +
            `while holding zero records. To check one, call gkill_get_kyous with count_only:true and ` +
            `data_types:["<the data_type>"] over the period you expect.`,
        );
      }
      if (warnings.length !== 0) {
        return { plugins, warnings };
      }
      return { plugins };
    }
    default:
      throw new GkillApiError(`Unknown tool: ${name}`);
  }
}

// isPluginPayload は get_kyous のペイロードがプラグイン由来かを判定する。
function isPluginPayload(value) {
  return value !== null && typeof value === "object" && value.kind === "plugin";
}

// hasPluginContentKey は本文取得に要る rep_name と id を Kyou が持つかを判定する。
// 2026-09-19 までペイロード側にも rep_name / kyou_id が写されていたが、Kyou 側と常に同値で
// 毎件3欄が二重に並ぶだけだったので落とした（ADR-0629）。鍵は Kyou 側から取る。
function hasPluginContentKey(kyou) {
  return (
    typeof kyou.rep_name === "string" && kyou.rep_name !== "" && typeof kyou.id === "string" && kyou.id !== ""
  );
}

/**
 * collectPluginPayloads は kyous[] から kind:"plugin" のエントリを取得順に集める。
 *
 * @param {Array<object>} kyous get_kyous のレスポンスの kyous 配列。
 * @returns {Array<{rep_name: string, kyou_id: string, payload: object}>} 本文取得の鍵（Kyou 側の
 *   rep_name / id）と、本文を書き込む先のペイロード (元オブジェクトの参照)。
 */
export function collectPluginPayloads(kyous) {
  if (!Array.isArray(kyous)) {
    return [];
  }
  const payloads = [];
  for (const kyou of kyous) {
    if (kyou === null || typeof kyou !== "object") {
      continue;
    }
    if (isPluginPayload(kyou.payload) && hasPluginContentKey(kyou)) {
      payloads.push({ rep_name: kyou.rep_name, kyou_id: kyou.id, payload: kyou.payload });
    }
  }
  return payloads;
}

/**
 * runGroupedWithConcurrency はキーごとに直列、キー間は並列でタスクを実行する。
 *
 * gkillのプラグインは1プロセスにつき1ミューテックスで直列化される。しかもGo側の
 * 30秒デッドラインはミューテックス待ちを含むので (plugin_repository_impl.go)、
 * 同一プラグインへ同時に投げると待ち時間が期限を食い潰し、期限切れ時の
 * Process.Kill() でプラグインプロセスが落ちる。だからキー内は必ず直列にする。
 *
 * worker が false を返すか例外を投げた場合、そのキーの残りは実行しない。
 * 例外は握り潰すので、この関数自体は reject しない。
 *
 * @param {Array<{key: string, item: unknown}>} entries 実行対象。
 * @param {number} concurrency 同時に走らせるキーの数。
 * @param {(item: unknown) => Promise<boolean>} worker 続行するなら true を返す。
 * @returns {Promise<void>}
 */
export async function runGroupedWithConcurrency(entries, concurrency, worker) {
  const groups = new Map();
  for (const { key, item } of entries) {
    const list = groups.get(key);
    if (list) {
      list.push(item);
    } else {
      groups.set(key, [item]);
    }
  }
  const queue = Array.from(groups.values());
  if (queue.length === 0) {
    return;
  }
  // JSはシングルスレッドなので、このカウンタの読み書きに排他は要らない。
  let next = 0;
  const width = Math.max(1, Math.min(concurrency, queue.length));
  await Promise.all(
    Array.from({ length: width }, async () => {
      for (let index = next++; index < queue.length; index = next++) {
        for (const item of queue[index]) {
          let keepGoing = false;
          try {
            keepGoing = await worker(item);
          } catch {
            keepGoing = false;
          }
          if (!keepGoing) {
            break;
          }
        }
      }
    }),
  );
}

function shortPluginContentError(error) {
  const message = error instanceof Error ? error.message : String(error);
  return message.length > MAX_PLUGIN_CONTENT_ERROR_LENGTH
    ? message.slice(0, MAX_PLUGIN_CONTENT_ERROR_LENGTH)
    : message;
}

/**
 * inlinePluginContents は kyous[] のプラグインペイロードに本文を埋め込む。
 *
 * ペイロードを破壊的に更新し、個別の失敗は content_status に落として
 * gkill_get_kyous 全体は落とさない (この関数は reject しない)。
 *
 * 実行中のリクエストは絶対に abort しない。gkill 側は HTTP リクエストの
 * コンテキストをそのままプラグイン呼び出しに渡しており、abort すると
 * プラグインプロセスが kill されるため。デッドラインは「新しいリクエストを
 * 始めない」ことだけで実現する。
 *
 * @param {(pathname: string, body: object) => Promise<object>} call サーバ固有のAPI呼び出し。
 * @param {Array<object>} kyous get_kyous のレスポンスの kyous 配列 (破壊的に更新される)。
 * @param {object} [options] maxTextLength / format / maxKyous / totalTextLength /
 *   concurrency / deadlineMs / maxHtmlLength / localeName / now。
 * @returns {Promise<{requested: number, inlined: number, truncated: number,
 *   skipped: number, errors: number, total_text_length: number}>} 集計。
 */
export async function inlinePluginContents(call, kyous, options = {}) {
  const {
    maxTextLength = DEFAULT_INLINE_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
    format = DEFAULT_PLUGIN_CONTENT_FORMAT,
    maxKyous = MAX_INLINE_PLUGIN_CONTENT_KYOUS,
    totalTextLength = INLINE_PLUGIN_CONTENT_TOTAL_TEXT_LENGTH,
    concurrency = INLINE_PLUGIN_CONTENT_REP_CONCURRENCY,
    deadlineMs = INLINE_PLUGIN_CONTENT_DEADLINE_MS,
    maxHtmlLength = MAX_INLINE_PLUGIN_CONTENT_HTML_LENGTH,
    localeName,
    now = Date.now,
  } = options;

  const payloads = collectPluginPayloads(kyous);
  const stats = {
    requested: payloads.length,
    inlined: 0,
    truncated: 0,
    skipped: 0,
    errors: 0,
    total_text_length: 0,
  };
  if (payloads.length === 0) {
    return stats;
  }

  const markSkipped = (targets, reason) => {
    for (const payload of targets) {
      payload.content_status = "skipped";
      payload.content_skipped_reason = reason;
    }
    stats.skipped += targets.length;
  };

  // 同じ (rep_name, kyou_id) が複数件返ることがあるので、取得は1回にまとめる。
  // 件数上限は「取得しにいく対象の数」に対して掛ける。
  const entries = [];
  const entryByKey = new Map();
  for (const { rep_name, kyou_id, payload } of payloads) {
    const key = `${rep_name} ${kyou_id}`;
    const hit = entryByKey.get(key);
    if (hit) {
      hit.payloads.push(payload);
      continue;
    }
    if (entryByKey.size >= maxKyous) {
      markSkipped([payload], "max_kyous");
      continue;
    }
    const entry = { rep_name, kyou_id, payloads: [payload] };
    entryByKey.set(key, entry);
    entries.push(entry);
  }

  const results = new Map();
  // あるrepで打ち切ったとき、そのrepの未処理エントリに付ける理由。
  const repStopReason = new Map();
  const startedAt = now();

  try {
    await runGroupedWithConcurrency(
      entries.map((entry) => ({ key: entry.rep_name, item: entry })),
      concurrency,
      async (entry) => {
        if (now() - startedAt >= deadlineMs) {
          repStopReason.set(entry.rep_name, "deadline");
          return false;
        }
        try {
          const response = await call(GET_PLUGIN_CONTENT_HTML_ENDPOINT, {
            rep_name: entry.rep_name,
            kyou_id: entry.kyou_id,
            ...(localeName ? { locale_name: localeName } : {}),
          });
          const rawHTML = typeof response.html === "string" ? response.html : "";
          const html = rawHTML.length > maxHtmlLength ? rawHTML.slice(0, maxHtmlLength) : rawHTML;
          results.set(entry, { html, html_clipped: html.length !== rawHTML.length });
          return true;
        } catch (error) {
          results.set(entry, { error: shortPluginContentError(error) });
          // タイムアウトはプラグインプロセスを殺しているので、同じrepに投げ続けても
          // コールドスタートで待たされるだけ。そのrepの残りは諦める。
          repStopReason.set(entry.rep_name, "rep_error");
          return false;
        }
      },
    );
  } catch {
    // runGroupedWithConcurrency は投げない設計だが、ここで落ちて
    // get_kyous 全体が失敗することだけは避ける。
  }

  const wantText = format === "text" || format === "both";
  const wantHTML = format === "html" || format === "both";

  // 予算は取得完了順ではなくKyouの並び順で適用する。
  // ネットワークのタイミングによらず同じ入力から同じ出力になる。
  let used = 0;
  for (const entry of entries) {
    const result = results.get(entry);
    if (result === undefined) {
      markSkipped(entry.payloads, repStopReason.get(entry.rep_name) ?? "deadline");
      continue;
    }
    if (result.error !== undefined) {
      for (const payload of entry.payloads) {
        payload.content_status = "error";
        payload.content_error = result.error;
      }
      stats.errors += entry.payloads.length;
      continue;
    }
    const converted = htmlToText(result.html, { maxLength: maxTextLength });
    const cost = (wantText ? converted.text.length : 0) + (wantHTML ? result.html.length : 0);
    // 1件目は必ず載せる。そうしないと「1件だけ全文が欲しい」ケースで
    // 上限に関係なく常に空振りしてしまう。
    if (used > 0 && used + cost > totalTextLength) {
      markSkipped(entry.payloads, "budget");
      continue;
    }
    used += cost;
    const truncated = converted.truncated || result.html_clipped;
    for (const payload of entry.payloads) {
      if (wantText) {
        payload.content_text = converted.text;
      }
      if (wantHTML) {
        payload.content_html = result.html;
      }
      payload.content_status = truncated ? "truncated" : "ok";
    }
    stats.inlined += entry.payloads.length;
    if (truncated) {
      stats.truncated += entry.payloads.length;
    }
  }
  stats.total_text_length = used;
  return stats;
}

/**
 * summarizeInlinePluginContent は get_kyous のサマリ行に足す一文を返す。
 * インライン化していないときは空文字を返す。
 *
 * @param {object|undefined} stats inlinePluginContents の戻り値。
 * @returns {string}
 */
export function summarizeInlinePluginContent(stats) {
  if (!stats || stats.requested === 0) {
    return "";
  }
  const notes = [];
  if (stats.truncated > 0) {
    notes.push(`${stats.truncated} truncated`);
  }
  if (stats.skipped > 0) {
    notes.push(`${stats.skipped} not fetched`);
  }
  if (stats.errors > 0) {
    notes.push(`${stats.errors} failed`);
  }
  const suffix = notes.length > 0 ? ` (${notes.join(", ")})` : "";
  return ` Embedded plugin content for ${stats.inlined} of ${stats.requested} plugin kyous${suffix}.`;
}
