---
name: gkill-docs
description: "gkill の資料層の役割分担と保守手順。AGENTS.md / CLAUDE.md / .claude/skills/ / documents/adr/ / documents/reverse/ / resources/manual_src/ / ABOUT_TEST.md のどれが何の正本か、npm run verify_docs が機械検査する内容（件数・リンク・ファイル名実在・ADR 構造・スキル索引網羅・サイズ上限・個人情報パターン・マニュアル生成鮮度と用語）、ADR の書き方と番号帯、マニュアルは manual_src だけを7言語セットで編集すること、スキルの追加・変更手順を扱う。資料・スキル・ADR・マニュアル原稿を編集するとき、verify_docs が落ちたとき必読。"
---

# 資料層の役割分担と保守手順

対象: `AGENTS.md` / `CLAUDE.md` / `.claude/skills/**` / `documents/**` / `resources/manual_src/**` / 各 `README.md`・`ABOUT_TEST.md`

**資料の正本の分割規約（どの層が何を持つか）は [documents/adr/README.md](../../../documents/adr/README.md) の「正本の分割規約」が正。** ここでは資料を触るときの手順だけを持つ。

## Documentation

- `resources/manual/` — HTML manuals (7 languages, 22 pages per language), embedded via `//go:embed` and served at `/resources/manual/`
- `documents/adr/` — Architecture Decision Record（現在 86 件）。**なぜそうなっているか**、とくに**採らなかった案とその理由**を残す層。Reverse docs = What / ADR = Why。禁止文の正本はこの CLAUDE.md とコードコメントのままで、ADR が持つのは却下案・実測値・事件譚だけ。索引と運用ルールは [documents/adr/README.md](../../../documents/adr/README.md)
- `documents/reverse/` — Reverse-engineered design documents (24 files). See `documents/reverse/README.md` for index. Key files: glossary.md (96 terms), api-endpoints.md (92 endpoints, 90 registered), usecase.md (86 use cases), sequence-diagrams.md (29 diagrams), scenario.md (cross-channel end-to-end usage scenarios with UML), testing-guide.md. `npm run verify_docs` (`src/tools/verify_docs.mjs`) machine-checks the counts, cross-links, referenced paths, Mermaid blocks, and manual freshness — it runs as part of `npm test`, so update the docs when a count changes.
- `src/ABOUT_TEST.md` — Test specification index, links to 23 subdirectory `ABOUT_TEST.md` files
- **What `verify_docs` covers** (extend it rather than hand-maintaining new numbers): file/test counts computed from the tree (`--list` prints them all), count assertions across `documents/reverse/*.md` + `CLAUDE.md` + `src/**/README.md` + `ABOUT_TEST.md`, markdown link resolution for that same set, `src/...` path references (warning; the backtick scan strips fenced blocks first — pairing across a ``` fence used to hide 65% of them), **a check that every filename mentioned in a doc actually exists** (this one alone caught 17 ghost entries left behind by deletions; `_`-prefixed suffix patterns and `xxx` placeholders are exempt), **ADR の番号帯**（`checkADRBands` — 帯の重なり・どの帯にも入らない番号・残り空きが10未満の帯を落とす。帯表は `documents/adr/README.md`「番号の付け方」がパースされる）, Mermaid block types, manual generation freshness / language page-set parity / a11y invariants / intra-manual links, a **terminology lint** that rejects internal code names (`IDF`, `WAN`, `Kyou`, `MiReKyou`, `Dnote`, `rudbeckia`, …) in `resources/manual_src/` outside `<code>` spans and `href`/`src` values, and a check that every `screen_name` the app passes to `HelpDialog` has a matching manual page. `--parity` (opt-in) reports per-page h2/h3/table drift against the Japanese original.

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
- 個人情報（実在の利用者ID・人名・企業名・端末名・メールアドレス・端末のローカル絶対パス）をリポジトリへ書かない。`checkPersonalInfo` が資料に加えて `src/**`・`resources/**`・`documents/**`・`.github/**` のテキストをパターン検査する（対象は git 追跡分+未追跡の非ignore分。`resources/gkill_sample_data/` は実データ由来を許容する運用で対象外）。パターンで表せない固有の NG 語は、リポジトリ直下の `verify_docs_personal_ngwords.local.txt`（gitignore 済み・1行1語・大小無視の部分一致・コミットしない）に置くとローカル検査に加わる。3文字級の短語は複合語や base64 に偶発一致するので、`_` などの区切りを含めた形で書く
- **コミット前の `npm run verify_docs` は絶対に飛ばさない。** `.githooks/pre-commit` が機械強制し、`core.hooksPath` は `npm i` の postinstall が設定する。フックが未設定の環境（クローン直後に `npm i` を打っていない等）でも、コミット前に手で実行すること。`--no-verify` の使用は何があっても禁止（AGENTS.md「AI エージェントへの約束」）

## 関連スキル

- [gkill-build-test](../gkill-build-test/SKILL.md) — `npm test`（verify_docs を含む）の実行

## 詳しい設計と却下案（ADR）

- [ADR-0802 plaing 綴りの凍結（ADR-0806 が置き換え）](../../../documents/adr/0802-freeze-plaing-spelling.md)
- [ADR-0806 綴りは凍結せず直す（互換を残さず、旧綴りのデータは一度きりで復旧）](../../../documents/adr/0806-fix-spellings-instead-of-freezing.md)
- [ADR-0803 verify_docs はファイル名の実在も検査する](../../../documents/adr/0803-verify-docs-checks-filenames.md)
- [ADR-0804 CLAUDE.md を AGENTS.md とスキルへ分割](../../../documents/adr/0804-split-claude-md-into-skills.md)
- [ADR-0805 ADR の採番はサブシステム別100番幅](../../../documents/adr/0805-adr-numbering-by-subsystem-hundreds.md)
