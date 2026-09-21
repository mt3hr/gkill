# ADR-0310: ChatGPT / Claude.ai の会話履歴はエクスポート ZIP のまま読み、展開済みの JSON は読まない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-18 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md`「プラグイン一覧と実測仕様」 / `documents/reverse/plugin-system.md`「取り込み元の ZIP — sdk.OpenSources」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/plugins/gkill_plugin_chatgpt/source.go` / `src/plugins/gkill_plugin_claudeai/source.go` |

## Context

ChatGPT と Claude.ai のエクスポートはどちらも ZIP で届く。

- ChatGPT: `<sha256>-<書き出し日時>-<uuid>.zip`。実物は数百エントリで、会話は ZIP 直下の `conversations-000.json` … `conversations-005.json`（旧エクスポートは `conversations.json` 1本）。残りは `chat.html`（数十MB）・`user.json`・添付ファイル（`file-*.dat`・画像・音声）
- Claude.ai: `conversations-000.zip` の中に `conversations.json` が1本（数百MB）。**エントリの更新時刻は 1980-01-01（ZIP の元期）で固定**

2026-09-18 まで両プラグインは展開済みの `conversations*.json` を `filepath.WalkDir` で探して `os.ReadFile` していた。各プラグインが SDK 以前の走査コードを約 200 行ずつ写しで持ち、差分判定は `path:mtime:size` の連結署名だった。利用者は毎回 ZIP を解凍して JSON だけを置き直す必要があり、ChatGPT の ZIP を丸ごと展開すると添付ファイルまで二重に置くことになる。fitbit / 位置情報 / archived_git_commit_log の3本は既に `sdk.OpenSources` で ZIP を展開せず中央ディレクトリだけを読んでいた（ADR-0303 / ADR-0309）。

## Decision

- **ZIP だけを読む。展開済みの JSON は読まない。** 走査は `sdk.OpenSources` に任せ、受理するエントリはベース名が `conversations-NNN.json` か `conversations.json` のものだけ
- **差分の署名はエントリの `Path:CRC32:Size`** を並べたもの。変わったら全会話を読み直す（キャッシュのスキーマと表は変えない）
- **アーカイブ単位**で `conversations-NNN.json` を優先し、その ZIP に無ければ `conversations.json` を読む
- **同じ会話 ID が複数の ZIP にあれば `update_time`（Claude.ai は `updated_at`）が新しい版を1つ採り**、採らなかった版のメッセージは持ち越さない。片方にしか無い会話は残る
- ZIP ではないものを指した指定（展開済みフォルダ・`conversations*.json` の直接指定）は `source_problems` として設定画面に出す。プラグインフォルダ自身の `manifest.json` / `config.json` はこの警告の対象にしない

## Rejected alternatives

- **展開済みの JSON も従来どおり読む（ZIP と併読）** — 同じフォルダに ZIP と展開済み JSON が並ぶ（実物がそう）と同じ会話を二重に読む。会話 ID の upsert で件数は増えないが、読む量が倍になり、どちらの書き出しか判別できない。fitbit / 位置情報が「展開済みフォルダは読まない」と決めた理由（ADR-0303）と同じ
- **ZIP を優先し、無ければ展開済み JSON を読む** — 移行期は動くが、ZIP を置き忘れたフォルダが**黙って古い JSON を読み続ける**。「ZIP が見つかりません」と設定画面に出るほうが早く気づける
- **エントリ単位の `file_cache` で増分取り込みにする（fitbit / 位置情報の形）** — 会話は ID が主キーで、エクスポートは `conversations*.json` が丸ごと入れ替わる。エントリが変わったときにその中のどの会話が変わったかは読まないと分からず、結局全部読む。全体再構築は実データ（ChatGPT 100MB / Claude.ai 203MB）で数十秒なので、増分にする理由が無い
- **署名に更新時刻を使い続ける** — Claude.ai の ZIP はエントリの更新時刻が 1980-01-01 固定で、**中身が変わっても動かない**。ZIP ファイル自体の mtime も、同期ツールが保存しない環境がある
- **配列を丸ごと `json.Unmarshal` する（従来の形）** — `json.Decoder` も値1つを内部バッファに全部載せてからデコードするので、203MB の生バイトが構造体の上に余計に載る。要素ごとに `Decode` すれば1会話ぶんで済む
- **同じ会話が複数 ZIP にあるときメッセージを和集合にする** — 古い書き出しで削除されたメッセージが残り続ける。書き出しは「その時点の会話全体」なので、新しい版をそのまま採るのが書き出しの意味に合う
- **`gkill_plugin_claudecode` の `source.go` も同時に SDK 化する** — claudecode は JSONL のフォルダを読むので ZIP の話ではない。今回は触らない

## Consequences

- 旧配置（展開済み JSON だけのフォルダ、`.../conversations*.json` のファイル指定）は**取り込みが止まる**。既存キャッシュの表示は残り、設定画面の「走査で見つかった問題」に「ZIP ではないので読みません」が出る。黙って0件にはならない
- 署名の形式が変わるので、配布後の初回は全会話の再構築が1回走る（`gen` 掃除で古い行は消える）
- `source_dirs` が空のときの既定（プラグインフォルダ自身）は据え置き。ZIP をプラグインフォルダに置けば読める
- 会話の畳み方（update_time が新しい版）はテストで守る。守らないと同じ会話が2つの ZIP にあるとき、ZIP 名の順で古い版が勝つことがある

## Evidence

- 実物の ChatGPT ZIP: 数百エントリ、会話ファイル数本（数MB〜数十MB）、`chat.html` 数十MB。ZIP 内の `conversations-*.json` は展開済みファイルと byte 一致
- 実物の Claude.ai ZIP: 1 エントリ（`conversations.json` 数百MB）、エントリの更新時刻 1980-01-01
- 実データで ZIP 直読みと旧配置（展開済み JSON）のキャッシュを突き合わせ、`conv_cache` / `msg_cache` の件数が一致（2026-09-18。ChatGPT・Claude.ai とも）

## Related tests

- `src/plugins/gkill_plugin_chatgpt/cache_test.go`
- `src/plugins/gkill_plugin_claudeai/cache_test.go`
- `src/server/gkill/plugin/sdk/source_test.go`
