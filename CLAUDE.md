# CLAUDE.md

このリポジトリの規約は [AGENTS.md](AGENTS.md) にある（Claude Code / Codex CLI / Cursor / Gemini CLI 等で共通）。
次の1行がその内容をこのファイルへ展開する。**規約の編集は AGENTS.md 側で行うこと。**

@AGENTS.md

## Claude Code 固有

- 領域別の不変条件は `.claude/skills/gkill-*/`（13スキル）が正本。該当パスを触るとスキルとして自動で読める。
  スキルが発動しなくても、AGENTS.md のルーティング表から同じファイルを Read すれば同じ内容になる。
- このファイルと AGENTS.md に領域別の規約本文を書き足さないこと。`npm run verify_docs` が
  サイズ上限・スキル索引の網羅・個人情報パターンを機械検査する。
