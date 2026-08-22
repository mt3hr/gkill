# ADR-0020: プラグインの打ち切りは「待つのをやめる」と「プロセスを殺す」を分け、期限はスロットを取ってから張る

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-05 |
| Sources | `d2733c65` / `d577bba7` / `documents/reverse/plugin-system.md`「5. 並行制御」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/plugin_repository_impl.go` |

## Context

`pluginRepositoryImpl.sendRequest` は `ctx.Done()` でプラグインプロセスを `Process.Kill()` していた。そして全HTTPパスが `r.Context()` を無変換で渡していたため、**HTTPクライアントが切断するとそのユーザのプラグインプロセスが落ちていた**。

稀な事故ではなく通常操作で日常的に起きていた。フロントは全リクエストに `AbortController` を張っており（`gkill-api-request.ts`）、ダッシュボードは再取得のたびに前の `get_kyous` を abort する（`use-dashboard-page.ts`）。編集ビュー12ファイルも `get_kyou` を abort する。つまり**画面の絞り込みを変えるだけでプラグインが落ちていた**。`post_plugin_config` に至っては設定書き込み中に SIGKILL されうる状態だった。

kill していたのには理由があった。リーダー goroutine が `scanner.Scan()` でブロックしていてプロセスを殺す以外に解除できず、かつ `resp.ID` と `req.ID` を突き合わせていないため、放置すると**次のリクエストの応答を横取りしてしまう**。つまり kill は「ID突き合わせが無いこと」の代償だった。

## Decision

トランスポートを作り直し、打ち切りの契機を2つに分ける。

- **呼び出し元のキャンセル** … 待つのをやめるだけ。プロセスには触らない
- **gkill 自身のデッドライン** … 詰まっているので回収する（既定30秒 / `IsAlive` は5秒）

直列化は mutex ではなく容量1のチャネル（`callSlot`）で行い、**期限はスロットを取ってから張る**。順番待ちの上限は別枠（`maxPluginQueueWait` 既定10秒）で、待ちきれなければ `ErrPluginBusy` を返すだけでプロセスには手を出さない。

## Rejected alternatives

- **`ctx.Done()` でプロセスを殺し続ける（改修前）** — 画面操作でプラグインが落ちる。実サーバへの300回の中断リクエストで、旧バイナリはプラグインが6/24/10回再起動した。

- **`r.Context()` をハンドラ側で切り離す（`context.WithoutCancel` 相当）** — 症状は消えるが、本当に詰まったプラグインを回収する手段まで無くなる。「待つのをやめる」と「殺す」は別々に要る。

- **タイミングで判定する（先に Done になったほうを見る）** — 呼び出し元が Deadline 付きだと**両者がほぼ同時に Done になり、`select` がどちらを選ぶか決まらない**。判定はタイミングではなく**エラーの種類**で行う。

- **期限を排他ロックの前に張る（改修前）** — 行列に並んでいるだけで期限を食い潰し、自分の番が来る前に**自分自身を殺していた**。MCP から同一プラグインへ並列に投げると必ず踏む（→ ADR-0051）。

- **`bufio.Scanner` を複数の goroutine から使う** — 並行安全ではない。プロセスごとに常駐リーダーを1本持たせ、`Scanner` に触れるのはリーダーだけにする。リクエスト側はバッファ1の `respCh` 経由で受け取る。

## Consequences

`sendRequest` は `resp.ID` を `req.ID` と突き合わせ、不一致なら読み捨てる（IDが空のものは SDK のパースエラー応答なので自分宛てとして扱う）。**この突き合わせが「殺さなくてよい」ことの根拠**なので、外すと kill に戻すしかなくなる。

`Close` は `readerDone` を待ってから `cmd.Wait()` する（stdout を読んでいる最中の `Wait` は `os/exec` が禁じている）。待つ間は `respCh` を読み捨てないとリーダーが送信でブロックしたままになる。

SDK・既存プラグイン・ハンドラは無変更で済んだ。SDK は元から全コマンドで `ID: req.ID` を返している。

## Evidence

- 実サーバに300回の中断リクエストを投げる A/B: 旧バイナリはプラグインが **6 / 24 / 10 回再起動**、修正版は **0回**。中断後の応答も正しい（ストリームがずれない）

## Related tests

- `src/server/gkill/dao/reps/plugin_repository_impl_test.go`
