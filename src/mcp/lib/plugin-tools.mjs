// プラグイン関連のMCPツール定義とハンドラ。
//
// read / write / readwrite の3サーバから共有する。サーバごとにAPI呼び出しの
// メソッド名が違う (callRead / callWrite / callApi) ため、呼び出し口は
// call(pathname, body) 形式のコールバックで受け取る。

import { GkillApiError } from "./errors.mjs";
import { normalizeLocaleOnlyArgs, normalizePluginContentArgs } from "./normalization.mjs";
import { htmlToText } from "./html-text.mjs";

export const GET_PLUGIN_LIST_ENDPOINT = "/api/get_plugin_list";
export const GET_PLUGIN_CONTENT_HTML_ENDPOINT = "/api/get_plugin_content_html";

export const PLUGIN_TOOL_NAMES = ["gkill_get_plugin_list", "gkill_get_plugin_content"];

export const PLUGIN_TOOLS = [
  {
    name: "gkill_get_plugin_list",
    description:
      "List the gkill plugins installed for the current user. Plugins are external programs that feed their own " +
      "records (kyou) into gkill — for example Claude Code / Claude.ai / ChatGPT conversation logs. " +
      "Each entry has name, version, description, data_type (the data_type its kyous carry), rep_name " +
      "(the repository name its kyous carry) and is_alive (whether the plugin process currently responds). " +
      "Use this to discover which data_type / rep_name values belong to plugins, then filter gkill_get_kyous " +
      "with query.reps (use_reps:true) or query.rep_types to fetch only that plugin's records. " +
      "Response fields: plugins[].",
    inputSchema: {
      type: "object",
      properties: {
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
      },
      additionalProperties: false,
    },
  },
  {
    name: "gkill_get_plugin_content",
    description:
      "Get the full content of a single plugin kyou. Plugin records keep their body inside the plugin itself, " +
      "not in gkill's database, so gkill_get_kyous returns only the metadata for them — this tool is the only " +
      "way to read the actual body (e.g. the whole text of a Claude Code message or a ChatGPT turn). " +
      "Find plugin kyous with gkill_get_kyous: their payload has kind='plugin' and carries rep_name and kyou_id; " +
      "pass those two values here. " +
      "By default the plugin's HTML is converted to plain text (format='text') because the raw HTML is mostly " +
      "presentation CSS/JS; pass format='html' or 'both' when you need the markup itself. " +
      "Response fields: rep_name, kyou_id, format, text (when format is text/both), html (when format is html/both), " +
      "text_truncated, html_size_bytes.",
    inputSchema: {
      type: "object",
      properties: {
        rep_name: {
          type: "string",
          description: "Repository name of the plugin, from the plugin payload's rep_name (or gkill_get_plugin_list).",
        },
        kyou_id: {
          type: "string",
          description: "ID of the plugin kyou, from the plugin payload's kyou_id.",
        },
        format: {
          type: "string",
          description:
            "Return format. 'text' (default) returns the HTML converted to plain text, 'html' returns the raw HTML, 'both' returns both.",
          enum: ["text", "html", "both"],
          default: "text",
        },
        max_text_length: {
          type: "integer",
          description:
            "Max characters of the converted text. Longer content is truncated and text_truncated becomes true. Default: 20000.",
        },
        locale_name: { type: "string", description: "Locale, e.g. ja/en." },
      },
      required: ["rep_name", "kyou_id"],
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
    case "gkill_get_plugin_content": {
      const parts = [`Fetched plugin content for ${payload.rep_name}/${payload.kyou_id}`];
      if (typeof payload.text === "string") {
        parts.push(`${payload.text.length} chars of text${payload.text_truncated ? " (truncated)" : ""}`);
      }
      if (typeof payload.html === "string") {
        parts.push(`${payload.html_size_bytes} bytes of HTML`);
      }
      return `${parts.join(", ")}.`;
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
      return {
        plugins: Array.isArray(response.plugins) ? response.plugins : [],
      };
    }
    case "gkill_get_plugin_content": {
      const normalized = normalizePluginContentArgs(args);
      const response = await call(GET_PLUGIN_CONTENT_HTML_ENDPOINT, {
        rep_name: normalized.rep_name,
        kyou_id: normalized.kyou_id,
        ...(normalized.locale_name ? { locale_name: normalized.locale_name } : {}),
      });
      const html = typeof response.html === "string" ? response.html : "";
      const payload = {
        rep_name: normalized.rep_name,
        kyou_id: normalized.kyou_id,
        format: normalized.format,
        html_size_bytes: Buffer.byteLength(html, "utf8"),
      };
      if (normalized.format === "text" || normalized.format === "both") {
        const converted = htmlToText(html, { maxLength: normalized.max_text_length });
        payload.text = converted.text;
        payload.text_truncated = converted.truncated;
      }
      if (normalized.format === "html" || normalized.format === "both") {
        payload.html = html;
      }
      return payload;
    }
    default:
      throw new GkillApiError(`Unknown tool: ${name}`);
  }
}
