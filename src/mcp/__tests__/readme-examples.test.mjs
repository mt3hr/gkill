/**
 * README の実行可能な例と現行スキーマの同期検査 (2026-08-30 MCPレビュー P1)。
 *
 * verify_docs はツール数などの件数しか守れず、README の JSON 例が
 * スキーマとずれても検出できない。実際に Mi の例が include_*_mi を欠いたまま
 * 「コピーすると0件」の状態で放置されていた。
 * README 中の ```json ブロックは gkill_get_kyous の引数例なので、
 * 実物の正規化器 (normalizeKyouArgs) へそのまま通す。キーの改名・廃止・
 * 意味変更で例が壊れたら、利用者より先にこのテストが落ちる。
 */

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, test, expect } from "vitest";

import { normalizeKyouArgs } from "../lib/normalization.mjs";
import { READ_TOOLS } from "../lib/read-tools.mjs";
import { FIND_QUERY_SCHEMA } from "../lib/find-query-schema.mjs";

const README_PATH = resolve(dirname(fileURLToPath(import.meta.url)), "../README.md");

// ```json フェンスの中身を出現順に取り出す。README に json フェンスで載るのは
// gkill_get_kyous の引数例だけ、という前提がこのテストの土台
// (別ツールの例を足すときはフェンス言語を変えるか、このテストへ分岐を足すこと)。
function extractJsonBlocks(markdown) {
  const blocks = [];
  const fence = /```json\r?\n([\s\S]*?)```/g;
  let match;
  while ((match = fence.exec(markdown)) !== null) {
    blocks.push(match[1]);
  }
  return blocks;
}

describe("README の gkill_get_kyous 例は現行スキーマで正規化できる", () => {
  const readme = readFileSync(README_PATH, "utf8");
  const blocks = extractJsonBlocks(readme);

  test("json ブロックが存在する (抽出の空振りをテスト成功と混同しない)", () => {
    expect(blocks.length).toBeGreaterThanOrEqual(6);
  });

  test("すべての例が JSON として妥当で、正規化器を通る", () => {
    for (const [index, block] of blocks.entries()) {
      let args;
      try {
        args = JSON.parse(block);
      } catch (error) {
        throw new Error(`README json example #${index + 1} is not valid JSON: ${error.message}`);
      }
      try {
        normalizeKyouArgs(args);
      } catch (error) {
        throw new Error(`README json example #${index + 1} does not normalize: ${error.message}`);
      }
    }
  });

  // for_mi は include_*_mi を最低1つ要求する (全て無指定は0件+warning)。
  // README の Mi 例がこの制約を欠いたまま出荷されていたのが元指摘。
  test("for_mi の例は include_*_mi を最低1つ持つ", () => {
    const parsed = blocks.map((block) => JSON.parse(block));
    const miExamples = parsed.filter((args) => args.query?.for_mi === true);
    expect(miExamples.length).toBeGreaterThan(0);
    const projectionFlags = [
      "include_create_mi",
      "include_check_mi",
      "include_limit_mi",
      "include_start_mi",
      "include_end_mi",
    ];
    for (const args of miExamples) {
      const hasProjection = projectionFlags.some((flag) => args.query[flag] === true);
      expect(hasProjection).toBe(true);
    }
  });

  // ページング例の cursor は「next_cursor を verbatim で返す」教えどおり
  // v2 複合形式 ({RFC3339}::{ID}) で示す。素の ISO 日時の例は旧形式の教材になる。
  test("cursor の例は v2 複合形式で示されている", () => {
    const parsed = blocks.map((block) => JSON.parse(block));
    const cursorExamples = parsed.filter((args) => typeof args.cursor === "string");
    expect(cursorExamples.length).toBeGreaterThan(0);
    for (const args of cursorExamples) {
      expect(args.cursor).toContain("::");
    }
  });

  // README のパラメータ表に散文で列挙された group_by の語彙は、正規化器では検証されない
  // (スキーマ enum の検査はクライアント側)。今回の修正前は url_domain だけ抜けた状態で
  // 放置されていたので、表の列挙をスキーマ enum とまるごと突き合わせる。
  test("group_by の語彙列挙はスキーマ enum と一致する", () => {
    const match = readme.match(/バケット集計（([^）]+)）/);
    expect(match, "README の group_by 行 (バケット集計（…）) が見つからない").toBeTruthy();
    const documented = match[1].split("/").map((value) => value.trim());
    const kyousTool = READ_TOOLS.find((tool) => tool.name === "gkill_get_kyous");
    expect(documented).toEqual(kyousTool.inputSchema.properties.group_by.enum);
  });

  // mi_sort_type は「対応する include_*_mi 射影があるときだけ効く」(スキーマの説明文)。
  // include_*_mi 最低1つの検査 (上) はこの対応ズレを検出できないので、例が自分の
  // 注意書きを守っていることまで見る。
  test("mi_sort_type の例は対応する include_*_mi 射影を立てている", () => {
    const SORT_TO_PROJECTION = {
      create_time: "include_create_mi",
      estimate_start_time: "include_start_mi",
      estimate_end_time: "include_end_mi",
      limit_time: "include_limit_mi",
    };
    // 対応表の語彙がスキーマ enum から乖離したら、まずここで気づく。
    expect(Object.keys(SORT_TO_PROJECTION).sort()).toEqual(
      [...FIND_QUERY_SCHEMA.properties.mi_sort_type.enum].sort(),
    );
    const parsed = blocks.map((block) => JSON.parse(block));
    const sortExamples = parsed.filter((args) => typeof args.query?.mi_sort_type === "string");
    expect(sortExamples.length).toBeGreaterThan(0);
    for (const args of sortExamples) {
      const projection = SORT_TO_PROJECTION[args.query.mi_sort_type];
      expect(projection, `unknown mi_sort_type ${args.query.mi_sort_type}`).toBeDefined();
      expect(args.query[projection], `example with mi_sort_type:${args.query.mi_sort_type} lacks ${projection}:true`).toBe(true);
    }
  });
});
