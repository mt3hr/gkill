---
name: gkill-plugin
description: "gkill プラグイン（src/plugins/ の独立バイナリ・plugin/sdk/・本体側 dao/reps/plugin_*.go）の約束。manifest.json の provides と型別/付随データ、PluginTypedIndex から即答してプラグインへ往復しない読み取り、stdio のスロット直列化とプロセス回収条件、常駐ビルダ+WAL、Google Takeout の zip 直読みと CRC32+Size 差分、本文表示 plugin-html-view.vue の iframe 注入手順を扱う。src/plugins/・src/server/gkill/plugin/・api/gkill_plugin/・dao/reps/plugin_*.go・plugin-html-view.vue を触るとき必読。「プラグインの本文がときどき空白」「プロセスが殺され続ける」「歩数が2倍になる」の調査でも必読。"
---

# プラグインの不変条件

対象: `src/plugins/**` / `src/server/gkill/plugin/**` / `src/server/gkill/api/gkill_plugin/**` / `src/server/gkill/dao/reps/plugin_*.go` / `src/client/pages/views/plugin-html-view.vue`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

## プラグイン一覧と実測仕様

```
  plugins/    # Standalone plugin binaries (each has its own go.mod, manifest.json, executable)
              #   examples/gkill_example/  — sample plugin (fixed Kyou response)
              #   gkill_plugin_chatgpt/    — ChatGPT conversation history plugin
              #   gkill_plugin_claudeai/   — Claude.ai conversation history plugin
              #   gkill_plugin_claudecode/ — Claude Code chat log plugin (one Kyou per human
              #                              message + one per its whole response run,
              #                              source folder configurable, SQLite differential cache)
              #   gkill_plugin_codex/      — OpenAI Codex CLI rollout logs, same Kyou granularity as
              #                              claudecode. Thread id is the FILENAME uuid, never
              #                              `session_meta.session_id` (absent in 23/52 real files and,
              #                              on a sub-agent, holds the PARENT's id). `session_meta`
              #                              repeats 1..13× per file: take identity from the FIRST
              #                              occurrence only (a sub-agent file carries the parent's
              #                              meta as its second) but merge cwd/git/originator across
              #                              all. Conversation comes from the `event_msg` lane only
              #                              (`response_item/message` mixes in injected preambles);
              #                              tool *calls* come from `response_item`, tool *results*
              #                              are never stored — they are 94.7% of the bytes. Kyou IDs
              #                              are UUIDv5 of (thread id, role, ordinal) since event_msg
              #                              records have no id; append-only logs keep them stable.
              #                              One real line is 19.9MB, so `reader.go` classifies from
              #                              the first 512 bytes and drains unwanted lines without
              #                              buffering. Background builder + WAL (4.5s first build /
              #                              <1s incremental on 245MB, 52 files → 301 Kyou)
              #   gkill_plugin_fitbit/     — Google Takeout Fitbit/Google Health, aggregated to
              #                              one KC per (day, metric). 34 metrics, background
              #                              build (~155s first / <1s incremental on a 271MB zip)
              #   gkill_plugin_google_locationhistory/ — Google Takeout location history as
              #                              GPSLog (no Kyou, `emits_kyou: false` so it stays out
              #                              of the rep list). Format detected by content, not path
              # The two Takeout plugins read the export **as a zip, never extracted** — scan lives
              # in `plugin/sdk/source.go` (`sdk.OpenSources`), shared by both. Differential test is
              # `(CRC32, Size)`, NOT mtime: Takeout stamps every entry with the same export time, so
              # mtime never moves when content changes. One export = "dir + the takeout timestamp in
              # the zip name", so split `-1-001`/`-1-002` parts merge but a second export dropped in
              # the same folder does not (folder alone would double every summed metric). fitbit
              # keeps an `export` table and folds only the lowest-rank export per (metric, day);
              # locationhistory needs no ranking because its read-time `SELECT DISTINCT (time, lat,
              # lng)` already collapses points across exports.
              # The six shippable plugins (chatgpt / claudeai / claudecode / codex / fitbit /
              # locationhistory — gkill_example is
              # excluded) take `source_dirs` in config.json (folders/globs), auto-create
              # that config.json next to manifest.json on first start (existing files are never
              # overwritten, via sdk.EnsureConfig + Handler.DefaultConfig), and can print their
              # embedded manifest.json / default config.json via --gkill-print-manifest /
              # --gkill-print-config. Their SQLite caches live under gkill's own cache dir
              # ($GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db — resolved in
              # plugin/sdk/cache_path.go (sdk.CacheDBPath) from the inherited GKILL_HOME env
              # var, falling back to the plugin folder), so `clear_cache plugin` can wipe them.
              # The 6 plugins had a byte-identical cache_path.go each; it now lives in the SDK
```

- `gkill/api/gkill_plugin/` — Plugin protocol types: `PluginManifest`, `PluginRequest`, `PluginResponse`, `PluginKyou`, `PluginTypedData`, `PluginGPSLog` (stdio newline-delimited JSON)
- `gkill/plugin/sdk/` — Plugin author SDK. `sdk.Run(sdk.Handler{...})` starts the stdio JSON message loop. `Handler` has 8 fields: `FindKyous` (required), `GetKyou`, `GetContentHTML`, `GetConfigHTML`, `PostConfig`, `GetGPSLogs`, `RepName`, `DefaultConfig`. Plugins are standalone binaries in `src/plugins/`

**プラグインの型別/付随データ:** `manifest.json` の `provides`（既定は空＝従来どおり）に種別を書くと、そのプラグインの記録が **native と同じ型別リポジトリに載る**。`kc` を宣言して `data_type: "kc"` を返せば `typed_kc` が埋まり Dnote の推移グラフで集計できる。`tag` を宣言すればタグ一覧（`get_all_tag_names`）に載るので、rykv の既定の絞り込み「タグ無し」から漏れる問題が起きない。`gpslog` は Kyou ではないので専用コマンド `get_gps_logs`（ページング必須）で受け渡す。

アダプタ（`dao/reps/plugin_typed_adapters.go` / `plugin_attached_adapters.go` / `gps_log_repository_plugin_impl.go`）の**読み取りは決してプラグインへ往復しない**。`PluginTypedIndex`（`plugin_typed_index.go`）が `find_kyous` 1回ぶんの不変スナップショットを持ち、そこから即答する。プラグイン呼び出しは容量1のスロットで直列化されるので、1件ずつ聞きに行くと一覧の行数ぶんの直列 stdio 呼び出しになりプロセスが殺され続ける。`GetLatestDataRepositoryAddress` は**型別は空・付随は実データ**（型別が返すと `UpdateTime` の揺れでレコードごと消え、付随が返さないと `--cache_in_memory=false` でタグが全部落ちる）。アダプタの `Close` は no-op（プロセスを閉じるのは本体と `PluginManager.CloseAll` だけ）。登録は `gkill_dao_manager.go` の `KCReps`→`Reps` コピーループより**後**（先だと二重検索になる）。詳細は `documents/reverse/plugin-system.md` の14章、却下案は [ADR-0021](../../../documents/adr/0021-plugin-provides-typed-index.md)。

本体側の受け口（`gkill/dao/reps/`）のプロセス管理:

`plugin_repository_impl.go` manages plugin subprocess lifecycle (start, slot-guarded stdio, one persistent reader goroutine per process, response-ID matching, auto-restart on crash). **呼び出し元のキャンセル（HTTPクライアントの切断）ではプロセスを回収しない**（フロントは全リクエストに `AbortController` を張っているため）。回収するのは gkill 自身のデッドライン超過時だけ。直列化は mutex ではなく容量1のチャネル（`callSlot`）で行い、**期限はスロットを取ってから張る**。順番待ちの上限は別枠（`maxPluginQueueWait` 既定10秒）で、待ちきれなければ `ErrPluginBusy` を返すだけでプロセスには手を出さない。プラグイン rep は `Repositories` の fan-out でスレッドプールのスロットを取らない（`goForRep`）ので、プラグインのロック待ちで検索全体が止まることはない

**プラグインの重い構築は常駐ビルダ + WAL + バッチcommit**（2026-08-21、監査 M-6）。claudecode も codex/fitbit と同じく `builder.go`（`EnsureStarted`/`Kick`/`loop`、mu(DB初期化)/buildMu(構築)分離で読み取り無待機、WAL 自前DSN、`cache_meta` に進捗）へ移行済み。`GetMessages`/`GetMessage`/`GetStats` は refresh を呼ばず現キャッシュ即答+Kick。同期・単一tx構築（デッドラインkill→進捗ゼロループ）を新規に書かないこと。 却下案（同期構築／ロック共有／デッドライン延長）と実測は [ADR-0024](../../../documents/adr/0024-plugin-background-builder-wal.md)。

**プラグイン本文（`plugin-html-view.vue`）の iframe 越しの受け渡し**（2026-08-15 に「Ryuu の PluginKyou がときどき空白」「本文をダブルクリックしても KyouDialog が開かない」を潰して確立）。本文の入れ方は `is_list_view = typeof props.height === 'number'` で分岐し、一覧は `srcdoc` 直書き、Ryuu(`'fit-content'`)・rykv 詳細ペイン(`'auto'`)・KyouDialog(`'unset'`) は定数ローダーへの postMessage 注入（ダイアログの `pushState` より後に iframe がナビゲートするとブラウザバックが2回要るため、ローダー方式は外せない）。注入は**ローダーが `gkill_plugin_loader_ready` を名乗ってから**行う ―― `iframe.contentWindow` は about:blank の時点から真なので、それを見て先に送るとリスナー未登録の iframe に届いて黙って消え、`sent_html` のせいで送り直さないぶん本文が二度と入らない（ServiceWorker が `get_plugin_content_html` をキャッシュ優先で返すので、2回目以降の表示ほど負けやすい＝「ときどき」）。ready のたびに `sent_html` を落として送り直し、Kyou が変わったら iframe ごと作り直す（`:key`。`document.open()` はローダーが `window` に張ったリスナーごと捨てるので使い回すと2件目が入らない）。**`@load` で `sent_html` を落としてはいけない**（`document.close()` でも `load` は発火しうるので注入ループになる）。iframe 内のダブルクリックは親のDOMへ伝播しないので、本文HTMLの末尾に転送スクリプトを足して `gkill_iframe_dblclick` を受け、**親が本物の `dblclick` を撃ち直す**（新しい emit 経路を作らないので `kyou-view.vue` と `ryuu-item-view.vue` の既存の `@dblclick` がそのまま拾う）。テーマ通知を最初のサイズ通知だけに限るのは、本文側がテーマ受信で測り直して送り返すため（毎回返すと10ms周期のピンポンになる）。`plugin-html-view.test.ts` が守る。

## 関連スキル

- [gkill-go-backend](../gkill-go-backend/SKILL.md) — `len(XxxReps) == 1` 判定禁止（provides プラグイン1つで長さが2になる）と検索フィルタ
- [gkill-mcp](../gkill-mcp/SKILL.md) — MCP からのプラグイン本文取得（`include_plugin_content`、同一プラグインへ並列に投げない）
- [gkill-client-foundation](../gkill-client-foundation/SKILL.md) — クライアント全域の規約（plugin-html-view の周辺）

## 詳しい設計と却下案（ADR）

- [ADR-0020 キャンセルとプロセス回収の切り分け](../../../documents/adr/0020-plugin-cancel-vs-kill.md)
- [ADR-0021 provides と PluginTypedIndex](../../../documents/adr/0021-plugin-provides-typed-index.md)
- [ADR-0022 差分判定は CRC32 と Size](../../../documents/adr/0022-plugin-cache-use-crc32-and-size.md)
- [ADR-0023 GPSLog 専用プラグインは Rep に出さない](../../../documents/adr/0023-plugin-emits-kyou-false.md)
- [ADR-0024 常駐ビルダと WAL](../../../documents/adr/0024-plugin-background-builder-wal.md)
- [ADR-0025 Codex のスレッドIDはファイル名から](../../../documents/adr/0025-codex-thread-id-from-filename.md)
