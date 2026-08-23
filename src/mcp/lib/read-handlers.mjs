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
} from "./constants.mjs";
import { normalizeKyouArgs, normalizeLocaleOnlyArgs, normalizeGpsArgs, normalizeIdfFileArgs } from "./normalization.mjs";
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
            include_id: normalized.include_id || false,
            include_rep_name: normalized.include_rep_name || false,
          },
          true,
          ctx.sid,
        );
        const payload = {
          kyous: Array.isArray(response.kyous) ? response.kyous : [],
          total_count: response.total_count ?? 0,
          returned_count: response.returned_count ?? 0,
          has_more: Boolean(response.has_more),
          ...(response.next_cursor ? { next_cursor: response.next_cursor } : {}),
          // M-05: 付随データの取得が一部失敗したら、結果が不完全なことを明示する
          // (AIが「欠落のない完全な記録」と誤解しないため)。
          ...(response.partial ? { partial: true, warnings: Array.isArray(response.warnings) ? response.warnings : [] } : {}),
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
        return {
          gps_logs: Array.isArray(response.gps_logs) ? response.gps_logs : [],
        };
      }
      case "gkill_get_application_config": {
        const normalized = normalizeLocaleOnlyArgs(args);
        const response = await ctx.client.callApi(
          "/api/get_application_config",
          normalized,
          true,
          ctx.sid,
        );
        const config = response.application_config || {};
        return {
          tag_struct: config.tag_struct,
          mi_board_struct: config.mi_board_struct,
          rep_struct: config.rep_struct,
          rep_type_struct: config.rep_type_struct,
          device_struct: config.device_struct,
          kftl_template_struct: config.kftl_template_struct,
          mi_default_board: config.mi_default_board,
          show_tags_in_list: config.show_tags_in_list,
        };
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
      const returnedCount = payload.returned_count ?? 0;
      const totalCount = payload.total_count ?? returnedCount;
      const remaining = totalCount - returnedCount;
      const pluginSuffix = summarizeInlinePluginContent(payload.plugin_content);
      if (payload.has_more && payload.next_cursor) {
        return `Returned ${returnedCount} of ${totalCount} kyou entries (${remaining} remaining). Next page: cursor="${payload.next_cursor}".${pluginSuffix}`;
      }
      return `Returned ${returnedCount} of ${totalCount} kyou entries (all results returned).${pluginSuffix}`;
    }
    case "gkill_get_mi_board_list":
      return `Fetched ${Array.isArray(payload.boards) ? payload.boards.length : 0} Mi boards.`;
    case "gkill_get_all_tag_names":
      return `Fetched ${Array.isArray(payload.tag_names) ? payload.tag_names.length : 0} tag names.`;
    case "gkill_get_all_rep_names":
      return `Fetched ${Array.isArray(payload.rep_names) ? payload.rep_names.length : 0} repository names.`;
    case "gkill_get_gps_log":
      return `Fetched ${Array.isArray(payload.gps_logs) ? payload.gps_logs.length : 0} GPS log entries.`;
    case "gkill_get_application_config":
      return "Fetched application configuration.";
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
