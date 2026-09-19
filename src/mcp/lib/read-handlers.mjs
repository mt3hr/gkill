// 読み取りツールのディスパッチと要約。read / readwrite / write の3サーバが共有する。
//
// 以前は read と readwrite に逐語コピーされており、readwrite 側の
// IDFファイルパス取得の1ツールだけが廃止済み client.callRead を呼び続けて
// 常に TypeError で静かに失敗していた（片側だけの直し漏れの温床）。
// 実装はこの1箇所が正本で、サーバ側は isReadToolName / handleReadToolCall /
// summarizeReadToolPayload へ委譲するだけにする。

import { GkillApiError, invalidArgument } from "./errors.mjs";
import {
  MAX_IDF_FILE_BYTES,
  APP_CONFIG_FIELDS,
  REP_INFOS_FIELDS,
  APP_CONFIG_UI_STATE_KEYS,
  ENTITY_TARGETS,
} from "./constants.mjs";
import { normalizeKyouArgs, normalizeLocaleOnlyArgs, normalizeGpsArgs, normalizeIdfFileArgs, normalizeAppConfigArgs, normalizeKyouHistoryArgs, normalizeRepNamesArgs, normalizeTagNamesArgs, normalizeRepInfosArgs, normalizeStatusArgs, normalizeMcpHelpArgs, appendStaleSchemaWarning, assertAggregationNotCombinedWithCursor, formatLocalRfc3339 } from "./normalization.mjs";
import { buildHelpPayload } from "./help-topics.mjs";
import { inlinePluginContents, summarizeInlinePluginContent } from "./plugin-tools.mjs";
import { normalizeMimeType, entityNotFoundMessage, appendStaleSchemaNoteToSummary, MINT_FILE_LINKS } from "./payload.mjs";
import { READ_TOOLS } from "./read-tools.mjs";
import { encodeGpsCursor, decodeGpsCursor } from "./gps-cursor.mjs";

// コーデックの正本は gps-cursor.mjs。ここからの re-export は既存の import 元を保つため。
export { encodeGpsCursor, decodeGpsCursor };

const READ_TOOL_NAMES = new Set(READ_TOOLS.map((tool) => tool.name));

// isReadToolName は name が読み取りツールかを返す。
export function isReadToolName(name) {
  return READ_TOOL_NAMES.has(name);
}

// handleReadToolCall は読み取りツール1件を処理する。
// ctx = { client, ctx.sid, isLocalTransport }。client は GkillClient（callApi / fetchFile / login）。
//
// ディスパッチ本体を包んで、古いツールスキーマを掴んだクライアントへの警告を
// **1箇所で**足す。ツールごとに書くと必ず足し忘れる。
export async function handleReadToolCall(ctx, name, args) {
  const payload = await dispatchReadToolCall(ctx, name, args);
  return appendStaleSchemaWarning(payload, name, args);
}

async function dispatchReadToolCall(ctx, name, args) {
  switch (name) {
      case "gkill_status": {
        normalizeStatusArgs(args);
        return buildStatusPayload(ctx);
      }
      case "gkill_get_mcp_help": {
        // 静的な本文。gkill へは往復しない（ADR-0622）。
        const normalized = normalizeMcpHelpArgs(args);
        return buildHelpPayload(normalized.topic);
      }
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
            // ---- v2 (ADR-0604)。include_id / include_rep_name は廃止（常時付与） ----
            count_only: normalized.count_only || false,
            group_by: normalized.group_by,
            data_types: normalized.data_types,
            create_apps: normalized.create_apps,
            update_apps: normalized.update_apps,
            num_min: normalized.num_min,
            num_max: normalized.num_max,
            idf_kinds: normalized.idf_kinds,
            include_file_size: normalized.include_file_size || false,
            // tag_entities / text_entities は頼まれたときだけ組ませる（ADR-0629）
            include_attached_ids: normalized.include_attached_ids || false,
          },
          true,
          ctx.sid,
        );
        // 入口で補った既定（for_mi の include_create_mi）は Go の warnings と同じ列に並べる。
        const warnings = [
          ...(Array.isArray(normalized.notes) ? normalized.notes : []),
          ...(Array.isArray(response.warnings) ? response.warnings : []),
        ];
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
          // プラグインの説明は各 Kyou へ焼き込まず、応答トップレベルへ rep 名ごと1回だけ
          // 載せる設計 (kyou_mcp_dto.go)。Go は返していたのにここがコピーしておらず、
          // 「各 Kyou にもトップレベルにも説明が無い」状態だった (2026-08-30 レビュー P1)。
          ...(Array.isArray(response.plugins) && response.plugins.length > 0
            ? { plugins: response.plugins }
            : {}),
          // 警告は partial に限らず常設（未知フィルタ値の指摘等）。
          // partial は付随データ欠落専用の印として従来の意味を保つ (M-05)。
          ...(response.partial ? { partial: true } : {}),
          ...(warnings.length > 0 ? { warnings } : {}),
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
          // 本文を足した後の実サイズで max_size_mb を守り直す（Go は本文の無い DTO しか測れない。ADR-0624）
          enforceKyousSizeBudget(payload, normalized.max_size_mb);
        }
        if (normalized.include_file_urls) {
          // HTTP のときだけ buildToolResult が公開URLを鋳造する印（JSON には出ない。ADR-0630）
          payload[MINT_FILE_LINKS] = true;
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
        // 絞り込みは Node 側。gkill は全件を返すので、contains / limit は送らない。
        const normalized = normalizeTagNamesArgs(args);
        const response = await ctx.client.callApi(
          "/api/get_all_tag_names",
          normalized.locale_name === undefined ? {} : { locale_name: normalized.locale_name },
          true,
          ctx.sid,
        );
        return paginateTagNames(
          Array.isArray(response.tag_names) ? response.tag_names : [],
          normalized,
        );
      }
      case "gkill_get_all_rep_names": {
        const normalized = normalizeRepNamesArgs(args);
        const response = await ctx.client.callApi(
          "/api/get_all_rep_names",
          normalized.locale_name === undefined ? {} : { locale_name: normalized.locale_name },
          true,
          ctx.sid,
        );
        // 絞り込みは Node 側実装。gkill は Reps の全名を返す（この規模の環境では数百件あり、
        // 「その名前の rep があるか」を確かめるだけで全件を読むことになっていた）。
        const repNames = Array.isArray(response.rep_names) ? response.rep_names : [];
        return paginateRepNames(repNames, normalized);
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
        const normalized = normalizeRepInfosArgs(args);
        const response = await ctx.client.callApi(
          "/api/get_rep_infos_mcp",
          normalized.locale_name === undefined ? {} : { locale_name: normalized.locale_name },
          true,
          ctx.sid,
        );
        const full = {
          rep_infos: Array.isArray(response.rep_infos) ? response.rep_infos : [],
          canonical_rep_types: Array.isArray(response.canonical_rep_types) ? response.canonical_rep_types : [],
          plugins: Array.isArray(response.plugins) ? response.plugins : [],
          // タグ・テキスト・通知・GPSログの格納先。rep_infos とは用途が違い、
          // query.reps へ渡すと Kyou の rep_name と一致せず静かに0件になる。
          attached_data_reps: Array.isArray(response.attached_data_reps) ? response.attached_data_reps : [],
        };
        // data_kinds 絞り込み。本番では約120件（歴代端末ぶんの Tag_ / Text_ / Notification_ / GPSLogs_）。
        if (normalized.data_kinds) {
          const wanted = new Set(normalized.data_kinds);
          full.attached_data_reps = full.attached_data_reps.filter((rep) => wanted.has(rep?.data_kind));
        }
        applyRepInfosRowFilters(full, normalized);
        // fields 射影。rep_infos[] だけで本番は数百件になるのに、
        // 「正準値と対応表だけ欲しい」呼び出しが多かった。
        // 許可リストの照合は normalizeRepInfosArgs も行うが、動的なプロパティ書き込みの
        // 直前でも弾く（app_config 側と同じ理由。CodeQL js/remote-property-injection #932 は
        // 書き込みと同じ関数内の Set.has ガードしか認識しない）。
        if (!normalized.fields) {
          return full;
        }
        const projected = {};
        for (const field of normalized.fields) {
          if (!REP_INFOS_FIELDS.has(field)) {
            continue;
          }
          projected[field] = full[field];
        }
        return projected;
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
          // 接続先の識別。gkill は元から返しているのに、この射影が捨てていた。
          user_id: config.user_id,
          device: config.device,
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
        // 許可リストの照合は normalizeAppConfigArgs も行う（外れた値はそこで例外になる）が、
        // 動的なプロパティ書き込みの直前でも弾く。CodeQL の js/remote-property-injection は
        // バリアガードが関数境界を越えず、Object.prototype.hasOwnProperty.call も認識しない。
        // 認識される唯一の形が「書き込みと同じ関数内の Set.has ガード」なので、この行を消すと
        // アラートが再発する（#932）。
        let projected = full;
        if (normalized.fields) {
          projected = {};
          for (const field of normalized.fields) {
            if (!APP_CONFIG_FIELDS.has(field)) {
              continue;
            }
            projected[field] = full[field];
          }
        }
        // UI 状態キー（ツリーエディタの一時状態）を既定で剥がす。
        // check_when_inited / is_force_hide は可視タグ判定に必要なので strip 対象に入れない
        if (!normalized.include_ui_state) {
          projected = stripAppConfigUiState(projected);
        }
        // contains で葉を刈り、compact で既定値の欄を落とし、max_size_mb に必ず収める（ADR-0629）。
        if (normalized.contains !== undefined) {
          projected = filterAppConfigStructs(projected, normalized.contains);
        }
        if (normalized.compact) {
          projected = compactAppConfigStructs(projected);
        }
        return capAppConfigSize(projected, normalized.max_size_mb);
      }
      case "gkill_get_idf_file": {
        const normalized = normalizeIdfFileArgs(args);
        // クエリの形は他の実装と揃える: ?is_video=true&thumb=WxH
        // (payload.mjs の file_url 注入 / http-transport.mjs の /files/ 配信 /
        //  クライアントの use-idf-kyou-view.ts build_media_url と同じ)。
        const fileQuery = [];
        if (normalized.is_video) {
          fileQuery.push("is_video=true");
        }
        if (normalized.thumb !== undefined) {
          fileQuery.push(`thumb=${normalized.thumb}`);
        }
        const filePath =
          "/files/" +
          encodeURIComponent(normalized.rep_name) +
          "/" +
          normalized.file_name
            .split("/")
            .map((s) => encodeURIComponent(s))
            .join("/") +
          (fileQuery.length === 0 ? "" : `?${fileQuery.join("&")}`);
        const fileSid = ctx.sid || (await ctx.client.login());
        let buffer, contentType;
        try {
          ({ buffer, contentType } = await ctx.client.fetchFile(filePath, fileSid));
        } catch (error) {
          // 404 の大半は rep 名か file 名の取り違えだが、HTTP の生の文言からはそれが読めない。
          // どちらも IDF ペイロードが正本で、rep_name は data_type とは別物。
          if (error instanceof GkillApiError && error.detail && error.detail.status === 404) {
            throw new GkillApiError(
              `File not found: rep_name=${JSON.stringify(normalized.rep_name)} ` +
                `file_name=${JSON.stringify(normalized.file_name)}. Both come from the IDF payload of ` +
                `gkill_get_kyous (payload.rep_name and payload.file_name) — rep_name is the repository, ` +
                `not the entry's data_type. List valid repository names with gkill_get_rep_infos. ` +
                `The file may also have been removed from the repository.`,
              error.detail,
            );
          }
          throw error;
        }
        const mimeType = normalizeMimeType(contentType);
        // base64はJSON-RPCレスポンスに素で載るので、青天井にすると数百MBの動画で応答が破裂する
        if (buffer.length > MAX_IDF_FILE_BYTES) {
          // file_url_full が載るのは画像の payload だけ（payload.mjs の applyFileLinks は
          // is_image のときだけサムネ file_url と原寸 file_url_full を分けて注入する）。
          // 非画像へ file_url_full を案内すると、存在しないフィールドを探させてしまう。
          const originalUrlField = mimeType.startsWith("image/") ? "file_url_full" : "file_url";
          throw new GkillApiError(
            `File is too large to return through MCP: ${buffer.length} bytes (limit ${MAX_IDF_FILE_BYTES}). ` +
              `If this is an image or a video, retry with thumb (e.g. thumb:"1024x1024", plus ` +
              `is_video:true for a video) to get a downscaled JPEG that fits. ` +
              `On stdio clients you can instead read the IDF payload's file_path directly — no size limit. ` +
              `Otherwise hand the user the payload's ${originalUrlField}, which is served from /files/ ` +
              `with no size limit.`,
            {
              file_name: normalized.file_name,
              file_size_bytes: buffer.length,
              max_bytes: MAX_IDF_FILE_BYTES,
            },
          );
        }
        return {
          // thumb のときは中身が JPEG なので、名前の拡張子もそれに合わせる。
          file_name: normalized.thumb === undefined
            ? normalized.file_name
            : thumbFileName(normalized.file_name, mimeType),
          mime_type: mimeType,
          file_size_bytes: buffer.length,
          is_image: mimeType.startsWith("image/"),
          // 縮小して取ったときだけ載せる。原寸と取り違えないための印。
          ...(normalized.thumb === undefined ? {} : { thumb: normalized.thumb }),
          file_content_base64: buffer.toString("base64"),
        };
      }
      case "gkill_get_kyou_history": {
        const normalized = normalizeKyouHistoryArgs(args);
        const target = ENTITY_TARGETS[normalized.data_type];
        // 型別エンドポイントの histories は IS_DELETED で絞らないので、
        // 削除済みの版もそのまま返る。これが「消したものを読み返す」唯一の経路
        const response = await ctx.client.callApi(
          target.getEndpoint,
          normalized.locale_name !== undefined
            ? { id: normalized.id, locale_name: normalized.locale_name }
            : { id: normalized.id },
          true, ctx.sid,
        );
        const histories = response[target.historiesKey];
        if (!Array.isArray(histories) || histories.length === 0) {
          throw new GkillApiError(entityNotFoundMessage(normalized.id, normalized.data_type));
        }
        // 各版の data_type は射影名（mi_check など。SQL の UNION の出力順で決まり、検索の mi_start とも
        // 更新応答の mi_create とも違う）で返ってくるので、この口が受理する語彙（エンティティ名）に揃える。
        // 同じ Mi が経路ごとに3通りの data_type を名乗っていた（2026-09-18 の実利用報告）。
        const versions = histories
          .slice(normalized.offset, normalized.offset + normalized.limit)
          .map((version) =>
            version !== null && typeof version === "object" ? { ...version, data_type: normalized.data_type } : version,
          );
        const hasMore = normalized.offset + versions.length < histories.length;
        return {
          id: normalized.id,
          data_type: normalized.data_type,
          latest_is_deleted: Boolean(histories[0].is_deleted),
          version_count: histories.length,
          offset: normalized.offset,
          returned_count: versions.length,
          has_more: hasMore,
          // 続きは next_offset を offset に渡す（履歴は Node が全版を持っているので offset で足りる。ADR-0626）
          ...(hasMore ? { next_offset: normalized.offset + versions.length } : {}),
          versions,
        };
      }

    default:
      throw new GkillApiError(`Unknown read tool: ${name}`);
  }
}

// buildStatusPayload は gkill_status の応答を組む。
//
// サーバ側の静的な部分（ctx.server: McpServerBase.describeServer）は必ず返し、
// gkill 由来の部分（接続先アカウントとビルド）は取れたときだけ足す。
// gkill へ届かないときも失敗にしない —— 「MCP は生きているが gkill が落ちている」を
// 区別できるのがこのツールの仕事で、失敗させると区別が消える。
// 届かない理由の本文は載せない（gkill-client のエラーは接続先 URL を含みうる。ADR-0707）。
// HTTP ステータスだけは端末固有ではないので gkill_error に載せ、全文はアクセスログへ残す。
async function buildStatusPayload(ctx) {
  const server = ctx.server ?? {};
  const startedAt = server.startedAt instanceof Date ? server.startedAt : null;
  const payload = {
    server_kind: server.kind ?? null,
    server_name: server.name ?? null,
    server_version: server.version ?? null,
    schema_revision: server.schemaRevision ?? null,
    tool_count: server.toolCount ?? null,
    transport: server.transport ?? (ctx.isLocalTransport ? "stdio" : "http"),
    started_at: startedAt ? formatLocalRfc3339(startedAt) : null,
    uptime_seconds: startedAt ? Math.max(0, Math.floor((Date.now() - startedAt.getTime()) / 1000)) : null,
    gkill_reachable: false,
  };
  try {
    // 接続先とビルドは ApplicationConfig にしか載っていない（専用 API は無い）。
    // 応答は大きいが MCP と gkill は同じ端末なので、ここで数十KB読むのは許容する。
    const response = await ctx.client.callApi("/api/get_application_config", {}, true, ctx.sid);
    const config = response.application_config || {};
    payload.gkill_reachable = true;
    payload.account = { user_id: config.user_id ?? null, device: config.device ?? null };
    payload.gkill = {
      version: config.version ?? null,
      commit_hash: config.commit_hash ?? null,
      build_time: config.build_time ?? null,
    };
  } catch (error) {
    payload.gkill_error = classifyGkillFailure(error);
    ctx.accessLog?.warn?.("status_gkill_unreachable", {
      error: error instanceof Error ? error.message : String(error),
    });
  }
  return payload;
}

// classifyGkillFailure は gkill へ届かなかった理由を、端末固有の情報を含まない語に畳む。
//
// gkill-client の Network error は detail に接続先 URL を持ち、Login failed / API error は
// gkill の応答（errors[] の error_code）を持つ。応答に載せてよいのは HTTP ステータスと
// エラーコードだけ（ADR-0707）。ログイン失敗だけは「再試行するな」を添える ——
// 資格情報の誤りは何度呼んでも直らず、gkill のログインは IP ごとに回数制限があるので、
// 確認のつもりの連打が他の経路のログインまで巻き込む（2026-09 の実測。ERR000005）。
function classifyGkillFailure(error) {
  if (!(error instanceof GkillApiError)) {
    return "unreachable";
  }
  const detail = error.detail;
  const errorCode = Array.isArray(detail?.errors) ? detail.errors[0]?.error_code : undefined;
  const codeSuffix = typeof errorCode === "string" && errorCode !== "" ? ` (${errorCode})` : "";
  if (error.message.startsWith("Login failed")) {
    return `login_failed${codeSuffix} — do not retry: the credentials this MCP server holds are rejected, and every attempt counts against gkill's per-IP login rate limit`;
  }
  if (typeof detail?.status === "number") {
    return `HTTP ${detail.status}`;
  }
  if (codeSuffix !== "") {
    return `api_error${codeSuffix}`;
  }
  return "unreachable";
}

// summarizeReadToolPayload は読み取りツールの結果要約を返す。対象外のツールは null。
// summarizeReadToolPayload は1行サマリを返す。
// applyRepInfosRowFilters は gkill_get_rep_infos の行絞り込み（rep_types / rep_names / contains / writable_only）を
// 4配列へ**その場で**適用する。fields 射影の前に呼ぶ。
//
// 列を削る fields だけでは、本番の応答は Archived Git の rep 名（plugins[] に78行）・歴代端末の GPSLogs_ / Tag_ /
// Text_（attached_data_reps[] に約120行）で埋まり、「gkill_add_tag はどこへ書くか」を知るために全部を読むことに
// なっていた（2026-09-18 の実利用報告）。rep_types の値は応答の canonical_rep_types[] と照合し、無い値は
// 0件で黙らずエラーにする（正準値の表を Node に複製しない）。contains の規則は gkill_get_all_rep_names と同じ。
function applyRepInfosRowFilters(full, normalized) {
  if (normalized.rep_types) {
    const canonical = new Set(full.canonical_rep_types);
    if (canonical.size > 0) {
      for (const repType of normalized.rep_types) {
        if (!canonical.has(repType)) {
          throw invalidArgument(
            "rep_types",
            `must be one of the canonical rep types: ${full.canonical_rep_types.join(", ")}`,
            repType,
          );
        }
      }
    }
    const wanted = new Set(normalized.rep_types);
    full.rep_infos = full.rep_infos.filter((rep) => wanted.has(rep?.rep_type));
  }
  if (normalized.rep_names) {
    const wanted = new Set(normalized.rep_names);
    const keep = (rep) => wanted.has(rep?.rep_name);
    full.rep_infos = full.rep_infos.filter(keep);
    full.plugins = full.plugins.filter(keep);
    full.attached_data_reps = full.attached_data_reps.filter(keep);
  }
  if (normalized.contains) {
    const needle = normalized.contains.toLowerCase();
    const keep = (rep) => typeof rep?.rep_name === "string" && rep.rep_name.toLowerCase().includes(needle);
    full.rep_infos = full.rep_infos.filter(keep);
    full.plugins = full.plugins.filter(keep);
    full.attached_data_reps = full.attached_data_reps.filter(keep);
  }
  if (normalized.writable_only) {
    const keep = (rep) => rep?.use_to_write === true;
    full.rep_infos = full.rep_infos.filter(keep);
    full.attached_data_reps = full.attached_data_reps.filter(keep);
    // プラグインは書き込み先にならない
    full.plugins = [];
  }
}

// 古スキーマの印の付け方は payload.mjs が正本（書き込み側と同じ文言にするため）。
export function summarizeReadToolPayload(name, payload) {
  return appendStaleSchemaNoteToSummary(summarizeReadToolPayloadBody(name, payload), payload);
}

function summarizeReadToolPayloadBody(name, payload) {
  switch (name) {
    case "gkill_status": {
      const kind = payload.server_kind ?? "unknown";
      const revision = payload.schema_revision ?? "unknown";
      const uptime = payload.uptime_seconds ?? 0;
      if (!payload.gkill_reachable) {
        return `MCP ${kind} server is up (schema_revision ${revision}, up ${uptime}s) but gkill is NOT reachable (${payload.gkill_error ?? "unreachable"}).`;
      }
      const userId = payload.account?.user_id ?? "unknown";
      const device = payload.account?.device ?? "unknown";
      return `Connected to ${userId}@${device} via ${kind} server (schema_revision ${revision}, up ${uptime}s).`;
    }
    case "gkill_get_mcp_help": {
      const length = typeof payload.text === "string" ? payload.text.length : 0;
      return `Help topic "${payload.topic ?? "index"}": ${payload.title ?? ""} (${length} chars).`;
    }
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
        // 0件のときは count_only の有無で文言が割れないようにする。
        // どちらの経路も kyous:[] なので payload からはモードを判別できない。
        return payload.total_count === 0
          ? "No entries matched."
          : `Counted ${payload.total_count} entries.`;
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
    case "gkill_get_all_tag_names": {
      const returned = Array.isArray(payload.tag_names) ? payload.tag_names.length : 0;
      if (payload.truncated) {
        return `Fetched ${returned} of ${payload.total_count} matching tag names (truncated — narrow with contains or raise limit).`;
      }
      return `Fetched ${returned} tag names.`;
    }
    case "gkill_get_all_rep_names": {
      const returned = Array.isArray(payload.rep_names) ? payload.rep_names.length : 0;
      if (payload.truncated) {
        return `Fetched ${returned} of ${payload.total_count} matching repository names (truncated — narrow with contains or raise limit).`;
      }
      return `Fetched ${returned} repository names.`;
    }
    case "gkill_get_gps_log": {
      if (Array.isArray(payload.buckets)) {
        return `Aggregated ${payload.buckets.length} daily buckets (${payload.total_count ?? 0} GPS points).`;
      }
      const returned = Array.isArray(payload.gps_logs) ? payload.gps_logs.length : 0;
      if (returned === 0 && payload.total_count !== undefined && !payload.has_more) {
        return payload.total_count === 0
          ? "No GPS points matched."
          : `Counted ${payload.total_count} GPS points.`;
      }
      if (payload.has_more && payload.next_cursor) {
        return `Returned ${returned} GPS points (${payload.remaining_count ?? 0} remaining). Next page: cursor="${payload.next_cursor}".`;
      }
      return `Returned ${returned} GPS points (0 remaining).`;
    }
    case "gkill_get_application_config":
      return `Fetched application configuration (${Object.keys(payload ?? {}).length} fields).`;
    case "gkill_get_kyou_history": {
      const total = payload.version_count ?? 0;
      const shown = payload.returned_count ?? 0;
      const deleted = payload.latest_is_deleted ? " — latest version is DELETED" : "";
      const offset = payload.offset ? ` from offset ${payload.offset}` : "";
      const more = payload.has_more ? ` (more available: pass offset:${payload.next_offset})` : "";
      return `Returned ${shown} of ${total} versions${offset}${more}${deleted}.`;
    }
    case "gkill_get_rep_infos": {
      // fields で rep_infos を外した呼び出しに「Fetched 0 repositories」と言うと、
      // 自分で外しただけなのに「リポジトリが0件」と読める（2026-08-25 の実利用レビュー）。
      const parts = [];
      if (Array.isArray(payload.rep_infos)) {
        // use_to_write を持つ行があるときだけ書ける本数を添える（古い gkill の応答には欄が無い）。
        const writable = payload.rep_infos.filter((rep) => rep?.use_to_write === true).length;
        const hasWriteFlag = payload.rep_infos.some((rep) => typeof rep?.use_to_write === "boolean");
        parts.push(
          hasWriteFlag
            ? `${payload.rep_infos.length} repositories (${writable} writable)`
            : `${payload.rep_infos.length} repositories`,
        );
      } else {
        parts.push("repositories omitted by fields");
      }
      parts.push(
        Array.isArray(payload.canonical_rep_types)
          ? `${payload.canonical_rep_types.length} canonical rep types`
          : "canonical rep types omitted by fields",
      );
      return `Fetched ${parts.join(", ")}.`;
    }
    case "gkill_get_idf_file":
      return `Retrieved file: ${payload.file_name} (${payload.file_size_bytes} bytes, ${payload.mime_type})`;
    default:
      return null;
  }
}

// ApplicationConfig の struct ツリーを持つキー。葉の識別欄はキーごとに違う。
const APP_CONFIG_STRUCT_KEYS = ["tag_struct", "mi_board_struct", "rep_struct", "rep_type_struct", "device_struct", "kftl_template_struct"];
// 葉の「識別欄」。name がこれと同じ値なら name を落とせる（compact）。contains の照合対象でもある。
const STRUCT_IDENTITY_KEYS = ["rep_name", "tag", "device", "rep_type", "board_name", "title", "template_name"];

// isStructNode はツリーのノード（配列の要素）か。
function isStructNode(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

// compactStructNode は既定値の欄を落とす（ADR-0629）。落とすのは
//   children が null / []、is_dir:false、ignore_check_rep_rykv:false、識別欄と同じ name
// だけ。check_when_inited / is_force_hide は可視判定に要るので触らない（省略された値の意味は説明文に書く）。
// 葉ノードが毎回 name==rep_name, children:null, is_dir:false, ignore_check_rep_rykv:false を持ち、
// rep_struct だけで 25,000 トークンを超えていた（2026-09-18 の実利用報告）。
function compactStructNode(node) {
  if (Array.isArray(node)) {
    return node.map((child) => compactStructNode(child));
  }
  if (!isStructNode(node)) {
    return node;
  }
  const out = {};
  for (const [key, value] of Object.entries(node)) {
    if (key === "children") {
      if (value === null || (Array.isArray(value) && value.length === 0)) continue;
      out[key] = compactStructNode(value);
      continue;
    }
    if ((key === "is_dir" || key === "ignore_check_rep_rykv") && value === false) continue;
    if (key === "name" && STRUCT_IDENTITY_KEYS.some((idKey) => node[idKey] === value)) continue;
    out[key] = value;
  }
  return out;
}

// compactAppConfigStructs は応答（fields 射影後）の struct ツリーだけを compact する。
export function compactAppConfigStructs(projected) {
  const out = { ...projected };
  for (const key of APP_CONFIG_STRUCT_KEYS) {
    if (key in out && out[key] !== null && out[key] !== undefined) {
      out[key] = compactStructNode(out[key]);
    }
  }
  return out;
}

// filterStructNodes は葉の識別欄か name に needle（大小無視の部分一致）を含む葉だけを残し、
// 葉が1つも残らない入れ物（children を持つノード）ごと落とす。
function filterStructNodes(nodes, needle) {
  if (!Array.isArray(nodes)) {
    return nodes;
  }
  const lowered = needle.toLowerCase();
  const matches = (node) =>
    [...STRUCT_IDENTITY_KEYS, "name"].some(
      (key) => typeof node[key] === "string" && node[key].toLowerCase().includes(lowered),
    );
  const out = [];
  for (const node of nodes) {
    if (!isStructNode(node)) continue;
    if (Array.isArray(node.children)) {
      const children = filterStructNodes(node.children, needle);
      if (children.length > 0) {
        out.push({ ...node, children });
      }
      continue;
    }
    if (matches(node)) {
      out.push(node);
    }
  }
  return out;
}

// filterAppConfigStructs は contains で struct ツリーを刈る。ツリー以外の欄はそのまま。
export function filterAppConfigStructs(projected, needle) {
  const out = { ...projected };
  for (const key of APP_CONFIG_STRUCT_KEYS) {
    if (key in out && Array.isArray(out[key])) {
      out[key] = filterStructNodes(out[key], needle);
    }
  }
  return out;
}

// capAppConfigSize は応答の JSON バイト数を max_size_mb に必ず収める。
// 超えていたら struct 欄を大きい順に {omitted_bytes} へ置き換え、warnings で fields / contains を案内する。
// ツリーはページングできないので、切る単位は欄1つ。get_rep_infos と違い rep_struct だけが
// 「丸ごと取るか丸ごと落とすか」の2択だった（2026-09-18 の実利用報告）。
export function capAppConfigSize(projected, maxSizeMb) {
  const maxBytes = Math.floor(maxSizeMb * 1024 * 1024);
  const size = (value) => Buffer.byteLength(JSON.stringify(value), "utf8");
  if (size(projected) <= maxBytes) {
    return projected;
  }
  const out = { ...projected };
  const warnings = [];
  const candidates = APP_CONFIG_STRUCT_KEYS.filter((key) => key in out && out[key] !== undefined)
    .map((key) => ({ key, bytes: size(out[key]) }))
    .sort((a, b) => b.bytes - a.bytes);
  for (const { key, bytes } of candidates) {
    if (size(out) <= maxBytes) break;
    out[key] = { omitted_bytes: bytes };
    warnings.push(
      `${key} (${bytes} bytes) was omitted to keep the response under max_size_mb (${maxBytes} bytes): ` +
        "narrow it with contains, request only the fields you need, or raise max_size_mb",
    );
  }
  out.warnings = [...(Array.isArray(out.warnings) ? out.warnings : []), ...warnings];
  return out;
}

// enforceKyousSizeBudget は include_plugin_content で本文を足した後の kyous[] を max_size_mb に収め直す。
//
// Go（handle_get_kyous_mcp.go）は本文の無い DTO の JSON を測って打ち切るので、Node が後から足す本文
// （最大 4000 字 × 20 件）は予算の外だった（max_size_mb:0.002・limit:2 で 3,596 バイトが警告なしで返る。
// 2026-09-18 の実利用報告）。規則は Go と同じ: 2件目以降は足す前に判定、先頭1件は超えても返して警告。
// 押し出した分は次頁へ —— カーソルは Go の encodeMCPCursor と同じ `{related_time}::{id}`
// （related_time はローカル時刻の RFC3339Nano で、Go の parseMCPCursor がそのまま受ける）。ADR-0624。
export function enforceKyousSizeBudget(payload, maxSizeMb) {
  if (!payload || !Array.isArray(payload.kyous) || payload.kyous.length === 0) {
    return payload;
  }
  const maxBytes = Math.floor(maxSizeMb * 1024 * 1024);
  const kept = [];
  let running = 0;
  let firstAlone = null;
  for (const kyou of payload.kyous) {
    const bytes = Buffer.byteLength(JSON.stringify(kyou), "utf8");
    if (kept.length > 0 && running + bytes > maxBytes) break;
    if (kept.length === 0 && bytes > maxBytes) firstAlone = bytes;
    running += bytes;
    kept.push(kyou);
  }
  const heldBack = payload.kyous.length - kept.length;
  const warnings = Array.isArray(payload.warnings) ? [...payload.warnings] : [];
  if (firstAlone !== null) {
    warnings.push(
      `single record with plugin content (${firstAlone} bytes) exceeds max_size_mb (${maxBytes} bytes); returned anyway to keep pagination progressing — lower plugin_content_max_text_length to shrink it`,
    );
  }
  if (heldBack > 0) {
    const last = kept[kept.length - 1];
    payload.kyous = kept;
    payload.returned_count = kept.length;
    payload.remaining_count = (payload.remaining_count ?? 0) + heldBack;
    payload.has_more = true;
    payload.next_cursor = `${last.related_time}::${last.id}`;
    warnings.push(
      `plugin content pushed the page over max_size_mb (${maxBytes} bytes): ${heldBack} of ${kept.length + heldBack} entries were held back and will be returned on the next cursor page (next_cursor is set)`,
    );
    if (payload.plugin_content && typeof payload.plugin_content === "object") {
      payload.plugin_content = recountInlinePluginContent(kept, payload.plugin_content);
    }
  }
  if (warnings.length > 0) {
    payload.warnings = warnings;
  }
  return payload;
}

// recountInlinePluginContent は残した kyous[] から plugin_content の件数を数え直す。
function recountInlinePluginContent(kyous, previous) {
  const stats = { ...previous, requested: 0, inlined: 0, truncated: 0, skipped: 0, errors: 0, total_text_length: 0 };
  for (const kyou of kyous) {
    const payload = kyou?.payload;
    if (!payload || payload.kind !== "plugin") continue;
    stats.requested += 1;
    switch (payload.content_status) {
      case "ok":
        stats.inlined += 1;
        break;
      case "truncated":
        stats.inlined += 1;
        stats.truncated += 1;
        break;
      case "skipped":
        stats.skipped += 1;
        break;
      case "error":
        stats.errors += 1;
        break;
      default:
        break;
    }
    stats.total_text_length +=
      (typeof payload.content_text === "string" ? payload.content_text.length : 0) +
      (typeof payload.content_html === "string" ? payload.content_html.length : 0);
  }
  return stats;
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

/**
 * paginateRepNames は rep名の全一覧に contains / limit を適用する。
 *
 * total_count は**絞り込み後・limit適用前**の件数。limit の前の件数を返さないと
 * 「contains に一致したのが何件か」が読めず、truncated の意味も決まらない。
 *
 * @param {string[]} repNames gkill が返した全rep名。
 * @param {{contains?: string, limit: number}} options 正規化済みの絞り込み条件。
 * @returns {{rep_names: string[], total_count: number, returned_count: number, truncated: boolean}} 応答。
 */
// thumbFileName は縮小版の名前。thumb を頼むとサーバは JPEG を返すので、
// 元の名前(.webp や .png)をそのまま返すと「拡張子と中身が食い違うファイル」を
// 案内することになり、名前で判断して保存する側が壊れる。
function thumbFileName(fileName, mimeType) {
  if (mimeType !== "image/jpeg") {
    return fileName;
  }
  return String(fileName).replace(/\.[^./\\]*$/, "") + ".jpg";
}

// paginateNameList は名前一覧に contains / limit を適用する共通部分。
// rep 名とタグ名で規則を分けない（片方だけ絞り込めると、もう片方は
// 「autolog 系のタグはあるか」を確かめるだけで全件を受け取ることになる）。
function paginateNameList(names, options) {
  let matched = names;
  if (options.contains !== undefined && options.contains !== "") {
    const needle = options.contains.toLowerCase();
    matched = names.filter((name) => String(name).toLowerCase().includes(needle));
  }
  const page = matched.slice(0, options.limit);
  return { page, total: matched.length };
}

export function paginateRepNames(repNames, options) {
  const { page, total } = paginateNameList(repNames, options);
  return {
    rep_names: page,
    total_count: total,
    returned_count: page.length,
    truncated: page.length < total,
  };
}

// paginateTagNames は paginateRepNames と同じ規則をタグ名へ当てる。
export function paginateTagNames(tagNames, options) {
  const { page, total } = paginateNameList(tagNames, options);
  return {
    tag_names: page,
    total_count: total,
    returned_count: page.length,
    truncated: page.length < total,
  };
}

// paginateGpsLogs は取得済みの全点列に limit/cursor/count_only/group_by を適用する。
export function paginateGpsLogs(gpsLogs, options) {
  // 規則の正本は normalization.mjs。get_kyous と同じ文言で弾く。
  assertAggregationNotCombinedWithCursor(options);
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
