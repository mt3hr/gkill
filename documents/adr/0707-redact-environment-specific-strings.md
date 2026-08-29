# ADR-0707: 端末固有の文字列は出口で伏せ、プラグインの診断文はAIへ返さない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | `.claude/skills/gkill-mcp/SKILL.md`「プラグインの診断文はAIへ返さない」節 / `.claude/skills/gkill-go-backend/SKILL.md`「レスポンスへ載る自由文は端末固有の情報を伏せてから出す」節 / `src/tools/verify_docs.mjs` の `checkPersonalInfo` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/message/redact.go` / `src/mcp/lib/plugin-tools.mjs` |

## Context

`gkill_plugin_uguisu` の `last_error` に、端末のローカル絶対パスがそのまま載っていた。

診断文を書いているのは別リポジトリ（uguisu）側で、しかも**意図的**である。「1件も返せなかった」ときに
読み取り元の指定とホームディレクトリを stderr へ書く。LocalSystem 起動でホームが
`C:\Windows\System32\config\systemprofile` に化ける事故は、それでしか診断できない。

そこから外への経路は無加工の3ホップだった。

| 段 | 場所 | 加工 |
|---|---|---|
| 収集 | `plugin_repository_impl.go` の `cmd.Stderr = io.MultiWriter(os.Stderr, p.stderrRing)` | 生バイト列を末尾4KB保持 |
| API | `handle_get_plugin_list.go` の `LastError: pluginRepo.LastStderr()` | なし |
| MCP | `plugin-tools.mjs` の `plugins: Array.isArray(response.plugins) ? response.plugins : []` | なし（素通し） |

`mcp-server-base.mjs` は payload 全体を `JSON.stringify` してテキストブロックに載せるので、
**4KB の生 stderr は必ず AI の目に入る**。AIがそれを資料やコミットメッセージへ引き写せば、
`verify_docs.mjs` の `checkPersonalInfo` が防いでいる混入がそのまま成立する。あの検査は
「リポジトリへ書き込まれたテキスト」の静的検査で、実行時にAIへ渡る文字列は原理的に対象外である。

同じ形の穴は `last_error` だけではなかった。`typed_index.last_build_error` には起動失敗のエラーが
文字列化されて入り、`fmt.Errorf("... plugin %s (%s): %w", ..., execPath, ...)` の `execPath` が乗る。
`content_error`（MCP の `inlinePluginContents`）も元は `handle_get_plugin_content_html.go` が
`err.Error()` を埋めた `ErrorMessage` で、同じ `execPath` が乗る。

## Decision

**防御線は gkill 側の出口に引く。** プラグインは別リポジトリの成果物であり、
「診断文に実パスを書くな」を書き手側の約束にはできない（どのプラグイン作者が書いても同じことが起きる）。
2層に重ねる。

**1. Go: レスポンスへ載る自由文は伏せてから出す。**
`message.RedactEnvironmentSpecific` が正本で、置換の基準は `verify_docs.mjs` の `checkPersonalInfo` と
同じ3パターンに揃える（＝伏せた文字列が引き写されても機械検査に落ちない）。

- `C:\Users\〈ユーザー名〉` / `/home/〈ユーザー名〉/` … ホーム配下のユーザー名の区間だけ
- `〈メールアドレス〉` … `@example.` 以外のメールアドレス

適用は `GkillError.MarshalJSON`（`ErrorMessage`）と、`GkillError` ではない
`last_error` / `typed_index.last_build_error` の2フィールドへの明示的な呼び出し。

**2. MCP: 診断文の中身はAIへ返さない。**
`handlePluginToolCall` が `plugins[]` から `last_error` と `typed_index.last_build_error` を落とし、
非空だったときだけ `has_last_error` / `has_last_build_error` を立て、**実際に落としたときだけ**
`warnings` へ1行足す。

パスの**形は残す**。`C:\Windows\System32\config\systemprofile` はユーザー名を含まないので素通しになり、
上に書いた LocalSystem の診断は従来どおり成立する。サーバのコンソールログには適用しない ——
端末に閉じた人間の診断チャネルで、リポジトリへ入る経路が無い。

## Rejected alternatives

- **uguisu 側の診断文を直す** — 直すべきは gkill 側。uguisu の記述は診断として正しく、
  同じことは**どのプラグインでも起きる**。プラグインは別リポジトリの成果物なので、
  書き手側の約束はレビューもテストも効かない。

- **絶対パスを丸ごと潰す（末尾の名前だけ残す）** — 「ホームが `systemprofile` を指している」という
  最も重要な診断が消える。実際にその事故を踏んでおり、そのとき見たのはパスの**形**だった。
  ユーザー名の区間だけを伏せれば、機械検査の基準は満たしたうえで形が残る。

- **MCP 側で落とすだけにする（Go では伏せない）** — HTTP API を直に叩く経路に生が残る。
  MCPサーバ自身がそうしているように、AIはAPIを直接叩ける。

- **Go 側で伏せるだけにする（MCPでは中身を返す）** — 伏せてもディレクトリ構成は残り、
  それがAIの文脈に入れば引き写される。`has_*` だけで外部監査 D2 の診断
  （「is_alive=true なのに0件」）は成立するので、中身を渡す必要がない。

- **生成側（`ErrorMessage:` の代入 684 箇所）で1つずつ包む** — 必ず足し忘れる。
  現に `err.Error()` を埋めているのは4箇所だけだが、再発防止になるのは
  「新しいハンドラが同じ書き方をしても自動で載る」ことのほうで、それは marshal 境界にしか無い。

- **`warnings` を常に出す** — 落としていないのに出せば常時ノイズになり、本当に効く警告まで
  読まれなくなる（ADR-0609 と同じ判断）。証拠がある（＝実際に落とした）ときだけ出す。

- **`verify_docs` の検査を実行時にも走らせる** — 検査器はビルド時のNodeスクリプトで、
  Goのランタイムからは呼べない。基準だけを揃え、正規表現を二重管理しない。

## Consequences

**プラグインの診断文をAPIへ足すときは `message.RedactEnvironmentSpecific` を通すこと。**
`GkillError` に載せるなら marshal 境界が自動で拾うが、独自フィールドは拾われない。

**MCP から `last_error` の中身は読めない。** 「is_alive=true なのに0件」を追うときは
`has_last_error` を見て、テキストが要るなら gkill を動かしている人にサーバのコンソールを
読んでもらう。AIが自分で読んで引き写す経路をあえて塞いである。

`GkillError.MarshalJSON` は全APIのエラー応答を通るが、`error_code` は触らない
（HTTPステータスの判定に使う値なので変えてはいけない）。

## Evidence

`gkill_plugin_uguisu` の `last_error` に端末のローカル絶対パスが載っている状態を実際に確認した
（2026-08-24 の利用者からの指摘）。変異テストでも確認済み ——
`handle_get_plugin_list.go` から伏せる呼び出しを外すと
`TestHandleGetPluginList_RedactsEnvironmentSpecificDiagnostics` が落ち、
失敗出力にプラグインディレクトリの絶対パスがユーザー名ごと現れる。

`ErrorMessage:` の代入は 684 箇所、うち `fmt.Sprintf` を使うのは 7 箇所、
`err.Error()` を埋めて実パスが乗りうるのは 4 箇所（いずれも 2026-08-24 時点の実測）。

## Related tests

- `src/server/gkill/api/message/redact_test.go`
- `src/server/gkill/api/message/gkill_error_test.go`
- `src/server/gkill/api/gkill_server_api/handle_get_plugin_list_test.go`
- `src/mcp/__tests__/plugin-tools.test.mjs`
