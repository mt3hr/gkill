// 書き込みツールのディスパッチと要約。write / readwrite の2サーバが共有する。
//
// 以前は write と readwrite に逐語コピーされており、add/update/delete の20ケース
// 約500行がバイト単位で同一だった。読み取り側は同じ形で
// gkill_get_idf_file_path だけが片側だけ古いまま静かに壊れた前例があるので
// (lib/read-handlers.mjs の冒頭を参照)、書き込み側も同じ形で1箇所へ寄せる。
// 実装はこの1箇所が正本で、サーバ側は isWriteToolName / handleWriteToolCall /
// summarizeWriteToolPayload へ委譲するだけにする。
//
// ツール定義の配列 (TOOLS) は各サーバファイルに残すこと。
// src/tools/verify_docs.mjs がサーバファイル中の `const TOOLS = [...]` リテラルを
// 走査してツール数を数えており、ここへ移すとツール数が黙って0になる。

import crypto from "node:crypto";

import { GkillApiError } from "./errors.mjs";
import { WRITE_TOOLS } from "./write-tools.mjs";
import { DELETE_TARGETS } from "./constants.mjs";
import {
  normalizeKmemoArgs,
  normalizeUrlogArgs,
  normalizeNlogArgs,
  normalizeLantanaArgs,
  normalizeTimeIsArgs,
  normalizeMiArgs,
  normalizeKcArgs,
  normalizeTagArgs,
  normalizeTextArgs,
  normalizeKftlArgs,
  normalizeDeleteArgs,
  normalizeUpdateKmemoArgs,
  normalizeUpdateUrlogArgs,
  normalizeUpdateNlogArgs,
  normalizeUpdateLantanaArgs,
  normalizeUpdateTimeIsArgs,
  normalizeUpdateMiArgs,
  normalizeUpdateKcArgs,
  normalizeUpdateTagArgs,
  normalizeUpdateTextArgs,
} from "./write-normalization.mjs";

// 書き込み時に記録する端末名。サーバ種別によらず共通。
// アプリ名 (create_app / update_app) はサーバごとに違うので ctx.appName で受け取る。
const WRITE_DEVICE = "mcp";

// resolveDefaultBoardName は gkill_add_mi の board_name 未指定時に使う既定板名を返す。
// ApplicationConfig の mi_default_board が取れなければ "Inbox" へ落とす
// （gkill の初期値と同じ。空文字で送るよりは名前のある板に入るほうが回復しやすい）。
async function resolveDefaultBoardName(ctx, localeName) {
  try {
    const response = await ctx.client.callApi(
      "/api/get_application_config",
      localeName !== undefined ? { locale_name: localeName } : {},
      true, ctx.sid,
    );
    const configured = response?.application_config?.mi_default_board;
    if (typeof configured === "string" && configured.trim() !== "") {
      return configured;
    }
  } catch {
    // 既定板が引けないことを理由にタスク作成そのものを失敗させない
  }
  return "Inbox";
}

// stripUrlogImages は URLog の応答から画像の base64 を落とす。
//
// サーバは登録時に対象URLを取得して favicon とサムネイルを埋める。
// 保存されること自体は正しいが、応答に載せると1件あたり 1.5〜2.7KB の base64 が
// MCP クライアントのコンテキストを食う（3件登録しただけで約6KB）。
// 読み返し側 (gkill_get_kyous の urlog payload) には元から載っていないので、
// ここで落としても情報は失われない。
function stripUrlogImages(urlog) {
  if (!urlog) {
    return null;
  }
  const { favicon_image: _favicon, thumbnail_image: _thumbnail, ...rest } = urlog;
  return rest;
}

const WRITE_TOOL_NAMES = new Set(WRITE_TOOLS.map((tool) => tool.name));

// isWriteToolName は name が書き込みツールかを返す。
export function isWriteToolName(name) {
  return WRITE_TOOL_NAMES.has(name);
}

// handleWriteToolCall は書き込みツール1件を処理する。
// ctx = { client, sid, userId, appName }。client は GkillClient（callApi）。
// appName は create_app / update_app に載るサーバ種別名
// （write は "gkill_mcp_write"、readwrite は "gkill_mcp_readwrite"）。
export async function handleWriteToolCall(ctx, name, args) {
  switch (name) {
      // ----- Write tools -----
      case "gkill_add_kmemo": {
        const normalized = normalizeKmemoArgs(args);
        const now = new Date().toISOString();
        const kmemo = {
          id: crypto.randomUUID(),
          rep_name: "",
          related_time: normalized.related_time || now,
          content: normalized.content,
          data_type: "kmemo",
          create_time: now, create_app: ctx.appName,
          create_device: WRITE_DEVICE, create_user: ctx.userId,
          update_time: now, update_app: ctx.appName,
          update_device: WRITE_DEVICE, update_user: ctx.userId,
          is_deleted: false,
        };
        const response = await ctx.client.callApi(
          "/api/add_kmemo",
          { kmemo, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { added_kmemo: response.added_kmemo || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_urlog": {
        const normalized = normalizeUrlogArgs(args);
        const now = new Date().toISOString();
        const urlog = {
          id: crypto.randomUUID(),
          rep_name: "",
          related_time: normalized.related_time || now,
          url: normalized.url,
          title: normalized.title || "",
          image_base64: "",
          data_type: "urlog",
          create_time: now, create_app: ctx.appName,
          create_device: WRITE_DEVICE, create_user: ctx.userId,
          update_time: now, update_app: ctx.appName,
          update_device: WRITE_DEVICE, update_user: ctx.userId,
          is_deleted: false,
        };
        const response = await ctx.client.callApi(
          "/api/add_urlog",
          { urlog, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return {
          added_urlog: stripUrlogImages(response.added_urlog),
          added_kyou: response.added_kyou || null,
        };
      }

      case "gkill_add_nlog": {
        const normalized = normalizeNlogArgs(args);
        const now = new Date().toISOString();
        const nlog = {
          id: crypto.randomUUID(),
          rep_name: "",
          related_time: normalized.related_time || now,
          shop: normalized.shop || "",
          title: normalized.title,
          amount: normalized.amount,
          data_type: "nlog",
          create_time: now, create_app: ctx.appName,
          create_device: WRITE_DEVICE, create_user: ctx.userId,
          update_time: now, update_app: ctx.appName,
          update_device: WRITE_DEVICE, update_user: ctx.userId,
          is_deleted: false,
        };
        const response = await ctx.client.callApi(
          "/api/add_nlog",
          { nlog, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { added_nlog: response.added_nlog || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_lantana": {
        const normalized = normalizeLantanaArgs(args);
        const now = new Date().toISOString();
        const lantana = {
          id: crypto.randomUUID(),
          rep_name: "",
          related_time: normalized.related_time || now,
          mood: normalized.mood,
          data_type: "lantana",
          create_time: now, create_app: ctx.appName,
          create_device: WRITE_DEVICE, create_user: ctx.userId,
          update_time: now, update_app: ctx.appName,
          update_device: WRITE_DEVICE, update_user: ctx.userId,
          is_deleted: false,
        };
        const response = await ctx.client.callApi(
          "/api/add_lantana",
          { lantana, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { added_lantana: response.added_lantana || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_timeis": {
        const normalized = normalizeTimeIsArgs(args);
        const now = new Date().toISOString();
        const timeis = {
          id: crypto.randomUUID(),
          rep_name: "",
          title: normalized.title,
          start_time: normalized.start_time || now,
          end_time: normalized.end_time || null,
          data_type: "timeis",
          create_time: now, create_app: ctx.appName,
          create_device: WRITE_DEVICE, create_user: ctx.userId,
          update_time: now, update_app: ctx.appName,
          update_device: WRITE_DEVICE, update_user: ctx.userId,
          is_deleted: false,
        };
        const response = await ctx.client.callApi(
          "/api/add_timeis",
          { timeis, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { added_timeis: response.added_timeis || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_mi": {
        const normalized = normalizeMiArgs(args);
        // board_name 未指定ならアカウントの既定板へ入れる。
        // Go 側の AddMi に既定補完は無く (MiDefaultBoard を見るのは KFTL 経路だけ)、
        // 空文字のまま送ると名前の無い板にタスクが積まれてどの画面にも出てこない。
        // 新規アカウントでは gkill_get_mi_board_list が [] を返すので、
        // 板一覧ではなく ApplicationConfig から引く。
        const board_name = normalized.board_name !== undefined
          ? normalized.board_name
          : await resolveDefaultBoardName(ctx, normalized.locale_name);
        const now = new Date().toISOString();
        const mi = {
          id: crypto.randomUUID(),
          rep_name: "",
          title: normalized.title,
          is_checked: normalized.is_checked,
          board_name,
          limit_time: normalized.limit_time || null,
          estimate_start_time: normalized.estimate_start_time || null,
          estimate_end_time: normalized.estimate_end_time || null,
          data_type: "mi",
          create_time: now, create_app: ctx.appName,
          create_device: WRITE_DEVICE, create_user: ctx.userId,
          update_time: now, update_app: ctx.appName,
          update_device: WRITE_DEVICE, update_user: ctx.userId,
          is_deleted: false,
        };
        const response = await ctx.client.callApi(
          "/api/add_mi",
          { mi, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { added_mi: response.added_mi || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_kc": {
        const normalized = normalizeKcArgs(args);
        const now = new Date().toISOString();
        const kc = {
          id: crypto.randomUUID(),
          rep_name: "",
          related_time: normalized.related_time || now,
          title: normalized.title,
          num_value: normalized.num_value,
          data_type: "kc",
          create_time: now, create_app: ctx.appName,
          create_device: WRITE_DEVICE, create_user: ctx.userId,
          update_time: now, update_app: ctx.appName,
          update_device: WRITE_DEVICE, update_user: ctx.userId,
          is_deleted: false,
        };
        const response = await ctx.client.callApi(
          "/api/add_kc",
          { kc, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { added_kc: response.added_kc || null, added_kyou: response.added_kyou || null };
      }

      case "gkill_add_tag": {
        const normalized = normalizeTagArgs(args);
        const now = new Date().toISOString();
        const tag = {
          id: crypto.randomUUID(),
          rep_name: "",
          target_id: normalized.target_id,
          tag: normalized.tag,
          // 送らないと Go のゼロ値 (0001-01-01) がそのまま保存され、
          // 応答からも「いつの注記か」が読めなくなる
          related_time: now,
          data_type: "tag",
          create_time: now, create_app: ctx.appName,
          create_device: WRITE_DEVICE, create_user: ctx.userId,
          update_time: now, update_app: ctx.appName,
          update_device: WRITE_DEVICE, update_user: ctx.userId,
          is_deleted: false,
        };
        const response = await ctx.client.callApi(
          "/api/add_tag",
          { tag, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        // AddTagResponse に added_kyou は無い（常に null が返るだけだった）
        return { added_tag: response.added_tag || null };
      }

      case "gkill_add_text": {
        const normalized = normalizeTextArgs(args);
        const now = new Date().toISOString();
        const text = {
          id: crypto.randomUUID(),
          rep_name: "",
          target_id: normalized.target_id,
          text: normalized.text,
          // 送らないと Go のゼロ値 (0001-01-01) がそのまま保存される
          related_time: now,
          data_type: "text",
          create_time: now, create_app: ctx.appName,
          create_device: WRITE_DEVICE, create_user: ctx.userId,
          update_time: now, update_app: ctx.appName,
          update_device: WRITE_DEVICE, update_user: ctx.userId,
          is_deleted: false,
        };
        const response = await ctx.client.callApi(
          "/api/add_text",
          { text, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        // AddTextResponse に added_kyou は無い（常に null が返るだけだった）
        return { added_text: response.added_text || null };
      }

      case "gkill_submit_kftl": {
        const normalized = normalizeKftlArgs(args);
        const response = await ctx.client.callApi(
          "/api/submit_kftl_text",
          { kftl_text: normalized.kftl_text, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { messages: response.messages || [] };
      }

      case "gkill_delete_kyou": {
        const normalized = normalizeDeleteArgs(args);
        const target = DELETE_TARGETS[normalized.data_type];
        if (!target) {
          throw new GkillApiError(`Unsupported data_type for delete: ${normalized.data_type}`);
        }
        // 1. Fetch current entity to preserve all data fields
        const getResponse = await ctx.client.callApi(
          target.getEndpoint, { id: normalized.id }, true, ctx.sid,
        );
        const histories = getResponse[target.historiesKey];
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Entity not found: ${normalized.id}`);
        }
        const current = histories[0];
        // 2. Set is_deleted + update metadata
        const now = new Date().toISOString();
        current.is_deleted = true;
        current.update_time = now;
        current.update_app = ctx.appName;
        current.update_device = WRITE_DEVICE;
        current.update_user = ctx.userId;
        // 3. Send update
        const response = await ctx.client.callApi(
          target.updateEndpoint,
          { [target.requestKey]: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        // 4. Return current with is_deleted=true and all data preserved
        const result = {};
        result[target.responseKey] = current;
        if (response.updated_kyou) result.updated_kyou = response.updated_kyou;
        return result;
      }

      // ----- Update tools -----
      case "gkill_update_kmemo": {
        const normalized = normalizeUpdateKmemoArgs(args);
        const getResponse = await ctx.client.callApi(
          "/api/get_kmemo", { id: normalized.id }, true, ctx.sid,
        );
        const histories = getResponse.kmemo_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Kmemo not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.content !== undefined) current.content = normalized.content;
        if (normalized.related_time !== undefined) current.related_time = normalized.related_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = ctx.appName;
        current.update_device = WRITE_DEVICE;
        current.update_user = ctx.userId;
        const response = await ctx.client.callApi(
          "/api/update_kmemo",
          { kmemo: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { updated_kmemo: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_urlog": {
        const normalized = normalizeUpdateUrlogArgs(args);
        const getResponse = await ctx.client.callApi(
          "/api/get_urlog", { id: normalized.id }, true, ctx.sid,
        );
        const histories = getResponse.urlog_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Urlog not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.url !== undefined) current.url = normalized.url;
        if (normalized.title !== undefined) current.title = normalized.title;
        if (normalized.related_time !== undefined) current.related_time = normalized.related_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = ctx.appName;
        current.update_device = WRITE_DEVICE;
        current.update_user = ctx.userId;
        const response = await ctx.client.callApi(
          "/api/update_urlog",
          { urlog: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { updated_urlog: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_nlog": {
        const normalized = normalizeUpdateNlogArgs(args);
        const getResponse = await ctx.client.callApi(
          "/api/get_nlog", { id: normalized.id }, true, ctx.sid,
        );
        const histories = getResponse.nlog_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Nlog not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.title !== undefined) current.title = normalized.title;
        if (normalized.amount !== undefined) current.amount = normalized.amount;
        if (normalized.shop !== undefined) current.shop = normalized.shop;
        if (normalized.related_time !== undefined) current.related_time = normalized.related_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = ctx.appName;
        current.update_device = WRITE_DEVICE;
        current.update_user = ctx.userId;
        const response = await ctx.client.callApi(
          "/api/update_nlog",
          { nlog: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { updated_nlog: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_lantana": {
        const normalized = normalizeUpdateLantanaArgs(args);
        const getResponse = await ctx.client.callApi(
          "/api/get_lantana", { id: normalized.id }, true, ctx.sid,
        );
        const histories = getResponse.lantana_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Lantana not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.mood !== undefined) current.mood = normalized.mood;
        if (normalized.related_time !== undefined) current.related_time = normalized.related_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = ctx.appName;
        current.update_device = WRITE_DEVICE;
        current.update_user = ctx.userId;
        const response = await ctx.client.callApi(
          "/api/update_lantana",
          { lantana: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { updated_lantana: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_timeis": {
        const normalized = normalizeUpdateTimeIsArgs(args);
        const getResponse = await ctx.client.callApi(
          "/api/get_timeis", { id: normalized.id }, true, ctx.sid,
        );
        const histories = getResponse.timeis_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`TimeIs not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.title !== undefined) current.title = normalized.title;
        if (normalized.start_time !== undefined) current.start_time = normalized.start_time;
        if (normalized.end_time !== undefined) current.end_time = normalized.end_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = ctx.appName;
        current.update_device = WRITE_DEVICE;
        current.update_user = ctx.userId;
        const response = await ctx.client.callApi(
          "/api/update_timeis",
          { timeis: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { updated_timeis: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_mi": {
        const normalized = normalizeUpdateMiArgs(args);
        const getResponse = await ctx.client.callApi(
          "/api/get_mi", { id: normalized.id }, true, ctx.sid,
        );
        const histories = getResponse.mi_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Mi not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.title !== undefined) current.title = normalized.title;
        if (normalized.board_name !== undefined) current.board_name = normalized.board_name;
        if (normalized.is_checked !== undefined) current.is_checked = normalized.is_checked;
        if (normalized.limit_time !== undefined) current.limit_time = normalized.limit_time;
        if (normalized.estimate_start_time !== undefined) current.estimate_start_time = normalized.estimate_start_time;
        if (normalized.estimate_end_time !== undefined) current.estimate_end_time = normalized.estimate_end_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = ctx.appName;
        current.update_device = WRITE_DEVICE;
        current.update_user = ctx.userId;
        const response = await ctx.client.callApi(
          "/api/update_mi",
          { mi: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { updated_mi: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_kc": {
        const normalized = normalizeUpdateKcArgs(args);
        const getResponse = await ctx.client.callApi(
          "/api/get_kc", { id: normalized.id }, true, ctx.sid,
        );
        const histories = getResponse.kc_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`KC not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.title !== undefined) current.title = normalized.title;
        if (normalized.num_value !== undefined) current.num_value = normalized.num_value;
        if (normalized.related_time !== undefined) current.related_time = normalized.related_time;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = ctx.appName;
        current.update_device = WRITE_DEVICE;
        current.update_user = ctx.userId;
        const response = await ctx.client.callApi(
          "/api/update_kc",
          { kc: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { updated_kc: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_tag": {
        const normalized = normalizeUpdateTagArgs(args);
        const getResponse = await ctx.client.callApi(
          "/api/get_tag_histories_by_tag_id", { id: normalized.id }, true, ctx.sid,
        );
        const histories = getResponse.tag_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Tag not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.tag !== undefined) current.tag = normalized.tag;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = ctx.appName;
        current.update_device = WRITE_DEVICE;
        current.update_user = ctx.userId;
        const response = await ctx.client.callApi(
          "/api/update_tag",
          { tag: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { updated_tag: current, updated_kyou: response.updated_kyou || null };
      }

      case "gkill_update_text": {
        const normalized = normalizeUpdateTextArgs(args);
        const getResponse = await ctx.client.callApi(
          "/api/get_text_histories_by_text_id", { id: normalized.id }, true, ctx.sid,
        );
        const histories = getResponse.text_histories;
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(`Text not found: ${normalized.id}`);
        }
        const current = histories[0];
        if (normalized.text !== undefined) current.text = normalized.text;
        const now = new Date().toISOString();
        current.update_time = now;
        current.update_app = ctx.appName;
        current.update_device = WRITE_DEVICE;
        current.update_user = ctx.userId;
        const response = await ctx.client.callApi(
          "/api/update_text",
          { text: current, want_response_kyou: true, locale_name: normalized.locale_name },
          true, ctx.sid,
        );
        return { updated_text: current, updated_kyou: response.updated_kyou || null };
      }
      default:
        throw new GkillApiError(`Unknown tool: ${name}`);
  }
}

// summarizeWriteToolPayload は書き込みツールの結果要約を返す。対象外のツールは null。
export function summarizeWriteToolPayload(name, payload) {
  switch (name) {
    // Write tools
    case "gkill_add_kmemo":
      return `Created kmemo: ${payload.added_kmemo?.id || "unknown"}`;
    case "gkill_add_urlog":
      return `Created urlog: ${payload.added_urlog?.id || "unknown"}`;
    case "gkill_add_nlog":
      return `Created nlog: ${payload.added_nlog?.id || "unknown"}`;
    case "gkill_add_lantana":
      return `Created lantana: ${payload.added_lantana?.id || "unknown"}`;
    case "gkill_add_timeis":
      return `Created timeis: ${payload.added_timeis?.id || "unknown"}`;
    case "gkill_add_mi":
      return `Created mi: ${payload.added_mi?.id || "unknown"}`;
    case "gkill_add_kc":
      return `Created kc: ${payload.added_kc?.id || "unknown"}`;
    case "gkill_add_tag":
      return `Added tag: ${payload.added_tag?.id || "unknown"}`;
    case "gkill_add_text":
      return `Added text: ${payload.added_text?.id || "unknown"}`;
    case "gkill_submit_kftl":
      return `KFTL submitted: ${Array.isArray(payload.messages) ? payload.messages.length : 0} messages.`;
    case "gkill_delete_kyou": {
      const keys = Object.keys(payload).filter((k) => k.startsWith("updated_"));
      return `Deleted (soft): ${keys.length > 0 ? keys.join(", ") : "completed"}`;
    }
    // Update tools
    case "gkill_update_kmemo":
      return `Updated kmemo: ${payload.updated_kmemo?.id || "unknown"}`;
    case "gkill_update_urlog":
      return `Updated urlog: ${payload.updated_urlog?.id || "unknown"}`;
    case "gkill_update_nlog":
      return `Updated nlog: ${payload.updated_nlog?.id || "unknown"}`;
    case "gkill_update_lantana":
      return `Updated lantana: ${payload.updated_lantana?.id || "unknown"}`;
    case "gkill_update_timeis":
      return `Updated timeis: ${payload.updated_timeis?.id || "unknown"}`;
    case "gkill_update_mi":
      return `Updated mi: ${payload.updated_mi?.id || "unknown"}`;
    case "gkill_update_kc":
      return `Updated kc: ${payload.updated_kc?.id || "unknown"}`;
    case "gkill_update_tag":
      return `Updated tag: ${payload.updated_tag?.id || "unknown"}`;
    case "gkill_update_text":
      return `Updated text: ${payload.updated_text?.id || "unknown"}`;
    default:
      return null;
  }
}
