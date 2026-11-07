# ADR-0315: AI チャット系プラグインの本文 HTML は SDK の chatview が1か所で描き、プラグインは共通モデルへ写すだけにする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-07 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md`「プラグイン一覧と実測仕様」 / `documents/reverse/plugin-system.md`「7. GetContentHTML — Kyou 1 件分の HTML 生成」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/plugin/sdk/chatview/chatview.go` |

## Context

AI チャット系の同梱プラグイン4本（ChatGPT / Claude.ai / Claude Code / Codex）は、Kyou 本文の HTML
（`GetContentHTML` の応答）をそれぞれ自前で組んでいた。ChatGPT と Claude.ai はほぼ同じコードの写しで、
Claude Code と Codex はそこから別々の方向へ育ち、本文生成のコードは1行も共有されていなかった。
結果として次の差が積もり、どれも例外もエラーも出さない。

- 見つからないときのページが ChatGPT / Claude.ai だけ無スタイル（テーマにも追従しない）
- 人間の吹き出しのクラスと CSS 変数が ChatGPT だけ `user` / `--msg-user-bg`（他は `human`）
- チップの区切り。Claude Code は MCP の HTML→テキスト変換で1語に連結されないよう空白を置いたが、
  Codex は詰めたままで、テキストでは project 名と branch 名がつながって読めた
- 返事の組み方。Codex は最終回答を本文にして途中報告・thinking・ツールを「作業の経過」1つに畳む
  （[ADR-0314](0314-codex-conversation-from-item-completed.md)）が、Claude Code は全テキストを並べ、
  thinking とツール群だけを個別に畳んでいた。rykv の一覧の行では Claude Code の返事が途中報告に埋もれる
- 応答側のラベルが Claude.ai も Claude Code も「Claude」で、モデル名は Codex しか出さない
- 4本とも Markdown を描かず、コードフェンスの記号がそのまま見える
- `<details>` の開閉で iframe の高さを測り直す `toggle` リスナーが ChatGPT / Claude.ai に無い

head（CSS 変数・テーマ追従・`gkill_iframe_size` の通知スクリプト）は4本に複製されていて、
1本を直しても残りは直らない。

## Decision

AI チャット系プラグインの本文 HTML は SDK のサブパッケージ `plugin/sdk/chatview` が1か所で描く。
プラグインの `render.go` は自分のログの形を `chatview.Message`（会話名・チップ・役割とラベル・本文・
前置き・本文ブロック・作業の経過・時刻）へ写すだけにし、head・吹き出し・チップ・折りたたみ・
「見つかりません」ページ・上限での打ち切りは `chatview.Render` / `RenderNotFound` が出す。

- 返事の組み方は ADR-0314 の形を4本に広げる。最終回答（Codex は `phase=final_answer`、
  Claude Code は印が無いので最後のテキスト）と質問を本文に、残りを「🛠 作業の経過」1つに時系列で畳む
- 出す情報の欄を揃える。会話名・チップ（project・branch・model・所要時間のうち持っているもの）・ラベル・本文・経過・時刻。
  Claude Code は `message.model` を、ChatGPT は `message.metadata.model_slug` を解析し始める（キャッシュ世代を上げる）
- 応答側のラベルは製品名（ChatGPT / Claude / Claude Code / Codex）。人間側は4本とも「あなた」
- Markdown はコードフェンスとインラインコードだけを描く（`chatview.RenderText`）。他はエスケープしたプレーンテキストのまま
- チップは空白1つで区切る。postMessage のキーは `gkill_iframe_size` / `gkill_theme` の2つだけで、テストが出力中のキー集合を固定する

## Rejected alternatives

- **プラグインごとの複製を残して差分だけ揃える** — 揃えた直後から再び別々に育つ。4本とも個別に `render.go` を持ち、
  片方だけ直すと他方は静かにずれる（チップの空白がその実例）
- **クライアント側で描く（プラグインは JSON を返す）** — `GetContentHTML` が HTML を返す契約に、
  MCP の `HtmlToText`・一覧行の `srcdoc`・第三者プラグインが乗っている。プロトコルと `plugin-html-view.vue` の両方を変えることになり、
  第三者プラグインの HTML は描けなくなる
- **goldmark で Markdown 全体を描く** — SDK は server モジュールの中なので gkill_server 本体にも依存が増え、
  4本のバイナリにも入る。AI の返事は Markdown とは限らず、見出し・箇条書きを描くと MCP のテキストの形まで変わる。
  利用者は「コードブロックだけ」を選んだ
- **プラグインごとの配色** — 一覧の行では4本の記録が交互に並ぶ。配色の変数は gkill の見た目であって、プラグインの個性ではない
- **`Handler.GetContentHTML` を廃して描画モデルを返させる** — fitbit や archived_git_commit_log は会話ではないので同じモデルに乗らない。
  契約は変えず、会話形式のプラグインが使う部品として置く
- **Claude Code の「全テキストを並べる」組み方を残す** — ADR-0314 が Codex で却下した理由（一覧の行で会話が途中報告に埋もれる）が
  Claude Code にもそのまま当たる
- **`chatview` を `sdk` パッケージにフラットに置く** — `sdk` はプロトコルの包で第三者の作者が読む。描画の型を接頭辞付きで混ぜると
  公開面が倍になる。サブパッケージなら到達可能性テストの許可リスト1行で済む

## Consequences

- **SDK を直しても、4本のバイナリを作り直して配り直すまで表示は変わらない。** 本体だけ更新しても旧 HTML のまま
- **Claude Code（世代5）と ChatGPT（世代3）はキャッシュを作り直す。** 配布後の初回起動は構築が終わるまで検索が空になる（ADR-0314 と同じ受容済みの結果）。
  Kyou ID は4本とも変えていないので、付けたタグ・テキストは外れない
- Claude Code の返事は、それまで本文に並んでいた途中のテキストが「作業の経過」の中へ移る。検索テキストには入ったままなので引ける
- MCP の `include_plugin_content` のテキストから、コードフェンスの記号と言語名が消える。`<pre>` はブロック要素として前後で改行する
- クライアントの「本文をコピー」（`extract_text_from_html`）は `textContent` なので、コードブロックと前後の文がつながる。
  今も送信者ラベル・チップ・本文がつながっており、現状比で後退ではない
- 新しい会話形式のプラグインは `chatview` を使うこと。独自の head を書くと、ここに挙げた差が再び生まれる
- postMessage のキーを足すときは、`chatview_test.go` の集合検査・クライアント・cross-boundary-map §9 を同時に変える

## Evidence

- 4本の head はそれぞれ約130行の複製で、変数の数だけが違っていた（コード由来）
- Codex の `render.go` は約600行から約230行、Claude Code の `render.go` は約370行から約120行へ減り、
  残ったのは本文と経過の分け方・前置きの見出し・サブエージェントの見出しだけ（コード由来）
- Codex の既存の描画テスト7本は無修正で通る。ClaudeCode のフィクスチャでは返事のチップが
  `myproj` / `main` / `claude-test-model` / `所要 1分0秒` の4つになる（テスト由来）
- `chatview_test.go` が出力中の `gkill_*` キー集合を `{gkill_iframe_size, gkill_theme}` に固定し、`dblclick` が出ないことも見る（テスト由来）
- 見た目は手元の dev 環境で目視（実測なし — 表示の決定）

## Related tests

- `src/server/gkill/plugin/sdk/chatview/chatview_test.go`
- `src/server/gkill/plugin/sdk/chatview/text_test.go`
- `src/plugins/gkill_plugin_codex/render_test.go`
- `src/plugins/gkill_plugin_claudecode/render_test.go`
- `src/plugins/gkill_plugin_chatgpt/render_test.go`
- `src/plugins/gkill_plugin_claudeai/render_test.go`
- `src/server/gkill/mcp/html_text_test.go`
