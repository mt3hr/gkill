// プラグイン関連のMCPツール定義とハンドラ、および
// gkill_get_kyous のレスポンスにプラグインKyouの本文を埋め込む処理。
//
// read / write / readwrite の3サーバから共有する。サーバごとにAPI呼び出しの
// メソッド名が違う (callRead / callWrite / callApi) ため、呼び出し口は
// call(pathname, body) 形式のコールバックで受け取る。

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
      "records (kyou) into gkill — for example Claude Code / Claude.ai / ChatGPT conversation logs. " +
      "Each entry has name, version, description, data_type (the data_type its kyous carry), rep_name " +
      "(the repository name its kyous carry) and is_alive (whether the plugin process currently responds). " +
      "Use this to discover which data_type / rep_name values belong to plugins, then filter gkill_get_kyous " +
      "with query.reps or query.rep_types to fetch only that plugin's records, passing " +
      "include_plugin_content:true to get their bodies in the same call. " +
      "Response fields: plugins[].",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
      },
      additionalProperties: false,
    },
  },
];

// summarizePluginToolPayload はプラグインツールの結果の1行サマリを返す。
// 対象外のツール名には null を返すので、呼び出し側は既存のsummarizeにフォールバックできる。
export function summarizePluginToolPayload(name, payload) {
  switch (name) {
    case "gkill_get_plugin_list":
      return `Fetched ${Array.isArray(payload.plugins) ? payload.plugins.length : 0} plugins.`;
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
      return {
        plugins: Array.isArray(response.plugins) ? response.plugins : [],
      };
    }
    default:
      throw new GkillApiError(`Unknown tool: ${name}`);
  }
}

// isPluginPayload は get_kyous のペイロードがプラグイン由来かを判定する。
// 本文取得には rep_name と kyou_id の両方が要るので、揃っていないものは対象外にする。
function isPluginPayload(value) {
  return (
    value !== null &&
    typeof value === "object" &&
    value.kind === "plugin" &&
    typeof value.rep_name === "string" &&
    value.rep_name !== "" &&
    typeof value.kyou_id === "string" &&
    value.kyou_id !== ""
  );
}

/**
 * collectPluginPayloads は kyous[] から kind:"plugin" のペイロードを取得順に集める。
 *
 * @param {unknown} kyous get_kyous のレスポンスの kyous 配列。
 * @returns {Array<object>} プラグインペイロードの配列 (元オブジェクトの参照)。
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
    if (isPluginPayload(kyou.payload)) {
      payloads.push(kyou.payload);
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
  for (const payload of payloads) {
    const key = `${payload.rep_name} ${payload.kyou_id}`;
    const hit = entryByKey.get(key);
    if (hit) {
      hit.payloads.push(payload);
      continue;
    }
    if (entryByKey.size >= maxKyous) {
      markSkipped([payload], "max_kyous");
      continue;
    }
    const entry = { rep_name: payload.rep_name, kyou_id: payload.kyou_id, payloads: [payload] };
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
