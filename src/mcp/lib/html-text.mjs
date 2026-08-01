// プラグインが返すコンテンツHTMLをAIが読めるプレーンテキストに落とすための変換。
//
// プラグインのコンテンツHTMLは <style> と <script> を含む完結したHTML文書で、
// バイト数の大半が表示用のボイラープレートになる。生HTMLをそのままMCPで返すと
// トークンを浪費するだけなので、既定ではここでテキストに変換して返す。
// 外部依存を増やしたくないので、正規表現ベースの軽量な変換にとどめる。

const BLOCK_TAGS = [
  "address",
  "article",
  "aside",
  "blockquote",
  "details",
  "dd",
  "div",
  "dl",
  "dt",
  "fieldset",
  "figcaption",
  "figure",
  "footer",
  "form",
  "h1",
  "h2",
  "h3",
  "h4",
  "h5",
  "h6",
  "header",
  "hr",
  "li",
  "main",
  "nav",
  "ol",
  "p",
  "pre",
  "section",
  "summary",
  "table",
  "tbody",
  "td",
  "tfoot",
  "th",
  "thead",
  "title",
  "tr",
  "ul",
];

// BOUNDARY はタグ由来の改行位置を表す内部マーカー。入力に含まれていたものは先に消す。
const BOUNDARY = String.fromCharCode(0);
const BOUNDARY_REGEX = new RegExp(BOUNDARY, "g");

const NAMED_ENTITIES = new Map([
  ["nbsp", " "],
  ["lt", "<"],
  ["gt", ">"],
  ["quot", '"'],
  ["apos", "'"],
  ["#39", "'"],
  ["yen", "¥"],
  ["copy", "©"],
  ["reg", "®"],
  ["hellip", "…"],
  ["mdash", "—"],
  ["ndash", "–"],
]);

// decodeHtmlEntities はHTMLエンティティを実文字に戻す。
// &amp; を最後に処理しないと "&amp;lt;" が "<" になってしまうので、順序を守る。
export function decodeHtmlEntities(text) {
  return String(text)
    .replace(/&#x([0-9a-fA-F]+);/g, (_match, hex) => safeFromCodePoint(Number.parseInt(hex, 16)))
    .replace(/&#(\d+);/g, (_match, dec) => safeFromCodePoint(Number.parseInt(dec, 10)))
    .replace(/&([a-zA-Z]+);/g, (match, name) => {
      const replacement = NAMED_ENTITIES.get(name.toLowerCase());
      return replacement === undefined ? match : replacement;
    })
    .replace(/&amp;/g, "&");
}

function safeFromCodePoint(codePoint) {
  if (!Number.isFinite(codePoint) || codePoint < 0 || codePoint > 0x10ffff) {
    return "";
  }
  try {
    return String.fromCodePoint(codePoint);
  } catch {
    return "";
  }
}

/**
 * htmlToText はHTMLをプレーンテキストに変換する。
 *
 * @param {string} html 変換対象のHTML。
 * @param {{maxLength?: number|null}} [options] maxLength を超えた分は切り落として末尾に印を付ける。
 * @returns {{text: string, truncated: boolean}} 変換結果と、切り詰めが起きたか。
 */
export function htmlToText(html, { maxLength = null } = {}) {
  if (typeof html !== "string" || html === "") {
    return { text: "", truncated: false };
  }

  let work = html;
  // コメント・スクリプト・スタイルは中身ごと落とす
  work = work.replace(/<!--[\s\S]*?-->/g, "");
  work = work.replace(/<script\b[^>]*>[\s\S]*?<\/script\s*>/gi, "");
  work = work.replace(/<style\b[^>]*>[\s\S]*?<\/style\s*>/gi, "");
  // タグ由来の改行は境界マーカーで表す。素の "\n" を入れてしまうと
  // </div><div> のような隣接だけで空行ができ、入れ子の深いHTMLが空行だらけになる。
  // マーカーの連続は後段でまとめて1つの改行に落とす。
  work = work.replace(BOUNDARY_REGEX, "");
  work = work.replace(/<br\s*\/?>/gi, BOUNDARY);
  for (const tag of BLOCK_TAGS) {
    work = work.replace(new RegExp(`</?${tag}\\b[^>]*>`, "gi"), BOUNDARY);
  }
  // 残ったタグ (span, strong, a など) は取り除くだけ
  work = work.replace(/<[^>]*>/g, "");

  // マーカーで区切り、区切りごとにエンティティを戻す。
  // デコードを分割より後にするのは、&#0; がマーカーと誤認されないようにするため。
  // セグメント内の素の改行は元テキスト由来なので保持し、空行の連続だけまとめる。
  const lines = [];
  const rawLines = work
    .split(BOUNDARY)
    .filter((segment) => segment !== "")
    .flatMap((segment) => decodeHtmlEntities(segment).split("\n"));
  for (const rawLine of rawLines) {
    const line = rawLine.replace(/[ \t ]+/g, " ").trim();
    if (line === "" && (lines.length === 0 || lines[lines.length - 1] === "")) {
      continue;
    }
    lines.push(line);
  }
  while (lines.length > 0 && lines[lines.length - 1] === "") {
    lines.pop();
  }

  let text = lines.join("\n");
  let truncated = false;
  if (maxLength !== null && Number.isFinite(maxLength) && maxLength > 0 && text.length > maxLength) {
    text = `${text.slice(0, maxLength)}\n…(truncated)`;
    truncated = true;
  }
  return { text, truncated };
}
