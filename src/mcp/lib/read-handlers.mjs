// 読み取りツールのディスパッチと要約。read / readwrite / write の3サーバが共有する。
//
// 以前は read と readwrite に逐語コピーされており、readwrite 側の
// gkill_get_idf_file_path だけが廃止済み client.callRead を呼び続けて
// 常に TypeError で静かに失敗していた（片側だけの直し漏れの温床）。
// 実装はこの1箇所が正本で、サーバ側は isReadToolName / handleReadToolCall /
// summarizeReadToolPayload へ委譲するだけにする。

import { GkillApiError } from "./errors.mjs";
import {
  MAX_IDF_FILE_BYTES,
  APP_CONFIG_UI_STATE_KEYS,
} from "./constants.mjs";
import { normalizeKyouArgs, normalizeLocaleOnlyArgs, normalizeGpsArgs, normalizeIdfFileArgs, normalizeAppConfigArgs } from "./normalization.mjs";
import { inlinePluginContents, summarizeInlinePluginContent } from "./plugin-tools.mjs";
import { normalizeMimeType } from "./payload.mjs";
import { READ_TOOLS } from "./read-tools.mjs";

const READ_TOOL_NAMES = new Set(READ_TOOLS.map((tool) => tool.name));

// isReadToolName は name が読み取りツールかを返す。
export function isReadToolName(name) {
  return READ_TOOL_NAMES.has(name);
}

// handleReadToolCall は読み取りツール1件を処理する。
// ctx = { client, ctx.sid, isLocalTransport }。client は GkillClient（callApi / fetchFile / login）。
export async function handleReadToolCall(ctx, name, args) {
  switch (name) {
      case "gkill_get_kyous": {
        const normalized = normalizeKyouArgs(args);
        const response = await ctx.client.callApi(
          "/api/get_kyous_mcp",
          {
            query: normalized.query,
            locale_name: normalized.locale_name,
            limit: normalized.limit,
            cursor: normalized.cursor,
            max_size_mb: normalized.max_size_mb,
            is_include_timeis: normalized.is_include_timeis,
            // ---- v2 (ADR-0053)。include_id / include_rep_name は廃止（常時付与） ----
            count_only: normalized.count_only || false,
            group_by: normalized.group_by,
            data_types: normalized.data_types,
            num_min: normalized.num_min,
            num_max: normalized.num_max,
            idf_kinds: normalized.idf_kinds,
            include_file_size: normalized.include_file_size || false,
          },
          true,
          ctx.sid,
        );
        const payload = {
          kyous: Array.isArray(response.kyous) ? response.kyous : [],
          // v2: total_count は cursor 無し応答（1ページ目・count_only・group_by）にのみ入る。
          // 旧v1の「?? 0」で埋める書き方はカーソルページで嘘の0を作るのでしない。
          ...(response.total_count !== undefined && response.total_count !== null
            ? { total_count: response.total_count }
            : {}),
          returned_count: response.returned_count ?? 0,
          remaining_count: response.remaining_count ?? 0,
          has_more: Boolean(response.has_more),
          ...(response.next_cursor ? { next_cursor: response.next_cursor } : {}),
          ...(Array.isArray(response.buckets) ? { buckets: response.buckets } : {}),
          // 警告は partial に限らず常設（未知フィルタ値の指摘等）。
          // partial は付随データ欠落専用の印として従来の意味を保つ (M-05)。
          ...(response.partial ? { partial: true } : {}),
          ...(Array.isArray(response.warnings) && response.warnings.length > 0
            ? { warnings: response.warnings }
            : {}),
        };
        if (normalized.include_plugin_content) {
          payload.plugin_content = await inlinePluginContents(
            (pathname, body) => ctx.client.callApi(pathname, body, true, ctx.sid),
            payload.kyous,
            {
              maxTextLength: normalized.plugin_content_max_text_length,
              format: normalized.plugin_content_format,
              localeName: normalized.locale_name,
            },
          );
        }
        return payload;
      }
      case "gkill_get_mi_board_list": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await ctx.client.callApi("/api/get_mi_board_list", normalized, true, ctx.sid);
        return {
          boards: Array.isArray(response.boards) ? response.boards : [],
        };
      }
      case "gkill_get_all_tag_names": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await ctx.client.callApi("/api/get_all_tag_names", normalized, true, ctx.sid);
        return {
          tag_names: Array.isArray(response.tag_names) ? response.tag_names : [],
        };
      }
      case "gkill_get_all_rep_names": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await ctx.client.callApi("/api/get_all_rep_names", normalized, true, ctx.sid);
        return {
          rep_names: Array.isArray(response.rep_names) ? response.rep_names : [],
        };
      }
      case "gkill_get_gps_log": {
        const normalized = normalizeGpsArgs(args);
        const response = await ctx.client.callApi(
          "/api/get_gps_log",
          {
            start_date: normalized.start_date,
            end_date: normalized.end_date,
            locale_name: normalized.locale_name,
          },
          true,
          ctx.sid,
        );
        // ページングは Node 側実装。gkill は期間内の全点を（時刻降順・重複排除済みで）返す。
        // 1日分ですら応答上限を超える時期があり（2026-08実測で1,595点/日）、
        // limit/cursor 無しでは実質使えなかった（外部監査 B5）。
        const gpsLogs = Array.isArray(response.gps_logs) ? response.gps_logs : [];
        return paginateGpsLogs(gpsLogs, normalized);
      }
      case "gkill_get_rep_infos": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await ctx.client.callApi("/api/get_rep_infos_mcp", normalized, true, ctx.sid);
        return {
          rep_infos: Array.isArray(response.rep_infos) ? response.rep_infos : [],
          canonical_rep_types: Array.isArray(response.canonical_rep_types) ? response.canonical_rep_types : [],
          plugins: Array.isArray(response.plugins) ? response.plugins : [],
        };
      }
      case "gkill_get_application_config": {
        const normalized = normalizeAppConfigArgs(args);
        const response = await ctx.client.callApi(
          "/api/get_application_config",
          normalized.locale_name !== undefined ? { locale_name: normalized.locale_name } : {},
          true,
          ctx.sid,
        );
        const config = response.application_config || {};
        const full = {
          tag_struct: config.tag_struct,
          mi_board_struct: config.mi_board_struct,
          rep_struct: config.rep_struct,
          rep_type_struct: config.rep_type_struct,
          device_struct: config.device_struct,
          kftl_template_struct: config.kftl_template_struct,
          mi_default_board: config.mi_default_board,
          show_tags_in_list: config.show_tags_in_list,
        };
        // fields 射影（実測で全量193.6k字＝応答上限超過。tag_struct 単体なら44.6k字）
        let projected = full;
        if (normalized.fields) {
          projected = {};
          for (const field of normalized.fields) {
            projected[field] = full[field];
          }
        }
        // UI 状態キー（ツリーエディタの一時状態）を既定で剥がす。
        // check_when_inited / is_force_hide は可視タグ判定に必要なので strip 対象に入れない
        if (!normalized.include_ui_state) {
          projected = stripAppConfigUiState(projected);
        }
        return projected;
      }
      case "gkill_get_idf_file": {
        const normalized = normalizeIdfFileArgs(args);
        const filePath =
          "/files/" +
          encodeURIComponent(normalized.rep_name) +
          "/" +
          normalized.file_name
            .split("/")
            .map((s) => encodeURIComponent(s))
            .join("/");
        const fileSid = ctx.sid || (await ctx.client.login());
        const { buffer, contentType } = await ctx.client.fetchFile(filePath, fileSid);
        // base64はJSON-RPCレスポンスに素で載るので、青天井にすると数百MBの動画で応答が破裂する
        if (buffer.length > MAX_IDF_FILE_BYTES) {
          throw new GkillApiError(
            `File is too large to return through MCP: ${buffer.length} bytes (limit ${MAX_IDF_FILE_BYTES}). ` +
              `Use gkill_get_idf_file_path to get the local path and read the file from the filesystem instead.`,
            {
              file_name: normalized.file_name,
              file_size_bytes: buffer.length,
              max_bytes: MAX_IDF_FILE_BYTES,
            },
          );
        }
        const mimeType = normalizeMimeType(contentType);
        return {
          file_name: normalized.file_name,
          mime_type: mimeType,
          file_size_bytes: buffer.length,
          is_image: mimeType.startsWith("image/"),
          file_content_base64: buffer.toString("base64"),
        };
      }
      case "gkill_get_idf_file_path": {
        const normalized = normalizeIdfFileArgs(args);
        // 絶対パスは同一マシンのクライアントにしか意味がない。
        // リモートクライアントに渡すとユーザのディレクトリ構造の漏洩になるので、gkillに問い合わせもしない。
        if (!ctx.isLocalTransport) {
          throw new GkillApiError(
            "Local file paths are available only to MCP clients running on the same machine (stdio transport). " +
              "Use gkill_get_idf_file to fetch the file content instead.",
          );
        }
        const response = await ctx.client.callApi(
          "/api/get_idf_file_path",
          {
            rep_name: normalized.rep_name,
            file_name: normalized.file_name,
            locale_name: normalized.locale_name,
          },
          true,
          ctx.sid,
        );
        return {
          rep_name: normalized.rep_name,
          file_name: normalized.file_name,
          file_path: response.file_path || "",
          exists: Boolean(response.exists),
        };
      }
    default:
      throw new GkillApiError(`Unknown read tool: ${name}`);
  }
}

// summarizeReadToolPayload は読み取りツールの結果要約を返す。対象外のツールは null。
export function summarizeReadToolPayload(name, payload) {
  switch (name) {
    case "gkill_get_kyous": {
      // v2: total_count は cursor 無し応答にのみ入る。残量の真実は remaining_count。
      // 旧実装の `total_count ?? returned_count` はカーソルページで
      // 「all results returned」と嘘の完了報告をしていた。
      if (Array.isArray(payload.buckets)) {
        return `Aggregated ${payload.buckets.length} buckets (${payload.total_count ?? 0} entries).`;
      }
      const returnedCount = payload.returned_count ?? 0;
      const remaining = payload.remaining_count ?? 0;
      if ((payload.kyous?.length ?? 0) === 0 && payload.total_count !== undefined && returnedCount === 0 && !payload.has_more) {
        return `Counted ${payload.total_count} entries.`;
      }
      const pluginSuffix = summarizeInlinePluginContent(payload.plugin_content);
      const totalPart = payload.total_count !== undefined ? ` of ${payload.total_count}` : "";
      if (payload.has_more && payload.next_cursor) {
        return `Returned ${returnedCount}${totalPart} kyou entries (${remaining} remaining). Next page: cursor="${payload.next_cursor}".${pluginSuffix}`;
      }
      return `Returned ${returnedCount}${totalPart} kyou entries (0 remaining).${pluginSuffix}`;
    }
    case "gkill_get_mi_board_list":
      return `Fetched ${Array.isArray(payload.boards) ? payload.boards.length : 0} Mi boards.`;
    case "gkill_get_all_tag_names":
      return `Fetched ${Array.isArray(payload.tag_names) ? payload.tag_names.length : 0} tag names.`;
    case "gkill_get_all_rep_names":
      return `Fetched ${Array.isArray(payload.rep_names) ? payload.rep_names.length : 0} repository names.`;
    case "gkill_get_gps_log": {
      if (Array.isArray(payload.buckets)) {
        return `Aggregated ${payload.buckets.length} daily buckets (${payload.total_count ?? 0} GPS points).`;
      }
      const returned = Array.isArray(payload.gps_logs) ? payload.gps_logs.length : 0;
      if (returned === 0 && payload.total_count !== undefined && !payload.has_more) {
        return `Counted ${payload.total_count} GPS points.`;
      }
      if (payload.has_more && payload.next_cursor) {
        return `Returned ${returned} GPS points (${payload.remaining_count ?? 0} remaining). Next page: cursor="${payload.next_cursor}".`;
      }
      return `Returned ${returned} GPS points (0 remaining).`;
    }
    case "gkill_get_application_config":
      return `Fetched application configuration (${Object.keys(payload ?? {}).length} fields).`;
    case "gkill_get_rep_infos": {
      const repCount = Array.isArray(payload.rep_infos) ? payload.rep_infos.length : 0;
      const typeCount = Array.isArray(payload.canonical_rep_types) ? payload.canonical_rep_types.length : 0;
      return `Fetched ${repCount} repositories (${typeCount} canonical rep types).`;
    }
    case "gkill_get_idf_file":
      return `Retrieved file: ${payload.file_name} (${payload.file_size_bytes} bytes, ${payload.mime_type})`;
    case "gkill_get_idf_file_path":
      return payload.exists
        ? `Resolved local file path: ${payload.file_path}`
        : `File not found in repository (no local path available).`;
    default:
      return null;
  }
}

// stripAppConfigUiState は struct ツリーから UI 状態キーを再帰的に剥がす。
// 対象キーは APP_CONFIG_UI_STATE_KEYS（check_when_inited / is_force_hide は含めない）。
export function stripAppConfigUiState(value) {
  if (Array.isArray(value)) {
    return value.map((item) => stripAppConfigUiState(item));
  }
  if (value !== null && typeof value === "object") {
    const out = {};
    for (const [key, child] of Object.entries(value)) {
      if (APP_CONFIG_UI_STATE_KEYS.has(key)) {
        continue;
      }
      out[key] = stripAppConfigUiState(child);
    }
    return out;
  }
  return value;
}

// GPSカーソル: base64url(JSON {t, n})。t=最後に返した点の related_time、
// n=同一時刻の中で消費済みの点数。サーバの並びは（時刻降順・座標タイブレーク）で
// 決定的なので、同一時刻ランの途中でも位置を特定できる（get_kyous v2 と同じ考え方）。
export function encodeGpsCursor(t, n) {
  return Buffer.from(JSON.stringify({ t, n }), "utf8").toString("base64url");
}

export function decodeGpsCursor(cursor) {
  try {
    const decoded = JSON.parse(Buffer.from(cursor, "base64url").toString("utf8"));
    if (typeof decoded.t === "string" && Number.isInteger(decoded.n) && decoded.n >= 0) {
      return decoded;
    }
  } catch {
    // fallthrough
  }
  throw new GkillApiError(`Invalid GPS cursor: ${JSON.stringify(cursor)} (pass next_cursor verbatim)`);
}

// paginateGpsLogs は取得済みの全点列に limit/cursor/count_only/group_by を適用する。
export function paginateGpsLogs(gpsLogs, options) {
  if (options.count_only && options.cursor) {
    throw new GkillApiError("count_only cannot be combined with cursor");
  }
  if (options.group_by && options.cursor) {
    throw new GkillApiError("group_by cannot be combined with cursor");
  }
  if (options.count_only) {
    return { gps_logs: [], total_count: gpsLogs.length, returned_count: 0, remaining_count: 0, has_more: false };
  }
  if (options.group_by === "day") {
    // 日別カバレッジ（キーは点の related_time のローカル日付）
    const counts = new Map();
    for (const point of gpsLogs) {
      const at = new Date(point.related_time);
      const key = `${at.getFullYear()}-${String(at.getMonth() + 1).padStart(2, "0")}-${String(at.getDate()).padStart(2, "0")}`;
      counts.set(key, (counts.get(key) || 0) + 1);
    }
    const buckets = [...counts.entries()]
      .map(([key, count]) => ({ key, count }))
      .sort((a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0));
    return { gps_logs: [], buckets, total_count: gpsLogs.length, returned_count: 0, remaining_count: 0, has_more: false };
  }

  let startIndex = 0;
  if (options.cursor) {
    const cursor = decodeGpsCursor(options.cursor);
    let sameTimeSeen = 0;
    startIndex = gpsLogs.length;
    for (let i = 0; i < gpsLogs.length; i++) {
      const t = gpsLogs[i].related_time;
      if (t === cursor.t) {
        sameTimeSeen++;
        if (sameTimeSeen > cursor.n) {
          startIndex = i;
          break;
        }
        continue;
      }
      // サーバは時刻降順なので、カーソル時刻より古い点が最初に現れた位置から再開
      if (new Date(t).getTime() < new Date(cursor.t).getTime()) {
        startIndex = i;
        break;
      }
    }
  }

  const batch = gpsLogs.slice(startIndex);
  const page = batch.slice(0, options.limit);
  const remaining = batch.length - page.length;
  const payload = {
    gps_logs: page,
    returned_count: page.length,
    remaining_count: remaining,
    has_more: remaining > 0,
  };
  if (!options.cursor) {
    payload.total_count = gpsLogs.length;
  }
  if (remaining > 0 && page.length > 0) {
    const lastTime = page[page.length - 1].related_time;
    let n = 0;
    for (let i = startIndex; i < startIndex + page.length; i++) {
      if (gpsLogs[i].related_time === lastTime) {
        n++;
      }
    }
    // カーソル位置は「同一時刻ランの先頭からの消費数」で表す必要がある。
    // ランがページ境界をまたぐとき、前ページで消費した分も数えないと重複して返す
    let priorSameTime = 0;
    for (let i = startIndex - 1; i >= 0; i--) {
      if (gpsLogs[i].related_time === lastTime) {
        priorSameTime++;
      } else {
        break;
      }
    }
    payload.next_cursor = encodeGpsCursor(lastTime, priorSameTime + n);
  }
  return payload;
}
