/**
 * Tests for lib/html-text.mjs — プラグインのコンテンツHTMLをAI向けの
 * プレーンテキストに落とす変換。
 */

import { describe, test, expect } from "vitest";
import { htmlToText, decodeHtmlEntities } from "../lib/html-text.mjs";

describe("decodeHtmlEntities", () => {
  test("decodes named entities", () => {
    expect(decodeHtmlEntities("a &lt;b&gt; c &quot;d&quot;")).toBe('a <b> c "d"');
  });

  test("decodes numeric and hex entities", () => {
    expect(decodeHtmlEntities("&#65;&#x42;")).toBe("AB");
  });

  test("decodes &amp; last so escaped entities survive", () => {
    expect(decodeHtmlEntities("&amp;lt;script&amp;gt;")).toBe("&lt;script&gt;");
  });

  test("decodes nbsp to a normal space", () => {
    expect(decodeHtmlEntities("a&nbsp;b")).toBe("a b");
  });

  test("leaves unknown entities untouched", () => {
    expect(decodeHtmlEntities("&notarealentity; x")).toBe("&notarealentity; x");
  });

  test("drops out-of-range code points instead of throwing", () => {
    expect(decodeHtmlEntities("&#x999999999;")).toBe("");
  });
});

describe("htmlToText", () => {
  test("returns empty result for empty or non-string input", () => {
    expect(htmlToText("")).toEqual({ text: "", truncated: false });
    expect(htmlToText(null)).toEqual({ text: "", truncated: false });
    expect(htmlToText(undefined)).toEqual({ text: "", truncated: false });
  });

  test("drops style and script contents", () => {
    const html =
      "<html><head><style>body { color: red; }</style>" +
      "<script>var x = 1; document.write('nope');</script></head>" +
      "<body><p>本文</p></body></html>";
    const { text } = htmlToText(html);
    expect(text).toBe("本文");
    expect(text).not.toContain("color");
    expect(text).not.toContain("document.write");
  });

  test("drops HTML comments", () => {
    expect(htmlToText("<p>a<!-- hidden -->b</p>").text).toBe("ab");
  });

  test("drops script contents even with attributes on the end tag", () => {
    // "</script\t\n bar>" のような終了タグも終了として扱う (CodeQL js/bad-tag-filter)
    const { text } = htmlToText("<p>a</p><script>var secret = 1;</script\t\n bar><p>b</p>");
    expect(text).toBe("a\nb");
    expect(text).not.toContain("secret");
  });

  test("does not end script at a longer tag name like </scripter>", () => {
    const { text } = htmlToText("<script>var x = '</scripter>';</script><p>ok</p>");
    expect(text).toBe("ok");
  });

  test("drops unclosed comment, script and style to the end of input", () => {
    expect(htmlToText("<p>a</p><!-- never closed <p>hidden</p>").text).toBe("a");
    expect(htmlToText("<p>a</p><script>var leak = 1;").text).toBe("a");
    expect(htmlToText("<p>a</p><style>body { color: red; }").text).toBe("a");
  });

  test("drops an unclosed tag instead of leaving it as text", () => {
    expect(htmlToText("<p>a</p><span class=x").text).toBe("a");
  });

  test("keeps a literal '<' that does not start a tag", () => {
    expect(htmlToText("<p>a < b</p>").text).toBe("a < b");
  });

  test("removes uppercase and attribute-heavy tags", () => {
    const { text } = htmlToText('<SCRIPT type="text/javascript">nope();</SCRIPT><P CLASS="x">a</P>');
    expect(text).toBe("a");
  });

  test("drops doctype and processing instructions", () => {
    expect(htmlToText('<!doctype html><?xml version="1.0"?><p>a</p>').text).toBe("a");
  });

  test("turns block boundaries and br into newlines", () => {
    const { text } = htmlToText("<div>one</div><div>two<br>three</div>");
    expect(text).toBe("one\ntwo\nthree");
  });

  test("keeps inline tags inline", () => {
    expect(htmlToText("<p>a <strong>bold</strong> word</p>").text).toBe("a bold word");
  });

  test("keeps details/summary text (plugin collapses tool calls there)", () => {
    const html = "<details><summary>ツール実行 3件</summary><pre>Read foo.go</pre></details>";
    const { text } = htmlToText(html);
    expect(text).toContain("ツール実行 3件");
    expect(text).toContain("Read foo.go");
  });

  test("collapses runs of blank lines into one", () => {
    const { text } = htmlToText("<div>a</div><div></div><div></div><div>b</div>");
    expect(text).toBe("a\nb");
  });

  test("decodes entities in the extracted text", () => {
    expect(htmlToText("<p>if a &lt; b &amp;&amp; c &gt; d</p>").text).toBe("if a < b && c > d");
  });

  test("does not resurrect escaped markup as tags", () => {
    // プラグインは本文を html.EscapeString してから埋め込むので、
    // エスケープ済みの <script> はタグ除去より後にデコードされて本文として残る。
    const { text } = htmlToText("<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>");
    expect(text).toBe("<script>alert(1)</script>");
  });

  test("truncates at maxLength and reports it", () => {
    const { text, truncated } = htmlToText("<p>abcdefghij</p>", { maxLength: 4 });
    expect(truncated).toBe(true);
    expect(text).toBe("abcd\n…(truncated)");
  });

  test("does not truncate when under maxLength", () => {
    const { text, truncated } = htmlToText("<p>abc</p>", { maxLength: 10 });
    expect(truncated).toBe(false);
    expect(text).toBe("abc");
  });

  test("ignores non-positive maxLength", () => {
    expect(htmlToText("<p>abc</p>", { maxLength: 0 }).truncated).toBe(false);
  });
});
