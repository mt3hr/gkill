---
name: gkill-docs
description: "gkill の資料層の役割分担と保守手順。AGENTS.md / CLAUDE.md / .claude/skills/ / documents/adr/ / documents/reverse/ / resources/manual_src/ / ABOUT_TEST.md のどれが何の正本か、npm run verify_docs が機械検査する内容（件数・リンク・ファイル名実在・ADR 構造・スキル索引網羅・サイズ上限・個人情報パターン・マニュアル生成鮮度と用語）、ADR の書き方と番号帯、マニュアルは manual_src だけを7言語セットで編集すること、スキルの追加・変更手順を扱う。資料・スキル・ADR・マニュアル原稿を編集するとき、verify_docs が落ちたとき必読。"
---

# 資料層の役割分担と保守手順

対象: `AGENTS.md` / `CLAUDE.md` / `.claude/skills/**` / `documents/**` / `resources/manual_src/**` / 各 `README.md`・`ABOUT_TEST.md`

**資料の正本の分割規約（どの層が何を持つか）は [documents/adr/README.md](../../../documents/adr/README.md) の「正本の分割規約」が正。** ここでは資料を触るときの手順だけを持つ。

## Documentation

- `resources/manual/` — HTML manuals (7 languages, 22 pages per language), embedded via `//go:embed` and served at `/resources/manual/`
- `documents/adr/` — Architecture Decision Record（現在 43 件）。**なぜそうなっているか**、とくに**採らなかった案とその理由**を残す層。Reverse docs = What / ADR = Why。禁止文の正本はこの CLAUDE.md とコードコメントのままで、ADR が持つのは却下案・実測値・事件譚だけ。索引と運用ルールは [documents/adr/README.md](../../../documents/adr/README.md)
- `documents/reverse/` — Reverse-engineered design documents (24 files). See `documents/reverse/README.md` for index. Key files: glossary.md (96 terms), api-endpoints.md (92 endpoints, 90 registered), usecase.md (87 use cases), sequence-diagrams.md (29 diagrams), scenario.md (cross-channel end-to-end usage scenarios with UML), testing-guide.md. `npm run verify_docs` (`src/tools/verify_docs.mjs`) machine-checks the counts, cross-links, referenced paths, Mermaid blocks, and manual freshness — it runs as part of `npm test`, so update the docs when a count changes.
- `src/ABOUT_TEST.md` — Test specification index, links to 23 subdirectory `ABOUT_TEST.md` files
- **What `verify_docs` covers** (extend it rather than hand-maintaining new numbers): file/test counts computed from the tree (`--list` prints them all), count assertions across `documents/reverse/*.md` + `CLAUDE.md` + `src/**/README.md` + `ABOUT_TEST.md`, markdown link resolution for that same set, `src/...` path references (warning; the backtick scan strips fenced blocks first — pairing across a ``` fence used to hide 65% of them), **a check that every filename mentioned in a doc actually exists** (this one alone caught 17 ghost entries left behind by deletions; `_`-prefixed suffix patterns and `xxx` placeholders are exempt), Mermaid block types, manual generation freshness / language page-set parity / a11y invariants / intra-manual links, a **terminology lint** that rejects internal code names (`IDF`, `WAN`, `Kyou`, `MiReKyou`, `Dnote`, `rudbeckia`, …) in `resources/manual_src/` outside `<code>` spans and `href`/`src` values, and a check that every `screen_name` the app passes to `HelpDialog` has a matching manual page. `--parity` (opt-in) reports per-page h2/h3/table drift against the Japanese original.

## 規約スキル（.claude/skills/）の保守手順

- スキルを足す/消す/改名したら、`AGENTS.md` のルーティング表を**同じコミットで**更新する（`checkSkills` が双方向網羅を検査して落とす）
- 件数の入った文をスキルへ移した/書いたら、`src/tools/verify_docs.mjs` の `buildCountAssertions()` の該当 `add()` の第1引数を**同じコミットで**差し替える（照合は素の部分文字列一致なので、語句が移った瞬間に赤くなる）
- 不変条件の節（太字リード文）の文言を変えない。ADR の `Sources` が節名の文字列で指しており、`checkADRSources` が「そのファイルにその節名があるか」を検査する
- スキルからの Markdown リンクは `../../../documents/...`（3階層固定）。バッククォートの `src/...` パス表記はリポジトリルート基準で検査されるので書き換え不要
- スキルに Mermaid を書かない（`checkMermaid` の対象は `documents/reverse/` だけ。検査されない図はドリフトする）
- 1スキル = SKILL.md 1ファイル。同ディレクトリに補助 `.md` を置かない（`checkSkills` が落とす）
- `AGENTS.md` / `CLAUDE.md` に領域別の規約本文を書き足さない。サイズ上限（`checkAgentEntrypoints`）に当たったら、上限を上げるのではなく中身をスキルへ落とす
- 高リスクなソースの先頭にはアンカーコメント「編集前に読む: `.claude/skills/<name>/SKILL.md`」を置いてある。参照先の実在は `checkSkillAnchors` が検査するので、スキルを改名したらコメントも追随させる
- 個人情報（実在の利用者ID・人名・メールアドレス・端末のローカル絶対パス）を資料へ書かない。`checkPersonalInfo` が資料全体をパターン検査する。パターンで表せない固有の NG 語は、リポジトリ直下の `verify_docs_personal_ngwords.local.txt`（gitignore 済み・1行1語・コミットしない）に置くとローカル検査に加わる

## 関連スキル

- [gkill-build-test](../gkill-build-test/SKILL.md) — `npm test`（verify_docs を含む）の実行

## 詳しい設計と却下案（ADR）

- [ADR-0060 plaing 綴りの凍結](../../../documents/adr/0060-freeze-plaing-spelling.md)
- [ADR-0061 verify_docs はファイル名の実在も検査する](../../../documents/adr/0061-verify-docs-checks-filenames.md)
- [ADR-0062 CLAUDE.md を AGENTS.md とスキルへ分割](../../../documents/adr/0062-split-claude-md-into-skills.md)
