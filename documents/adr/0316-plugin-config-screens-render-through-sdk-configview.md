# ADR-0316: 同梱プラグインの設定画面 HTML は SDK の configview が1か所で描き、プラグインは文言と統計を渡すだけにする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-07 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md`「プラグイン一覧と実測仕様」 / `documents/reverse/plugin-system.md`「設定ファイル（config.json）」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/plugin/sdk/configview/configview.go` |

## Context

同梱プラグイン7本（ChatGPT / Claude.ai / Claude Code / Codex / fitbit / 位置情報 / archived_git_commit_log）は、
設定画面の HTML（`GetConfigHTML` の応答）をそれぞれ自前の `html.go` で組んでいた。head（CSS 変数と規則）・
保存スクリプト（`gkill_plugin_config` の postMessage と `gkill_plugin_config_result` の受信）・取り込みの進捗・
統計表・「設定されている指定」「展開結果」「データソースを変える」の各節・config.json の記入例が7本の近似コピーで、
本文 HTML を chatview へ寄せた（[ADR-0315](0315-chat-plugins-render-through-sdk-chatview.md)）あとも
設定画面だけが残っていた。コピーごとに育った結果、次の差が積もり、どれも例外もエラーも出さない。

- Claude.ai 版は textarea / button の規則で `--field-bg` / `--btn-bg` / `--btn-color` を参照していたが、
  head にその変数の定義が無い。入力欄は背景が透明、ボタンは文字が見えない
- ChatGPT 版は textarea / button の規則そのものを持たず、ブラウザ既定の見た目で出る
- 明暗の決め方が2系統。4本（チャット系と archived_git）は OS の配色（`prefers-color-scheme`）だけに追従し、
  fitbit / 位置情報は親からの `gkill_theme` も見る。設定ダイアログ（use-plugin-config-dialog.ts）は iframe の load で
  `gkill_theme` を送っているので、前者は gkill のテーマと食い違うことがある
- 進捗の出し方。fitbit だけ件数と割合の棒を出し、他は「取り込み中 n / m」の文だけ。位置情報は進捗を出さない
- 保存スクリプトは欄を増やすたびに手で書き足す（fitbit / 位置情報で5〜6欄）。欄の id と postMessage のキーを
  片方だけ変えると、設定したつもりの値が保存されない
- 件数の桁区切り・時刻の書式・0 件の表記（`-` / `—` / 空）が揃っていない

## Decision

同梱プラグインの設定画面 HTML は SDK のサブパッケージ `plugin/sdk/configview` が1か所で描く。
プラグインの `html.go` は自分の文言と統計を `configview.Page`（題・読み込み状況の帯・進捗・統計表・警告・
指定一覧・展開結果・追加の節・編集フォーム・config.json の案内）へ写すだけにし、head・保存スクリプト・
進捗の棒・一覧の上限・エスケープは `configview.Render` が出す。

- 文言の欄は全部 `configview.Prose` を通す。描くのは `` `code` `` と `**強調**` の2つだけで、他はエスケープする。
  HTML を直接受け取る欄は置かない
- 編集フォームは `Field{Key, Kind, Label, Value}` の列で宣言し、要素 id（`gkill_<Key>`）と postMessage のキー（`<Key>`）と
  保存スクリプトの var 名（`<Key>` の camelCase）を同じ Key から生成する。チェックボックスは `'1'` / `''` で送る
  （PostConfig の読み方に合わせた）。Key の形が外れた欄は黙って落とさず、画面に警告を出す
- 明暗は2段。OS の配色に追従し、親から `gkill_theme` が届けば `data-theme` が勝つ（`light` なら OS が暗くても明るくする）
- 進捗は状態（scanning / ingesting / folding）ごとの文言を SDK が持ち、件数があれば割合と棒を出す
- 件数は3桁区切り（`configview.Count`）、時刻は `2006-01-02 15:04:05`（`FormatUnix` / `FormatTime`）、未実行は `-`、
  空の値は `—`、一覧は既定 20 件で「ほか N 件」
- source_dirs の説明文は `SourceDirsNote` が組み、プラグインは「マッチしたものをどう扱うか」の1文と固有の補足だけを渡す
- postMessage のキーは `gkill_theme` / `gkill_plugin_config` / `gkill_plugin_config_result` の3つだけで、
  `configview_test.go` が出力中のキー集合を固定する。参照している CSS 変数が明暗の両方で定義されていることも同じテストが見る

## Rejected alternatives

- **7本の複製を残して差分だけ揃える** — ADR-0315 と同じ理由。揃えた直後から再び別々に育つ。Claude.ai の未定義変数は、
  ChatGPT 版からコピーして textarea の規則だけ足したときに起きた（変数の定義は別の版から来ていた）
- **chatview に設定画面も入れる** — chatview は会話形式の本文で、fitbit / 位置情報 / archived_git は使わない。
  設定画面は7本全部が使う。使う集合が違うものを1パッケージにすると、片方の変更で両方のテストと利用者が動く
- **文言の欄に生の HTML を受け取る（`template.HTML` 型）** — 文言は Go のリテラルなので安全に見えるが、
  走査の問題文やエラー文を同じ欄に流し込んだ瞬間にエスケープされない。`` `code` `` と `**強調**` だけの記法で
  7本の文言は全部書けた（`<ol>` の手順・`<code>` のパスとキー・太字の件数）
- **フォームの保存スクリプトを欄ごとにプラグインが書く** — id とキーと var 名を3か所で揃える作業が残り、
  fitbit の `html_test.go` が守っている「現在値と保存キーが同じ欄に揃っている」を7本ぶん書くことになる
- **設定画面の体裁をクライアント側（Vue）に持たせ、プラグインは JSON を返す** — `get_config_html` が HTML を返す契約に
  第三者プラグインが乗っている。ADR-0315 の却下理由と同じ
- **明暗を `gkill_theme` だけにする** — 設定ダイアログが送るのは load の1回で、届く前の一瞬と古い本体では無スタイルになる。
  OS 追従を下敷きにして親の通知で上書きするのが、どちらが欠けても壊れない形

## Consequences

- **SDK を直しても、7本のバイナリを作り直して配り直すまで表示は変わらない。** 本体だけ更新しても旧 HTML のまま
- 「データソースを変える」「指定を変える」の見出しは「設定」に揃う。config.json の記入例は折りたたみに入る（フォームが主経路）
- fitbit の進捗の帯は橙から緑になり、他の6本には割合と棒が付く。位置情報の警告は表の前に出る
- archived_git の「取り込んだリポジトリ」の行は `rep名 <zip!/.git> n コミット` の順になる（名前 → パス → 説明の固定順）
- 新しい同梱プラグインの設定画面は `configview` を使うこと。独自の head を書くと、ここに挙げた差が再び生まれる
- postMessage のキーを足すときは、`configview_test.go` の集合検査・クライアント・cross-boundary-map §9 を同時に変える

## Evidence

- 7本の `html.go` は合計約1,800行で、head が各60〜110行、保存スクリプトが各20〜35行の複製だった。
  置き換え後は各60〜170行で、残ったのは文言・統計の行・警告の条件・フォームの欄の宣言だけ（コード由来）
- fitbit の `html_test.go` と archived_git の `render_test.go` の設定画面テストは無修正で通る（テスト由来）
- `configview_test.go` が参照中の CSS 変数の定義を検査し、Claude.ai 版の形（`var(--field-bg)` の参照だけがある）を
  落とすことを確かめた（テスト由来）
- 見た目は Playwright で明暗それぞれ描いて目視（表示の決定。実測なし）

## Related tests

- `src/server/gkill/plugin/sdk/configview/configview_test.go`
- `src/server/gkill/plugin/sdk/configview/form_test.go`
- `src/server/gkill/plugin/sdk/configview/text_test.go`
- `src/plugins/gkill_plugin_fitbit/html_test.go`
- `src/plugins/gkill_plugin_archived_git_commit_log/render_test.go`
- `src/server/gkill/usecase/source_conventions_scan_test.go`
