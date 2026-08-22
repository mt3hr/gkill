# ADR-0024: プラグインの重い構築は常駐ビルダ + WAL + バッチcommit

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-21 |
| Sources | `3c3bfade` / `bb364253`（監査 M-6） / `documents/reverse/plugin-system.md`「6. SQLite3 キャッシュ」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/plugins/gkill_plugin_codex/builder.go` |

## Context

プラグインのハンドラは**数十msで返す必要がある**。gkill 側の実行デッドラインは30秒、`IsAlive` は5秒（→ ADR-0020）。

ところが取り込みは重い。実データで Fitbit の Takeout（271MB）が初回155秒、Codex のロールアウトログ（245MB / 52ファイル）が4.5秒。同期でフル構築すると**デッドラインで殺され、次回また最初からやり直す進捗ゼロループ**に入る。プラグインは永久に使えるようにならない。

さらに、構築と読み取りを**同じ mutex** で直列化していたため、初回構築のあいだ `find_kyous` が全部詰まっていた。

## Decision

重い構築は**常駐ビルダ**へ移す。`builder.go` の `EnsureStarted` / `Kick` / `loop` の形で、

- **構築ロックと読み取りロックを分ける**（`mu` はDB初期化、`buildMu` は構築）。読み取りは無待機
- **WAL を自前DSNで有効化**し、**バッチcommit** で進める
- 進捗を `cache_meta` に持つ
- `GetMessages` / `GetMessage` / `GetStats` は refresh を呼ばず、**現キャッシュから即答して `Kick` するだけ**

読み取りは「そこまで取り込めたぶん」を返す。

## Rejected alternatives

- **同期・単一トランザクションで構築する** — デッドライン kill → 次回また最初から、の**進捗ゼロループ**になる。1回も完了しないので、どれだけ待っても使えるようにならない。

- **構築と読み取りで mutex を共有する** — 初回構築（実データで155秒）のあいだ全ハンドラが詰まり、結局デッドラインで殺される。ロックを分けることが、進捗を保つことと同じくらい効く。

- **デッドラインを延ばす（30秒 → 数分）** — 画面の1操作が数分固まる。しかもデッドラインは「詰まったプラグインを回収する」ための仕組みなので（→ ADR-0020）、延ばすと本当に詰まったときの回収が効かなくなる。

- **ハンドラの中で少しずつ進める（呼ばれるたびに1歩）** — 呼ばれない限り進まない。利用者が画面を閉じている間に取り込んでおく、ができない。

- **キャッシュを WAL にしない** — バッチcommit のたびにジャーナルの作り直しが走る。キャッシュは持ち回らない派生物なので WAL の制約（→ ADR-0013）がかからない。

## Consequences

「refresh を呼ばず現キャッシュ即答 ＋ `Kick`」が全ハンドラの形になる。**新しいハンドラで同期 refresh を書かないこと。**

読み取りが部分状態を返すので、利用者から見ると「取り込み中は件数が増えていく」。これは進捗が見えるぶん、無反応より良いという判断。

対象は claudecode / codex / fitbit / locationhistory / chatgpt / claudeai。claudecode も 2026-08-21 に同じ形へ移行済み。

## Evidence

- Fitbit（Takeout 271MB）: 初回 約155秒 / 増分 1秒未満
- Codex（245MB / 52ファイル → 301 Kyou）: 初回 4.5秒 / 増分 1秒未満
- gkill 側のデッドライン: 実行30秒 / `IsAlive` 5秒 / 順番待ち10秒

## Related tests

- `src/plugins/gkill_plugin_claudecode/cache_test.go`
- `src/plugins/gkill_plugin_codex/cache_test.go`
- `src/plugins/gkill_plugin_fitbit/cache_test.go`
- `src/server/gkill/plugin/sdk/cache_path_test.go`
