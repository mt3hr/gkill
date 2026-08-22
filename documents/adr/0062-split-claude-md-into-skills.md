# ADR-0062: AI向け規約は AGENTS.md（核）と規約スキルへ分割し、入口の肥大化を機械検査で防ぐ

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-22 |
| Sources | 分割前の `CLAUDE.md`（500行・126,140B） / `src/tools/verify_docs.mjs` / `.gitignore` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/tools/verify_docs.mjs`（checkSkills / checkAgentEntrypoints / checkADRSources / checkPersonalInfo） |

## Context

`CLAUDE.md` が 500行・126,140B まで肥大化し、毎セッション全文が常時コンテキストへ読み込まれていた。
節別に実測分類すると、**全タスクで常に必要なのは約 14.8KB（12%）**で、残り約 108KB（86%）は
「特定領域のファイルを触るときだけ必要な不変条件ブロック」だった（クライアント系だけで約 55KB）。

同時に、Claude Code 以外の AI エージェント（Codex CLI / Cursor / Gemini CLI / Copilot 等）向けの
入口が1つも無く、ルート `README.md` から `CLAUDE.md` へのリンクすら無かった。

## Decision

正本を3層に分ける。

- **`AGENTS.md`** — 全AI共通の核（ビルド・ソースツリー・命名・凍結綴り・Kyou 型一覧・`errors` は成功時 null）と、
  **「触る場所ごとの必読資料」ルーティング表**（触るパス → スキル → 読み落とすと何が起きるか）＋症状表。
  Claude Code は `CLAUDE.md` の `@AGENTS.md` import で展開して読む
- **`.claude/skills/gkill-*/SKILL.md`（13本）** — 領域別の禁止文・不変条件の**正本**。逐語移送で内容は変えない。
  `.gitignore` を `/.claude/*` + `!/.claude/skills/` へ書き換えて git 追跡し、`verify_docs` の検査対象に載せる
- **`CLAUDE.md`・他AI入口（copilot-instructions / .cursor/rules / .gemini/settings.json）** — 導線だけ。規約本文ゼロ

再発防止を `verify_docs` で機械化する: 入口のサイズ上限（`checkAgentEntrypoints`。**上限に当たったら
上限を上げず中身をスキルへ落とす**）、スキル ↔ ルーティング表の双方向網羅と frontmatter 検査（`checkSkills`）、
ADR の Sources が指す節名の実在（`checkADRSources`）、個人情報パターンの混入検査（`checkPersonalInfo`）。
件数アサーション21件は移送先ファイルへ付け替えた。

## Rejected alternatives

- **内容を `CLAUDE.md` と `AGENTS.md` の両方に複製する** — 片方だけ直す事故が構造的に必ず起きる。
  シンボリックリンクは Windows（git 設定で symlink 無効）で使えない
- **中立ディレクトリ（documents/agents/ 等）を正本にして SKILL.md は生成スタブにする** — 生成スクリプトと
  鮮度検査という可動部が増え、Claude は「スキル → 正本」の2ホップ読込になる。スキル機構が読む場所を
  そのまま正本にすれば可動部はゼロで済む
- **ネストした AGENTS.md（src/client/AGENTS.md 等）で領域分割する** — Codex のネスト読込は cwd 基準で、
  gkill の作業は常にリポジトリルートから行うため発火しない
- **不変条件1件=1行の全件索引を常時側に置き、id で本文と双方向機械検査する** — 約50行・13KB の常時コストと
  id 保守（索引行・本文見出し・移行台帳の3点整合）が重い。ルーティング表の stakes 列＋症状表＋既存の
  ソース走査テスト5本で読み落としを受ける
- **サイズ上限を設けない** — 今回の肥大化（126KB）がそのまま再発する
- **`documents/reverse/` へ混ぜる** — What（現状）と禁止文（規約）の層が混ざり、`documents/adr/README.md`
  「正本の分割規約」が崩れる
- **スキル別の `.cursor/rules/*.mdc` / `.github/instructions/*` を手書きで並べる** — 本文の手書き複製は即
  ドリフトする。入口はポインタ1本ずつに留め、必要になったら生成方式を別途検討する

## Consequences

- 常時コンテキストは 126KB → 約 20KB（AGENTS.md ≈17KB ＋ CLAUDE.md ラッパ ＋ スキル description 13本）。
  領域別の本文は該当作業のときだけ読み込まれる
- 読み落としの防御は「ルーティング表の stakes 列（常時）→ スキル description の自動発動（Claude）→
  ソース内アンカーコメント → ソース走査テスト（CI）」の多層になる。オンデマンド化した以上、
  読み落としの可能性はゼロにならない — stakes 列に実測値と症状を書き続けることが唯一の対価
- ADR 20本の Sources が規約スキルを指すようになり、`checkADRSources` が節名の実在を検査する。
  **スキル内の節（太字リード文）の文言を変えると CI が落ちる**（意図した挙動。節名は出典アンカーである）
- `.gitignore` の `/.claude` は `/.claude/*` + `!/.claude/skills/` になった。`settings.local.json` は
  引き続き無視される。skills が再び ignore されると `checkSkills` と件数アサーションが大声で落ちる
- 衛星リポジトリ（gkill の家族）へも「AGENTS.md（核）＋規約スキル＋薄い CLAUDE.md」の同型を
  横展開できる（本 ADR のスコープ外）

## Evidence

- 分割前 `CLAUDE.md` の節別実測: 全タスク常時 ≈14.8KB(12%) / 領域別 ≈108KB(86%)（うち src/client 系 ≈55KB、
  rykv/mi 列系 13.7KB・KFTL 6.1KB・タグ 4.8KB・ポート 3.5KB）/ ほぼ参照されない ≈2.5KB
- 分割後の実測: AGENTS.md 17,321B / スキル13本 計約 147KB（オンデマンド）/ CLAUDE.md 802B
- `verify_docs.mjs` の件数アサーション `add('CLAUDE.md', …)` は21件あり、素の部分文字列一致のため
  移送と同時の付け替えが必須だった
- ADR の Sources に `CLAUDE.md`「節名」が20ファイル。これはどの検査にも掛かっていなかった
- `git check-ignore` 実測: `/.claude`（ディレクトリ除外形）のままでは `!/.claude/skills/**` を足しても
  再包含されない（git はネガティブパターンで親ディレクトリの除外を打ち消せない）

## Related tests

- `src/tools/verify_docs.mjs` — `checkSkills`（0本検出・frontmatter・ルーティング表双方向）/
  `checkAgentEntrypoints`（サイズ上限・`@AGENTS.md`・入口の規約本文混入）/ `checkADRSources` /
  `checkPersonalInfo`。`verify_docs.mjs` 自身に単体テストは無いので、検査を変えたら
  わざと壊して落ちることを手で確認する（ADR-0061 と同じ運用）
- 逐語移送の検証は移行時に1度だけ行った（旧 `CLAUDE.md` の全行が移送先に存在することの行単位照合）。
  恒久テストにはしない — 旧版に永久に縛られるため
