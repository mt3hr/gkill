// レスポンスのペイロード加工。3つのMCPサーバで完全に同じものを使う。
//
// 以前は gkill-read-server.mjs / gkill-write-server.mjs / gkill-readwrite-server.mjs へ
// 逐語コピーされていた（read の94%、write の89%が readwrite と重複していた）。
// 片方だけ直すと静かにずれるので、ここに1つだけ置く。

// リモート向け file_url の既定サムネサイズ (gkillの ?thumb=WxH に渡す。長辺上限1024)。
export const DEFAULT_FILE_LINK_THUMB = "1024x1024";

// MINT_FILE_LINKS は「この応答には公開ファイルURLを発行してよい」の印。
// gkill_get_kyous のハンドラが include_file_urls:true のときだけ payload に立て、
// buildToolResult（mcp-server-base.mjs）が HTTP のときに見る。Symbol なので JSON には出ず、
// structuredContent にも混ざらない。以前は HTTP なら全応答で idf ごとに2本鋳造していた（ADR-0630）。
export const MINT_FILE_LINKS = Symbol("gkill.mint_file_links");
// 配信ルートが受け付けるサムネ指定の検証用。
export const THUMB_QUERY_REGEX = /^\d{1,4}x\d{1,4}$/;
// 一辺の上限。Go 側 thumbFileServer.maxSize の写し
// (dao/reps/idf_thumb_file_server.go)。これを超えると Go はサムネを作らず
// **黙って原本を返す**ので、送る前に弾く必要がある。
export const MAX_THUMB_SIZE = 1024;

// Content-Type ヘッダから "; charset=..." などのパラメータを落とし、MIME型だけにする。
export function normalizeMimeType(contentType) {
  return String(contentType || "").split(";")[0].trim();
}

// file_path はこのマシン上の絶対パス。同一マシンで動くクライアント (stdio) にしか意味がなく、
// リモートクライアントに渡すとユーザのディレクトリ構造を漏らすことになるので取り除く。
export function stripFilePaths(value) {
  if (Array.isArray(value)) {
    for (const item of value) stripFilePaths(item);
    return value;
  }
  if (value !== null && typeof value === "object") {
    delete value.file_path;
    for (const key of Object.keys(value)) stripFilePaths(value[key]);
  }
  return value;
}

// idfペイロード (rep_name + file_name を持つ) を判定する。
export function isIdfPayload(value) {
  return (
    value !== null &&
    typeof value === "object" &&
    typeof value.rep_name === "string" &&
    typeof value.file_name === "string"
  );
}

// リモートクライアント向けに、idfペイロードへ期限付きの公開ファイルURLを注入する。
// 実パス (file_path) は同時に取り除く。ローカルクライアント (stdio) では呼ばない。
// ctx = { publicBaseUrl, store }, gkillSessionId は発行元のOAuthセッション。
export function applyFileLinks(value, ctx, gkillSessionId) {
  if (Array.isArray(value)) {
    for (const item of value) applyFileLinks(item, ctx, gkillSessionId);
    return value;
  }
  if (value === null || typeof value !== "object") {
    return value;
  }
  if (isIdfPayload(value)) {
    delete value.file_path;
    const { token, expiresAt } = ctx.store.mintLink({
      gkillSessionId,
      repName: value.rep_name,
      fileName: value.file_name,
      isImage: Boolean(value.is_image),
    });
    const base = `${ctx.publicBaseUrl}/files/${token}`;
    if (value.is_image) {
      // 既定は軽量なサムネ、原寸は file_url_full で別途取得できる
      value.file_url = `${base}?thumb=${DEFAULT_FILE_LINK_THUMB}`;
      value.file_url_full = base;
    } else {
      value.file_url = base;
    }
    // 期限を添える。無いと人間へ渡したリンクがいつ切れるか誰にも分からない（2026-09-18 の実利用報告）。
    value.file_url_expires_at = new Date(expiresAt).toISOString();
    return value;
  }
  for (const key of Object.keys(value)) applyFileLinks(value[key], ctx, gkillSessionId);
  return value;
}

export function summarizeToolError(name, error, detail) {
  const prefix = name ? `${name} failed` : "Tool call failed";
  if (detail && detail.field) {
    return `${prefix}: ${error} (field: ${detail.field})`;
  }
  return `${prefix}: ${error}`;
}

// entityNotFoundMessage は「1件を型別に引いたが見つからない」ときの文言。
//
// **read / write の両方から使う。** 以前は3種類に割れていて
// （read の `Entity not found: {id}`、write の親切版、update 9本の `Kmemo not found: {id}`）、
// 同じ状況で受け取る説明が呼んだツールによって違った（2026-08-25 の実利用レビュー）。
//
// 取得は型別エンドポイントなので、ID が無いのか型を取り違えたのかは
// サーバの応答からは区別できない。**区別できないことを言う**のが唯一正しい案内で、
// 「ID が存在しない」と断定してはいけない。
// 実際 data_type:"urlog" で kmemo の id を引くと、この経路へ来る。
// appendStaleSchemaNoteToSummary は、本文の warnings に古スキーマの指摘があるとき
// 1行サマリにも印を付ける。本文の warnings を読まない経路でも気づけるようにするため。
//
// 以前は読み取りの要約器だけが持っており、書き込みの要約器には無かった。
// gkill_delete_kyou / gkill_restore_kyou の targets（後から足した非string型の引数）が
// まさに古スキーマで壊れる側なので、片側だけだと「同じ古さなのに読み取りでしか
// 知らされない」ことになる（ADR-0609 / ADR-0611）。
export function appendStaleSchemaNoteToSummary(summary, payload) {
  if (summary === null || summary === undefined) {
    return summary;
  }
  if (payload === null || typeof payload !== "object" || !Array.isArray(payload.warnings)) {
    return summary;
  }
  if (!payload.warnings.some((warning) => String(warning).includes("tool schema snapshot looks stale"))) {
    return summary;
  }
  return `${summary} (this client's tool schema looks stale — reconnect the MCP client)`;
}

// 1行サマリに載せる warning は最初の1件だけ。長文でも要点が残るよう緩めに切る
// (全文は本文の warnings[] にある)。
const SUMMARY_WARNING_MAX_LENGTH = 200;

// appendWarningsToSummary は本文の warnings / partial を1行サマリへ昇格させる。
// 以前は要約が件数と cursor だけで、未知タグで0件でも `No entries matched.` としか
// 出なかった。要約だけを見る利用者・モデルが「本当に0件」と誤読する
// (2026-08-30 レビュー P1)。
//
// - partial は付随データ欠落専用の印 (M-05) のままで、意味は変えない。ここは表示だけ。
// - warning があっても partial:false になりうる (壊れた rep の全期間 count 等) ので、
//   partial では warnings の代用にならない。両方を独立に見る。
// - 古スキーマ警告は appendStaleSchemaNoteToSummary が専用文言で扱うのでここでは飛ばす。
// - 新しい部分成功フラグは足さない (ADR-0216)。
export function appendWarningsToSummary(summary, payload) {
  if (summary === null || summary === undefined) {
    return summary;
  }
  if (payload === null || typeof payload !== "object") {
    return summary;
  }
  const parts = [];
  if (payload.partial) {
    parts.push("PARTIAL: attached data may be missing for some entries.");
  }
  const warnings = Array.isArray(payload.warnings)
    ? payload.warnings.filter((warning) => !String(warning).includes("tool schema snapshot looks stale"))
    : [];
  if (warnings.length > 0) {
    let first = String(warnings[0]);
    if (first.length > SUMMARY_WARNING_MAX_LENGTH) {
      first = `${first.slice(0, SUMMARY_WARNING_MAX_LENGTH)}…`;
    }
    const more = warnings.length > 1 ? ` (+${warnings.length - 1} more)` : "";
    parts.push(`WARNING: ${first}${more}`);
  }
  if (parts.length === 0) {
    return summary;
  }
  return `${summary} ${parts.join(" ")}`;
}

export function entityNotFoundMessage(id, dataType) {
  return (
    `Entity not found: ${id} (looked it up as data_type ${JSON.stringify(dataType)}; ` +
    `the lookup is per-type, so a wrong data_type looks exactly like a wrong id. ` +
    // gkill_get_kyous を名指ししない: 書き込み専用サーバには載っていないので、
    // そこで出すと「案内されたツールが無い」になる（read 側は持っている）。
    `Confirm the entry's data_type, and check you are on the account that holds it ` +
    `(gkill_get_application_config with fields:["user_id"]), then retry.)`
  );
}
