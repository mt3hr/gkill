---
name: gkill-docs
description: "gkill の資料層の役割分担と保守手順。AGENTS.md / CLAUDE.md / .claude/skills/ / documents/adr/ / documents/reverse/ / resources/manual_src/ / ABOUT_TEST.md のどれが何の正本か、npm run verify_docs が機械検査する内容（件数・リンク・ファイル名実在・ADR 構造・スキル索引網羅・サイズ上限・個人情報パターン・マニュアル生成鮮度と用語）、ADR の書き方と番号帯、マニュアルは manual_src だけを7言語セットで編集すること、スキルの追加・変更手順を扱う。資料・スキル・ADR・マニュアル原稿を編集するとき、verify_docs が落ちたとき必読。"
---

# 資料層の役割分担と保守手順

対象: `AGENTS.md` / `CLAUDE.md` / `.claude/skills/**` / `documents/**` / `resources/manual_src/**` / 各 `README.md`・`ABOUT_TEST.md`

**資料の正本の分割規約（どの層が何を持つか）は [documents/adr/README.md](../../../documents/adr/README.md) の「正本の分割規約」が正。** ここでは資料を触るときの手順だけを持つ。

## Documentation

- `resources/manual/` — HTML manuals (7 languages, 24 pages per language), embedded via `//go:embed` and served at `/resources/manual/`
- `documents/adr/` — Architecture Decision Record（現在 131 件）。**なぜそうなっているか**、とくに**採らなかった案とその理由**を残す層。Reverse docs = What / ADR = Why。禁止文の正本はこの CLAUDE.md とコードコメントのままで、ADR が持つのは却下案・実測値・事件譚だけ。索引と運用ルールは [documents/adr/README.md](../../../documents/adr/README.md)
- `documents/reverse/` — Reverse-engineered design documents (25 files). See `documents/reverse/README.md` for index. Key files: glossary.md (98 terms), api-endpoints.md (97 endpoints), cross-boundary-map.md (every cross-language / cross-process hop — TS ↔ `/api` ↔ Go handler ↔ usecase, MCP, Android, Wear OS, CLI, plugin stdio, iframe postMessage — as hand-written tables that `checkCrossBoundaryDoc` compares with the code; when you add such a hop, add its row there), usecase.md (94 use cases), sequence-diagrams.md (29 diagrams), scenario.md (cross-channel end-to-end usage scenarios with UML), testing-guide.md. `npm run verify_docs` (`src/tools/verify_docs.mjs`) machine-checks the counts, cross-links, referenced paths, Mermaid blocks, and manual freshness — it runs as part of `npm test`, so update the docs when a count changes.
- `src/ABOUT_TEST.md` — Test specification index, links to 24 subdirectory `ABOUT_TEST.md` files
- **What `verify_docs` covers** (extend it rather than hand-maintaining new numbers): file/test counts computed from the tree (`--list` prints them all), count assertions across `documents/reverse/*.md` + `CLAUDE.md` + `src/**/README.md` + `ABOUT_TEST.md`, markdown link resolution for that same set, `src/...` path references (warning; the backtick scan strips fenced blocks first — pairing across a ``` fence used to hide 65% of them), **a check that every filename mentioned in a doc actually exists** (this one alone caught 17 ghost entries left behind by deletions; `_`-prefixed suffix patterns and `xxx` placeholders are exempt), **ADR の番号帯**（`checkADRBands` — 帯の重なり・どの帯にも入らない番号・残り空きが10未満の帯を落とす。帯表は `documents/adr/README.md`「番号の付け方」がパースされる）, the **cross-boundary map** (`checkCrossBoundaryDoc`: the 14 marked tables in `documents/reverse/cross-boundary-map.md` against the route table, handlers, `GkillAPI`, the Service Worker, MCP, Wear OS, CLI, plugin host/SDK and postMessage keys; a missing Web → Go row is reported with a ready-to-paste expected row), **reverse doc index coverage** (`checkReverseDocIndex`: every reverse doc is in the README reading list, summary table and dependency graph, and in the folder-structure tree), **test file index coverage** (`checkTestFileCoverage`: the basename of every test file that `listTestFiles()` enumerates — `src/server` `*_test.go` including MCP, vitest `*.test.ts`, Playwright `*.spec.ts`, `src/tools/__tests__` `*.test.mjs|js|ts`, and Android / Wear OS `.kt` files containing `@Test`; the standalone plugin modules under `src/plugins/` are excluded — must appear in some `ABOUT_TEST.md`, so a new test file needs an index line in the same commit), **the category table sum in `src/server/ABOUT_TEST.md`** (`checkServerAboutTestCategorySum`: the rows of the `| カテゴリ | … |` table must add up to its `**合計 N ファイル**` line), Mermaid block types, manual generation freshness / language page-set parity / a11y invariants / intra-manual links, a **terminology lint** that rejects internal code names (`IDF`, `WAN`, `Kyou`, `MiReKyou`, `Dnote`, `rudbeckia`, … and their lower-case forms `idf` / `kyou` / `kftl` / `rykv` / `dnote` / `timeis` / …, matched as whole words; lower-case `playing` is left out because it is an English word) and internal identifiers (`/api/`, `ERR000`, `GKILL_HOME`, `FindQuery`, `tx_id`, matched as substrings) in `resources/manual_src/`. It looks only at what the reader sees: `<code>` / `<pre>` (with or without attributes), `<script>` / `<style>` and comments are dropped, and of the tag attributes only the `alt` / `title` / `aria-label` values are checked (`href` / `src` / `class` / `id` are not). The lists and the stripping are `USER_DOC_FORBIDDEN_TERMS` / `USER_DOC_FORBIDDEN_IDENTIFIERS` / `manualProseForTermCheck`（`documents/reverse/user-guide.md` も同じ語で検査する。コードフェンス・インラインコード・Markdown リンクのリンク先を除いた本文を見る）, and a check that every `screen_name` the app passes to `HelpDialog` has a matching manual page. `--parity` (opt-in) reports per-page h2/h3/table drift against the Japanese original.

## 規約スキル（.claude/skills/）の保守手順

- スキルを足す/消す/改名したら、`AGENTS.md` のルーティング表を**同じコミットで**更新する（`checkSkills` が双方向網羅を検査して落とす）
- 件数の入った文をスキルへ移した/書いたら、`src/tools/verify_docs.mjs` の `buildCountAssertions()` の該当 `add()` の第1引数を**同じコミットで**差し替える（照合は素の部分文字列一致なので、語句が移った瞬間に赤くなる）
- 不変条件の節（太字リード文）の文言を変えない。ADR の `Sources` が節名の文字列で指しており、`checkADRSources` が「そのファイルにその節名があるか」を検査する
- ADR を書くときの番号は、`documents/adr/README.md`「番号の付け方」の帯表からサブシステムで選ぶ（100番幅。正本はあの表だけ）。**帯が窮屈でも空いている別の帯へ逃がさない。** 逃がすと番号不変ルールで恒久的にずれたまま残る（実例と経緯は [ADR-0805](../../../documents/adr/0805-adr-numbering-by-subsystem-hundreds.md)）。`checkADRBands` が残り空き10未満で落とすので、落ちたら帯を広げるか新しい帯を切ること
- スキルからの Markdown リンクは `../../../documents/...`（3階層固定）。バッククォートの `src/...` パス表記はリポジトリルート基準で検査されるので書き換え不要
- スキルに Mermaid を書かない（`checkMermaid` の対象は `documents/reverse/` だけ。検査されない図はドリフトする）
- 1スキル = SKILL.md 1ファイル。同ディレクトリに補助 `.md` を置かない（`checkSkills` が落とす）
- `AGENTS.md` / `CLAUDE.md` に領域別の規約本文を書き足さない。サイズ上限（`checkAgentEntrypoints`）に当たったら、上限を上げるのではなく中身をスキルへ落とす
- 高リスクなソースの先頭にはアンカーコメント「編集前に読む: `.claude/skills/<name>/SKILL.md`」を置いてある。参照先の実在は `checkSkillAnchors` が検査するので、スキルを改名したらコメントも追随させる
- 個人情報（実在の利用者ID・人名・企業名・端末名・メールアドレス・端末のローカル絶対パス）をリポジトリへ書かない。`checkPersonalInfo` が資料に加えて `src/**`・`resources/**`・`documents/**`・`.github/**`・ルート直下の設定と資料・`.githooks/` のテキストをパターン検査する（依存の lock とライセンス一覧は第三者の文字列なので対象外）（対象は git 追跡分+未追跡の非ignore分。`resources/gkill_sample_data/` は実データ由来を許容する運用で対象外）。作業ツリーから消しても `git rm` でステージしていないファイルは、コミットに載る index の版を読んで検査する（`readPersonalInfoTarget`。飛ばすと、消したつもりの実データ入りファイルがステージ漏れのままコミットされる）。パターンで表せない固有の NG 語は、リポジトリ直下の `verify_docs_personal_ngwords.local.txt`（gitignore 済み・1行1語・大小無視の部分一致・コミットしない）に置くとローカル検査に加わる。3文字級の短語は複合語や base64 に偶発一致するので、`_` などの区切りを含めた形で書く
- **コミット前の `npm run verify_docs` は絶対に飛ばさない。** `.githooks/pre-commit` が機械強制し、`core.hooksPath` は `npm i` の postinstall が設定する。フックが未設定の環境（クローン直後に `npm i` を打っていない等）でも、コミット前に手で実行すること。`--no-verify` の使用は何があっても禁止（AGENTS.md「AI エージェントへの約束」）

## 実データ由来の値の線引き

資料・コメント・テスト・コミットメッセージに、利用者の実環境や実記録から得た値を書かない。代わりの形で書く（理由と却下案は ADR-0807）。

| 書かない | 代わりに書く形 |
|---|---|
| 実データの件数・rep 数・容量・所要時間・割合、実測から逆算した値 | 数万件・数百 rep・数百 MB・GB 級・十数秒・9 割超 |
| 観測した日付（事故・実測・実利用報告・レビュー・監査・利用者報告の日） | 書かない。変更日（ADR の `Date`、「X 日に改名」）は書いてよい |
| 座標・端末 ID・場所 ID・歩数・時刻付きの書き出し名 | `351234000`・`35.1234`・`1000000001`・`10000` 歩・`takeout-20240101T000000Z` |
| 実在のリポジトリ名・rep 名・タグ名・端末名・画面解像度・プロセス起動時刻 | `repo-alpha`・`Box_TestPC_20240101`・`TagA`・`testuser` |
| テストの固定時刻で秒や小数秒が切りの悪いもの | 秒 00・小数秒 000（`2024-01-01T00:00:00Z`） |

書いてよい具体値は、コード・テスト由来の件数（verify_docs の件数検査に載せる）、ベンチとサンプルデータの値、仕様の定数、リポジトリ内ファイルの行数とサイズ。**同じ行に出所（コード由来・テスト由来・ベンチ・サンプルデータ・ファイル名）を書く。** ADR の Evidence も同じで、実環境で測った値は概数で書く。

機械検査（この PC の利用者設定側にあり、無い環境では省略される）:

- AI が書く瞬間: 座標・端末 ID などの硬い形は止め、精密な数値は警告する
- `.githooks/pre-commit`: staged の**追加行**に形を当てる。verify_docs が要求する件数字句（`node src/tools/verify_docs.mjs --count-phrases`）と、削除側にある同じ値（既存行の手直し）は通す。テスト系のパスでは精密な数値は注意だけになる
- `.githooks/commit-msg`: メッセージ全文に同じ形を当てる。出所の目印は同じ行にしか効かない
- 誤検知で止まったら、利用者の了承を取ってから行に `shape-ok（理由）` を付ける。形では拾えない実在の名前は NG 語辞書の担当

## 関連スキル

- [gkill-build-test](../gkill-build-test/SKILL.md) — `npm test`（verify_docs を含む）の実行

## 詳しい設計と却下案（ADR）

- [ADR-0802 plaing 綴りの凍結（ADR-0806 が置き換え）](../../../documents/adr/0802-freeze-plaing-spelling.md)
- [ADR-0806 綴りは凍結せず直す（互換を残さず、旧綴りのデータは一度きりで復旧）](../../../documents/adr/0806-fix-spellings-instead-of-freezing.md)
- [ADR-0803 verify_docs はファイル名の実在も検査する](../../../documents/adr/0803-verify-docs-checks-filenames.md)
- [ADR-0804 CLAUDE.md を AGENTS.md とスキルへ分割](../../../documents/adr/0804-split-claude-md-into-skills.md)
- [ADR-0807 実データ由来の値は、書く瞬間とコミットの瞬間に形で止める](../../../documents/adr/0807-stop-real-data-values-at-write-and-commit.md)
- [ADR-0805 ADR の採番はサブシステム別100番幅](../../../documents/adr/0805-adr-numbering-by-subsystem-hundreds.md)
