// レスポンスのペイロード加工。3つのMCPサーバで完全に同じものを使う。
//
// 以前は gkill-read-server.mjs / gkill-write-server.mjs / gkill-readwrite-server.mjs へ
// 逐語コピーされていた（read の94%、write の89%が readwrite と重複していた）。
// 片方だけ直すと静かにずれるので、ここに1つだけ置く。

// リモート向け file_url の既定サムネサイズ (gkillの ?thumb=WxH に渡す。長辺上限1024)。
export const DEFAULT_FILE_LINK_THUMB = "1024x1024";
// 配信ルートが受け付けるサムネ指定の検証用。
export const THUMB_QUERY_REGEX = /^\d{1,4}x\d{1,4}$/;

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
    const token = ctx.store.mint({
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
