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

const BLOCK_TAG_SET = new Set(BLOCK_TAGS);

// BOUNDARY はタグ由来の改行位置を表す内部マーカー。入力に含まれていたものは先に消す。
const BOUNDARY = String.fromCharCode(0);

// タグ開始の判定。sticky フラグで走査位置から直接マッチさせる。
const TAG_START_REGEX = /<(\/?)([a-zA-Z][a-zA-Z0-9-]*)/y;

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
 * stripTags はHTMLからタグを取り除き、タグ由来の改行位置を BOUNDARY マーカーで
 * 表したテキストを返す。コメント・script・style は中身ごと落とす。
 *
 * 正規表現replaceの繰り返しだと未閉鎖の <!-- や <script>、"</script foo>" 形式の
 * 終了タグが取りこぼされて中身が本文に漏れるため、1パスの文字走査で処理する。
 * タグ由来の改行に素の "\n" を使わないのは、</div><div> のような隣接だけで
 * 空行ができて入れ子の深いHTMLが空行だらけになるため。マーカーの連続は
 * 後段でまとめて1つの改行に落とす。
 *
 * @param {string} html 対象のHTML。
 * @returns {string} タグを除去し BOUNDARY マーカーを挿入したテキスト。
 */
function stripTags(html) {
  let out = "";
  let i = 0;
  const len = html.length;
  while (i < len) {
    const ch = html[i];
    if (ch === BOUNDARY) {
      // 入力に混ざっていた内部マーカーはタグ由来の改行と区別できないので捨てる
      i++;
      continue;
    }
    if (ch !== "<") {
      out += ch;
      i++;
      continue;
    }
    if (html.startsWith("<!--", i)) {
      const commentEnd = html.indexOf("-->", i + 4);
      i = commentEnd === -1 ? len : commentEnd + 3;
      continue;
    }
    // <!doctype ...> や <?xml ...?> はタグごと捨てる
    if (html[i + 1] === "!" || html[i + 1] === "?") {
      const declEnd = html.indexOf(">", i + 2);
      i = declEnd === -1 ? len : declEnd + 1;
      continue;
    }
    TAG_START_REGEX.lastIndex = i;
    const tagMatch = TAG_START_REGEX.exec(html);
    if (!tagMatch) {
      // タグ開始に見えない "<" (例: "a < b") はリテラルとして残す
      out += ch;
      i++;
      continue;
    }
    const isClosing = tagMatch[1] === "/";
    const tagName = tagMatch[2].toLowerCase();
    const tagEnd = html.indexOf(">", i + tagMatch[0].length);
    // ">" の無い未閉鎖タグは末尾まで捨てる
    i = tagEnd === -1 ? len : tagEnd + 1;
    if (!isClosing && (tagName === "script" || tagName === "style")) {
      i = skipRawTextContent(html, i, tagName);
      continue;
    }
    if (tagName === "br" || BLOCK_TAG_SET.has(tagName)) {
      out += BOUNDARY;
    }
  }
  return out;
}

/**
 * skipRawTextContent は <script>/<style> の中身を終了タグごと読み飛ばす。
 *
 * @param {string} html 対象のHTML。
 * @param {number} from 開始タグの ">" の直後の位置。
 * @param {string} tagName "script" または "style" (小文字)。
 * @returns {number} 終了タグの ">" の直後の位置。終了タグが無ければ末尾。
 */
function skipRawTextContent(html, from, tagName) {
  const lowerHtml = html.toLowerCase();
  const closer = "</" + tagName;
  let i = from;
  while (i < html.length) {
    const found = lowerHtml.indexOf(closer, i);
    if (found === -1) {
      return html.length;
    }
    const after = html[found + closer.length];
    // "</scripter>" のような別タグ名は終了タグではないので読み進める
    if (after !== undefined && after !== ">" && after !== "/" && !/\s/.test(after)) {
      i = found + closer.length;
      continue;
    }
    const closeEnd = html.indexOf(">", found + closer.length);
    return closeEnd === -1 ? html.length : closeEnd + 1;
  }
  return html.length;
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

  const work = stripTags(html);

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
