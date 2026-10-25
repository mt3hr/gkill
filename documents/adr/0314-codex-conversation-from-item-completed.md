# ADR-0314: Codex 0.151 以降の会話は item_completed から読み、返事は最終回答を本文にして経過を畳む

| | |
|---|---|
| Status | Accepted |
| Date | 2026-10-25 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md`「プラグイン一覧と実測仕様」 / `src/plugins/gkill_plugin_codex/README.md` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/plugins/gkill_plugin_codex/loader.go` |

## Context

この ADR は [ADR-0306](0306-codex-thread-id-from-filename.md) の「会話は event_msg レーンだけ」を新しい形式へ広げるもので、置き換えではない。
スレッド識別子・identity の取り方・Kyou ID の作り方などほかの決定は ADR-0306 のまま有効。

Codex CLI は 0.151 から、会話をロールアウトに書く場所を変えた。それまで人間の発言と返事は
`event_msg/user_message`・`agent_message`・`agent_reasoning`（旧レーン）に書かれていたが、
0.151 以降はこれらが1行も出ず、`event_msg/item_completed` の `item.type` が
`UserMessage`・`AgentMessage`・`Reasoning`（新レーン）の行にだけ書かれる。
変更ファイルは `patch_apply_end` から `FileChange` へ、Web 検索は `response_item/web_search_call` から
`Extension`（kind `web.search`）へ移った。ツールの実行結果も `CommandExecution`・`McpToolCall` として同じ種別に載る。

プラグインは旧レーンしか読んでいなかったので、新しいログでは人間の発言が1件も見つからなかった。
返事の区切りが付かず、1セッションのツール呼び出しが全部1件の返事に溜まった（「exec ×数千」「所要 数千分」）。
例外もエラーも出ず、設定画面の件数も0にはならないので、壊れていることに気付けなかった。

あわせて、新しい Codex は作業中に追加の指示を受け付け、質問ツール（`request_user_input_async`）で人間に尋ねる。
回答は `<send_user_message_question_reply>` に包まれた JSON（`[{question, answer}]`）として人間の発言の行に載る。

## Decision

会話は旧レーンと新レーンの両方から読み、旧レーンの会話が出たターンでは新レーンの会話を捨てる。
Kyou は Claude.ai プラグインと同じ1メッセージ1件（人間の発言で1件、次の発言までの返事で1件）のままとする。
返事は最終回答（`phase=final_answer`）と質問を本文にし、途中報告・思考・ツール・変更ファイルは「作業の経過」1つに畳む。

## Rejected alternatives

- **会話を `response_item/message` から作る** — 新しい形式でも role=developer の指示や `<environment_context>` などの注入が混ざる。ADR-0306 と同じ理由で使わない。新レーンの `UserMessage` は人間が打った入力だけを持つ。

- **ファイルごとにどちらかのレーンを選ぶ（旧レーンが1行でもあれば新レーンを読まない）** — 旧版で始めたセッションを新版で resume すると、1ファイルの前半が旧レーン、後半が新レーンになる。ファイル単位で選ぶと後半が丸ごと消える。ターン単位なら両方を拾え、同じターンに両方が並んだときだけ片方を捨てられる。

- **重複を気にせず両方のレーンを読む** — 両方を書く版があった場合、同じ発言が2件ずつ並ぶ。旧版のころから新レーンにしか無い計画（Plan）だけは重複判定の対象にしない。

- **Kyou を Codex のターン（turn_id）単位にする** — 新しい Codex は1ターンの中に作業中の追加指示と質問への回答を何度も受け付けるので、1件が数十時間ぶんの作業になりうる。各発言の時刻もタイムラインから消える。利用者は Claude.ai プラグインと同じ1メッセージ1件を選んだ。

- **途中報告も含めて Codex の発言を全部本文に並べる** — 実ログでは最終回答1つに途中報告がおよそ6つ付く。rykv は一覧の行に詳細 HTML をそのまま描くので、会話が途中報告に埋もれる。途中報告は経過の中に時系列で残し、検索テキストにも入れるので、読めなくなるわけではない。

- **質問をツール一覧の1行のままにする** — ツールの要約は200字で切れ、設問と選択肢が読めない。回答だけが次の発言として宙に浮く。

- **質問と最終回答を両方とも本文に出す** — Codex は質問ツールを呼ぶとき、たいてい同じ文面を最終回答にも書く（実ログで3分の2ほど）。最終回答が設問の書き出しを含むときは、質問を経過の側へ回す。

- **item_completed を全部溜めてから中身で捨てる** — `item_completed` のバイトの9割はコマンドの出力（`CommandExecution`）と `Extension` が占め、画像生成の行は1行で 2MiB を超える。上限超過として数えられ、設定画面に「ログの形式が変わった可能性」という誤った警告が出る。`item.type` と `item.kind` は先頭512バイトにあるので、旧レーンと同じく先頭だけで読み捨てる。

- **パーサを直したことを README で案内し、利用者にキャッシュを消してもらう** — 差分更新は変わっていないファイルを二度と読まないので、配布しただけでは過去のセッションが直らない。消し忘れても気付けない。キャッシュの世代（`cacheSchemaVersion`）を上げ、次の起動で作り直させる。

## Consequences

**Codex がまたログの形式を変えると、会話は例外もエラーも出さずに消える。** 手がかりは、
設定画面の「知らない種別」と、人間の発言が0件のまま返事1件にツール呼び出しが溜まる症状（所要時間のチップが数千分になる）。
知らない `item.type` は読み捨てずに拾うが、会話として解釈するのは上の種類だけ。

新しい形式のスレッドで以前にできていた「スレッド丸ごと1件」の Kyou は無くなり、発言ごとの Kyou に置き換わる。
これに付けたタグやテキストの紐付けは切れる。旧形式のファイルは発言の連番が変わらないので、Kyou ID は動かない。

質問の重複判定は、設問の書き出し（30文字）が最終回答にそのまま含まれるかで見る。
最終回答が設問を要約して言い直した場合は、両方が本文に出る（害は重複表示だけで、情報は落ちない）。

旧レーンのターンの追跡は `task_started` と `turn_context` の turn_id に頼る。
どちらも出ない形のログでは、旧レーンの会話はすべて空の turn_id に属すとみなし、同じく turn_id の無い新レーンの項目を捨てる。

キャッシュの世代を上げたので、配布後の初回起動ではすべてのファイルを読み直し、構築が終わるまで検索は空になる。

## Evidence

- 手元の実ログ（0.151〜0.163 の数ファイル・GB 級）: 旧レーンの `user_message` / `agent_message` は0行、新レーンの `UserMessage` は数百件
- 新レーンの `item.type` は全行で先頭512バイト以内にあった。`item_completed` のバイトの9割超は `CommandExecution` と `Extension`
- 人間の発言のうち1割強が質問への回答（JSON の包み）、1割ほどがアプリ内ブラウザの前置き付き。同じターンに人間の発言が複数ある箇所は数十
- 直したプラグインで実ログを構築: 数十秒、Kyou は人間の発言と返事がそれぞれ数百件で、ほぼ同数。上限超過で落とした行は0
- 合成フィクスチャ（`testdata/item_lane/` の1ファイル、テスト由来）で、発言3件・返事3件に分かれること、質問への回答が回答文だけになることを固定

## Related tests

- `src/plugins/gkill_plugin_codex/loader_test.go`
- `src/plugins/gkill_plugin_codex/reader_test.go`
- `src/plugins/gkill_plugin_codex/fold_test.go`
- `src/plugins/gkill_plugin_codex/render_test.go`
