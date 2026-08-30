// 書き込みツールのディスパッチと要約。write / readwrite の2サーバが共有する。
//
// 以前は write と readwrite に逐語コピーされており、add/update/delete の20ケース
// 約500行がバイト単位で同一だった。読み取り側は同じ形で
// IDFファイルパス取得の1ツールだけが片側だけ古いまま静かに壊れた前例があるので
// (lib/read-handlers.mjs の冒頭を参照)、書き込み側も同じ形で1箇所へ寄せる。
// 実装はこの1箇所が正本で、サーバ側は isWriteToolName / handleWriteToolCall /
// summarizeWriteToolPayload へ委譲するだけにする。
//
// ツール定義の配列 (TOOLS) は各サーバファイルに残すこと。
// src/tools/verify_docs.mjs がサーバファイル中の `const TOOLS = [...]` リテラルを
// 走査してツール数を数えており、ここへ移すとツール数が黙って0になる。

import crypto from "node:crypto";

import { GkillApiError, invalidArgument } from "./errors.mjs";
import { WRITE_TOOLS } from "./write-tools.mjs";
import { entityNotFoundMessage, appendStaleSchemaNoteToSummary } from "./payload.mjs";
import { appendStaleSchemaWarning } from "./normalization.mjs";
import { ENTITY_TARGETS, unknownToolMessage } from "./constants.mjs";
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
  normalizeRestoreArgs,
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

// assertBoardExists は board_name が実在の板名か照合し、無ければ呼び出し側エラーで弾く。
// gkill 側 (AddMi / UpdateMi) は板の実在を確かめず、未知の板名は新しい板の作成になる。
// それが既定の仕様だが、AI の typo がそのまま新しい板になると気付きにくいので、
// allow_create_board:false のときだけこの照合を通す (2026-08-30 MCPレビュー、フラグ追加)。
// 照合は完全一致 —— 板名の大小・空白ゆらぎを吸収すると「似た名前の別の板」へ落ちて
// しまい、typo 検出という目的と矛盾する。
async function assertBoardExists(ctx, boardName, localeName) {
  const response = await ctx.client.callApi(
    "/api/get_mi_board_list",
    localeName !== undefined ? { locale_name: localeName } : {},
    true, ctx.sid,
  );
  const boards = Array.isArray(response.boards) ? response.boards : [];
  if (!boards.includes(boardName)) {
    throw invalidArgument(
      "board_name",
      `unknown board ${JSON.stringify(boardName)} — gkill would create a new board with this name. ` +
        `Pass allow_create_board:true to allow that, or pick an existing board (gkill_get_mi_board_list)`,
      boardName,
    );
  }
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

// mergeStored は「サーバが保存した版」を手元の値に重ねる。
//
// 以前は書き込み応答が、ローカルで組んだ current をそのまま返していた。すると
// update_time が JS の UTC・ミリ秒つき (…T18:47:31.035Z) のままになり、同じ応答の
// updated_kyou (…T03:47:31+09:00) と**日付表記まで食い違う**。保存は1秒解像度なので、
// そのミリ秒はそもそも存在しない精度でもあった。
// 置き換えではなく重ねるのは、応答が部分的でも手元のフィールドを落とさないため。
function mergeStored(current, stored) {
  return stored ? { ...current, ...stored } : current;
}

// nextUpdateTime は「現在値より必ず後」になる更新時刻を返す。
//
// **UPDATE_TIME は1秒解像度で保存される** (sqlite3impl.TimeLayout)。
// 履歴の取得は ID + UpdateTime で dedup し、検索の最新版判定は
// 厳密な UpdateTime.After で行うので、同じ秒の中で delete → restore すると
// 新しい版が最新と見なされず、黙って何も起きない（あるいは版が入れ替わる）。
// 実時刻が現在値と同じ秒に落ちるときだけ1秒進める。
function nextUpdateTime(current) {
  const now = Date.now();
  const previous = Date.parse(current?.update_time ?? "");
  if (!Number.isFinite(previous)) {
    return new Date(now).toISOString();
  }
  // 保存されるのは秒までなので、秒に丸めてから比較する
  const previousSecond = Math.floor(previous / 1000) * 1000;
  const nowSecond = Math.floor(now / 1000) * 1000;
  return new Date(nowSecond > previousSecond ? now : previousSecond + 1000).toISOString();
}

// UPDATE_TARGETS は gkill_update_* 9ツールの「型ごとに違うところ」だけを持つ表。
//
// 取得→patch→更新の手順そのものは9本とも同じで、以前は24行のブロックが9本並んでいた。
// 取得先・更新先・応答キーの対応は constants.mjs の ENTITY_TARGETS に既にあり、
// softDeleteOne と gkill_get_kyou_history はそちらを使っている。
// **同じ対応表が「表」と「9箇所の直書き」の2形態で存在していた**ので、表側へ寄せた。
// 直書きだったころは「見つからない」のメッセージも3種類に割れていた。
const UPDATE_TARGETS = {
  kmemo: { normalize: normalizeUpdateKmemoArgs, patchFields: ["content", "related_time"] },
  urlog: { normalize: normalizeUpdateUrlogArgs, patchFields: ["url", "title", "related_time"] },
  nlog: { normalize: normalizeUpdateNlogArgs, patchFields: ["title", "amount", "shop", "related_time"] },
  lantana: { normalize: normalizeUpdateLantanaArgs, patchFields: ["mood", "related_time"] },
  timeis: { normalize: normalizeUpdateTimeIsArgs, patchFields: ["title", "start_time", "end_time"] },
  mi: {
    normalize: normalizeUpdateMiArgs,
    patchFields: ["title", "board_name", "is_checked", "limit_time", "estimate_start_time", "estimate_end_time"],
    // allow_create_board:false のときだけ、移動先の板名を実在の板と照合する
    // (add と同じ typo ガード。patchFields ではないので実体には書かれない)。
    preUpdate: async (ctx, normalized) => {
      if (normalized.allow_create_board === false && normalized.board_name !== undefined) {
        await assertBoardExists(ctx, normalized.board_name, normalized.locale_name);
      }
    },
  },
  kc: { normalize: normalizeUpdateKcArgs, patchFields: ["title", "num_value", "related_time"] },
  tag: { normalize: normalizeUpdateTagArgs, patchFields: ["tag"] },
  text: { normalize: normalizeUpdateTextArgs, patchFields: ["text"] },
};

/**
 * runUpdate は gkill_update_* 1件を処理する。
 *
 * gkill に部分更新のAPIは無いので、現在値を取って渡された欄だけ上書きし、
 * 同じ型の更新APIへ送り直す（patch semantics）。渡さなかった欄は保持される。
 *
 * @param {object} ctx ハンドラ文脈。
 * @param {string} dataType エンティティ種別（ENTITY_TARGETS のキー）。
 * @param {unknown} args ツール引数。
 * @returns {Promise<object>} updated_xxx と updated_kyou。
 */
async function runUpdate(ctx, dataType, args) {
  const spec = UPDATE_TARGETS[dataType];
  const target = ENTITY_TARGETS[dataType];
  if (!spec || !target) {
    throw new GkillApiError(`Unsupported data_type for update: ${dataType}`);
  }
  const normalized = spec.normalize(args);
  // 型固有の事前検証 (現状は mi の板名照合だけ)。取得より前に置いて fail-fast にする。
  if (spec.preUpdate) {
    await spec.preUpdate(ctx, normalized);
  }
  const getResponse = await ctx.client.callApi(target.getEndpoint, { id: normalized.id }, true, ctx.sid);
  const histories = getResponse[target.historiesKey];
  if (!Array.isArray(histories) || histories.length === 0) {
    // 文言は削除・復活と同じものを使う。以前は型ごとに "Kmemo not found" のような
    // 別文言で、型の取り違えとID不在が見分けられなかった。
    throw new GkillApiError(entityNotFoundMessage(normalized.id, dataType));
  }
  const current = histories[0];
  // 未指定 = 触らない。null は「消す」の意味を持つ欄があるので !== undefined で見る
  // （TimeIs の end_time が唯一の例。3値パッチ）。
  const patchedFields = spec.patchFields.filter((field) => normalized[field] !== undefined);
  // 更新する欄が1つも無いなら、内容の同じ版を積むだけになるので弾く。
  // 追記型なので no-op でも履歴は1つ増え、あとから読む側には
  // 「何が変わったのか」が区別できない（delete/restore の二重操作ガードと同じ理由）。
  if (patchedFields.length === 0) {
    throw new GkillApiError(
      `No fields to update for ${dataType} ${normalized.id} (nothing changed; ` +
        `pass at least one of ${spec.patchFields.join(", ")}).`,
    );
  }
  for (const field of patchedFields) {
    current[field] = normalized[field];
  }
  // softDeleteOne と同じく nextUpdateTime を通す。UPDATE_TIME は1秒解像度で、
  // 最新版の判定は厳密な After なので、同じ秒の中で2回更新すると
  // 2回目が最新と見なされず黙って消える。
  current.update_time = nextUpdateTime(current);
  current.update_app = ctx.appName;
  current.update_device = WRITE_DEVICE;
  current.update_user = ctx.userId;
  const response = await ctx.client.callApi(
    target.updateEndpoint,
    { [target.requestKey]: current, want_response_kyou: true, locale_name: normalized.locale_name },
    true, ctx.sid,
  );
  return {
    [target.responseKey]: mergeStored(current, response[target.responseKey]),
    updated_kyou: response.updated_kyou || null,
  };
}

/**
 * softDeleteOne は1件の is_deleted を切り替える。削除と復活で共通。
 *
 * gkill に専用の削除APIは無く、現在値を取って is_deleted を立て、同じ型の更新APIへ
 * 送り直す patch 方式なので、1件につき2往復かかる。
 *
 * @param {object} ctx ハンドラ文脈。
 * @param {{id: string, data_type: string}} entry 対象。
 * @param {boolean} deleting true=削除、false=復活。
 * @param {string|undefined} localeName サーバメッセージのロケール。
 * @returns {Promise<object>} 単件形式の応答。
 */
async function softDeleteOne(ctx, entry, deleting, localeName) {
  const target = ENTITY_TARGETS[entry.data_type];
  if (!target) {
    throw new GkillApiError(`Unsupported data_type for ${deleting ? "delete" : "restore"}: ${entry.data_type}`);
  }
  // 1. 現在値を取る（データ欄を落とさずに patch するため）
  const getResponse = await ctx.client.callApi(target.getEndpoint, { id: entry.id }, true, ctx.sid);
  const histories = getResponse[target.historiesKey];
  if (!Array.isArray(histories) || histories.length === 0) {
    throw new GkillApiError(entityNotFoundMessage(entry.id, entry.data_type));
  }
  const current = histories[0];
  // 無意味な版を積まない。「消えたのか、元から無かったのか、既に消えていたのか」を
  // 呼び出し側が区別できるようにする。
  if (deleting && current.is_deleted) {
    throw new GkillApiError(
      `Entity is already deleted: ${entry.id} (nothing changed; read it with gkill_get_kyou_history or undo with gkill_restore_kyou)`,
    );
  }
  if (!deleting && !current.is_deleted) {
    throw new GkillApiError(`Entity is already active (not deleted): ${entry.id}`);
  }
  // 2. is_deleted と更新メタデータを差し替える
  current.is_deleted = deleting;
  current.update_time = nextUpdateTime(current);
  current.update_app = ctx.appName;
  current.update_device = WRITE_DEVICE;
  current.update_user = ctx.userId;
  // 3. 更新APIへ送る
  const response = await ctx.client.callApi(
    target.updateEndpoint,
    { [target.requestKey]: current, want_response_kyou: true, locale_name: localeName },
    true, ctx.sid,
  );
  const result = {};
  // 復活はキー名だけ restored_ に付け替える（サーバ側の応答キーは updated_ のまま）
  const responseKey = deleting ? target.responseKey : `restored_${entry.data_type}`;
  result[responseKey] = mergeStored(current, response[target.responseKey]);
  if (response.updated_kyou) result.updated_kyou = response.updated_kyou;
  return result;
}

/**
 * runSoftDeleteTargets は単件と一括の両方を捌く。
 *
 * 一括は**直列**に回す。1件が取得+更新の2往復なので、並列にすると
 * 同じ書き込み口へ一斉に投げることになる。
 * 途中で失敗しても止めない —— DBトランザクションではないので、
 * 「どこまで消したか」を返さないと利用者は後始末ができない（KFTL の created[] と同じ考え方）。
 *
 * @param {object} ctx ハンドラ文脈。
 * @param {{targets: Array<{id: string, data_type: string}>, batch: boolean, locale_name?: string}} normalized 正規化済み引数。
 * @param {boolean} deleting true=削除、false=復活。
 * @returns {Promise<object>} 単件なら従来どおりの形、一括なら results[] を持つ形。
 */
async function runSoftDeleteTargets(ctx, normalized, deleting) {
  if (!normalized.batch) {
    return softDeleteOne(ctx, normalized.targets[0], deleting, normalized.locale_name);
  }
  const results = [];
  let succeeded = 0;
  for (const entry of normalized.targets) {
    try {
      await softDeleteOne(ctx, entry, deleting, normalized.locale_name);
      results.push({ id: entry.id, data_type: entry.data_type, ok: true });
      succeeded++;
    } catch (error) {
      results.push({
        id: entry.id,
        data_type: entry.data_type,
        ok: false,
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }
  return {
    results,
    succeeded_count: succeeded,
    failed_count: results.length - succeeded,
  };
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
// ディスパッチ本体を包んで、古いツールスキーマを掴んだクライアントへの警告を
// **1箇所で**足す。読み取り側（handleReadToolCall）と同じ形。
// 書き込み側にも非string型の後付け引数（delete / restore の targets）があるので、
// 片側だけに掛けると「同じ古さなのに読み取りでしか知らされない」ことになる。
export async function handleWriteToolCall(ctx, name, args) {
  const payload = await dispatchWriteToolCall(ctx, name, args);
  return appendStaleSchemaWarning(payload, name, args);
}

async function dispatchWriteToolCall(ctx, name, args) {
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
          {
            urlog,
            want_response_kyou: true,
            locale_name: normalized.locale_name,
            // fetch_metadata / fetch_favicon (既定 true) を Go 側の抑止フラグへ反転して写す。
            // 両方 false ならサーバは対象サイトにも favicon サービスにも外向き通信しない。
            skip_fetch_metadata: normalized.fetch_metadata === false,
            skip_fetch_favicon: normalized.fetch_favicon === false,
          },
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
        // allow_create_board:false のときだけ、指定された板名を実在の板と照合する。
        // 既定板への補完値は照合しない (板が1つも無い新規アカウントで既定板すら弾いてしまう)。
        if (normalized.allow_create_board === false && normalized.board_name !== undefined) {
          await assertBoardExists(ctx, normalized.board_name, normalized.locale_name);
        }
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
          {
            kftl_text: normalized.kftl_text,
            locale_name: normalized.locale_name,
            idempotency_key: normalized.idempotency_key,
          },
          true, ctx.sid,
        );
        // created は「実際に書かれたもの」。KFTL は1つのテキストから複数の Kyou を作るので、
        // これが無いと呼び出し側は何が作られたか分からない。
        // 失敗時も途中まで書けたぶんが入る（KFTL は DB トランザクションを使わない）。
        return { messages: response.messages || [], created: response.created || [] };
      }

      case "gkill_delete_kyou": {
        const normalized = normalizeDeleteArgs(args);
        return runSoftDeleteTargets(ctx, normalized, true);
      }

      case "gkill_restore_kyou": {
        const normalized = normalizeRestoreArgs(args);
        return runSoftDeleteTargets(ctx, normalized, false);
      }

      // ----- Update tools -----
      case "gkill_update_kmemo":
      case "gkill_update_urlog":
      case "gkill_update_nlog":
      case "gkill_update_lantana":
      case "gkill_update_timeis":
      case "gkill_update_mi":
      case "gkill_update_kc":
      case "gkill_update_tag":
      case "gkill_update_text": {
        const dataType = name.slice("gkill_update_".length);
        return runUpdate(ctx, dataType, args);
      }
      default:
        throw new GkillApiError(unknownToolMessage(name));
  }
}

// batchSoftDeleteSummary は一括削除・一括復活の1行サマリを作る。
//
// 失敗件数を出さないと `failed_count:1` でも「completed」と読めてしまう。
// 一括は DB トランザクションではないので、**どこまで済んだか**がサマリの本題
// （2026-08-25 の実利用レビュー）。
function batchSoftDeleteSummary(verb, payload) {
  const succeeded = payload.succeeded_count ?? 0;
  const failed = payload.failed_count ?? 0;
  const total = succeeded + failed;
  if (failed === 0) {
    return `${verb}: ${succeeded}/${total} entries.`;
  }
  return `${verb}: ${succeeded}/${total} entries — ${failed} FAILED (see results[] for the reason of each).`;
}

// gkill_add_* / gkill_update_* の1行要約は、型ごとに違うのが「動詞」と「応答キー」だけ。
// case を18本並べると、欄を1つ足すとき18箇所を触ることになり、1つ落としても
// テストは緑のまま（各ツールのテストは自分の case しか見ない）。表から作る（ADR-0611）。
//
// 動詞が "Added" なのは tag / text だけ。付随データは「作る」のではなく既存の記録へ「付ける」。
const ADD_SUMMARY_VERBS = { tag: "Added", text: "Added" };

const ENTITY_SUMMARIZERS = new Map();
for (const dataType of Object.keys(UPDATE_TARGETS)) {
  const addVerb = ADD_SUMMARY_VERBS[dataType] || "Created";
  ENTITY_SUMMARIZERS.set(
    `gkill_add_${dataType}`,
    (payload) => `${addVerb} ${dataType}: ${payload[`added_${dataType}`]?.id || "unknown"}`,
  );
  ENTITY_SUMMARIZERS.set(
    `gkill_update_${dataType}`,
    (payload) => `Updated ${dataType}: ${payload[`updated_${dataType}`]?.id || "unknown"}`,
  );
}

// summarizeWriteToolPayload は書き込みツールの結果要約を返す。対象外のツールは null。
export function summarizeWriteToolPayload(name, payload) {
  const entitySummarizer = ENTITY_SUMMARIZERS.get(name);
  if (entitySummarizer) {
    return appendStaleSchemaNoteToSummary(entitySummarizer(payload), payload);
  }
  return appendStaleSchemaNoteToSummary(summarizeWriteToolPayloadBody(name, payload), payload);
}

function summarizeWriteToolPayloadBody(name, payload) {
  switch (name) {
    case "gkill_submit_kftl": {
      const created = Array.isArray(payload.created) ? payload.created : [];
      if (created.length === 0) {
        return "KFTL submitted: nothing was written (blank lines and idempotent replays write nothing).";
      }
      const kinds = {};
      for (const record of created) {
        const kind = record.updated ? `${record.data_type} (updated)` : record.data_type;
        kinds[kind] = (kinds[kind] || 0) + 1;
      }
      const breakdown = Object.entries(kinds)
        .map(([kind, count]) => (count === 1 ? kind : `${kind} x${count}`))
        .join(", ");
      return `KFTL submitted: wrote ${created.length} record(s) — ${breakdown}.`;
    }
    case "gkill_restore_kyou": {
      if (Array.isArray(payload.results)) {
        return batchSoftDeleteSummary("Restored", payload);
      }
      const keys = Object.keys(payload).filter((k) => k.startsWith("restored_"));
      return `Restored: ${keys.length > 0 ? keys.join(", ") : "completed"}`;
    }
    case "gkill_delete_kyou": {
      if (Array.isArray(payload.results)) {
        return batchSoftDeleteSummary("Deleted (soft)", payload);
      }
      const keys = Object.keys(payload).filter((k) => k.startsWith("updated_"));
      return `Deleted (soft): ${keys.length > 0 ? keys.join(", ") : "completed"}`;
    }
    default:
      return null;
  }
}
