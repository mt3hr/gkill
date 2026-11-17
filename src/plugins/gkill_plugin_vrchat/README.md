# gkill_plugin_vrchat

VRChat の出力ログ（`output_log_*.txt`）から、**どのワールドに・誰と・どれぐらい・何をしていたか**を gkill の記録として表示するプラグイン。

1本で TimeIs・URLog・Kmemo を返し、gkill は native の記録と同じ型別ビューで描く（本文 HTML は持たない）。

| 何を | gkill の記録 | タグ |
| --- | --- | --- |
| ワールドへの滞在（入室〜退室） | TimeIs。タイトルはワールド名 | `VRChat_world` |
| そのワールドの情報（滞在ごとに1件、入室の時刻） | URLog。URL は VRChat 公式サイトの起動用ページ（`worldId` と `instanceId` 付き）、説明にワールド ID とインスタンス | `VRChat_world` |
| 同じ部屋にいた人（その人の入室〜退室） | TimeIs。タイトルは `名前（usr_…）` | `VRChat_room` |
| 自分のアバターの切り替え・動画プレイヤーの URL | Kmemo。`アバター変更: …` / `動画: …` | `VRChat_event` |

VRChat が終了したら（落ちたときはログの最後の行で）、その時点で開いている TimeIs を全部閉じる。

TimeIs は native と同じく、開始の Kyou（`timeis_start`、時刻は開始）を返し、検索条件の `include_end_timeis` が
true のときは終了の Kyou（`timeis_end`、時刻は終了）も返す。gkill の画面は既定で true なので、
滞在や同じ部屋の人はタイムラインの開始と終了の2か所に出る（gkill 1.1.12 以降。[ADR-0318](../../../documents/adr/0318-pass-find-query-to-plugins.md)）。

1本のプラグインが複数の型別の種類を返すのはこのプラグインが最初で、gkill 1.1.12 以降が要る
（それより前の gkill はプラグインの TimeIs を「その時刻に実行中か」で絞らず、全件が実行中に見える。[ADR-0317](../../../documents/adr/0317-plugin-mixed-typed-kinds-and-playing-time.md)）。

## セットアップ

### 1. ビルド

```bash
cd src/plugins/gkill_plugin_vrchat
go build .

# Windows 向けにクロスコンパイルする場合
GOOS=windows GOARCH=amd64 go build -o gkill_plugin_vrchat.exe .
```

### 2. 配置

```
$GKILL_HOME/plugins/{userID}/gkill_plugin_vrchat/
    manifest.json           # 必須。バイナリに埋め込んであるので --gkill-print-manifest で出せる
    gkill_plugin_vrchat.exe # Windows
    config.json             # 初回起動時に自動生成される。既存のものは上書きされない
```

ディレクトリ名・`manifest.json` の `name`・`executable` は**すべて `gkill_plugin_vrchat`** で一致させること。
`manifest.json` と既定の `config.json` は `--gkill-print-manifest` / `--gkill-print-config` でバイナリ自身から出せる。

> **PowerShell 5.1 では `>` を使わないこと。** UTF-16LE で書かれ、壊れた `manifest.json` は
> `plugin_manager.go` が**無言で読み飛ばす**（詳しくは `gkill_plugin_codex` の README）。

置いたら gkill を再起動する。

### 3. データソースのフォルダを指定する

`config.json` の `source_dirs` に、出力ログが入ったフォルダかファイルを書く。設定画面からも編集できる。
既定は VRChat の書き出し先（`~/AppData/LocalLow/VRChat/VRChat`）。

```json
{
  "source_dirs": [
    "~/AppData/LocalLow/VRChat/VRChat",
    "$HOME/Kyou/VRChat_*"
  ]
}
```

**VRChat は古いログを数日で消す。** 消える前に同期スクリプトなどで別の場所へ運び、その場所も書いておくこと。
同じ起動のログが複数の場所にあっても（元のファイルと運んだコピー、プレイ中に運んだ短い版とあとの完全な版）、
1回の起動は1回ぶんとして数える（下の「同じ起動のログが複数あるとき」）。

## 動作の要点

### 拾う行

各行は `yyyy.MM.dd HH:mm:ss <レベル> - <本文>`。時刻の無い行（例外のスタックトレースなどの継続行）は読まない。
時刻はタイムゾーンを持たないので、プラグインが動く PC のローカル時刻として読む。

| 本文 | 使い道 |
| --- | --- |
| `[Behaviour] Entering Room: <ワールド名>` | 滞在の開始 |
| `[Behaviour] Joining wrld_<ID>:<インスタンス>` | 滞在のワールド ID とインスタンス |
| `[Behaviour] OnLeftRoom` | 滞在の終了（その部屋の人も全員終了） |
| `[Behaviour] OnPlayerJoined <名前> (usr_<ID>)` / `OnPlayerLeft …` | 人の出入り |
| `[Behaviour] Initialized PlayerAPI "<名前>" is local` | 自分の名前（同部屋から外し、アバターの切り替えを自分のものだけにする） |
| `[Behaviour] Switching <名前> to avatar <アバター名>` | アバターの切り替え |
| `[Video Playback] Attempting to resolve URL '<URL>'` | 動画の URL（続けて同じ URL は1件） |
| `VRCApplication: HandleApplicationQuit` | 終了 |

今の VRChat は `Entering Room`（ワールド名）を先に、`Joining wrld_`（ID）を後に書く。古い書式は逆順なので、
開いている滞在にまだ ID が無ければそこへ付け、あれば次の部屋の ID として控える。
自分の名前の行は自分の `OnPlayerJoined` より後に出るので、ファイルを読み終えてから記録を作る。

### 記録はログの中身だけで決まる

1ファイル = 1回の起動で、記録はそのファイルの中で閉じる。終了の行が無いときは最後の時刻付きの行で閉じるので、
プレイ中に運んだログは最後の行で仮に閉じ、次の同期で伸びる。TimeIs の更新時刻は終了時刻にしてある（伸びると更新時刻も動く）。
実行中（終了なし）の TimeIs は出さない。画面の「終了」ボタンで native の版が書かれてしまうため。

### 検索条件

期間とワードのほか、gkill から届く検索条件の全体（`sdk.Query.FindQuery`。ADR-0318）のうち次を見る。

- `include_end_timeis`: true なら TimeIs の終了の Kyou も返す。開始の Kyou は開始時刻、終了の Kyou は終了時刻で期間を判定する。
  1件の TimeIs から Kyou が2つできうるので、LIMIT は新しい順に並べてから掛ける。終了の Kyou は ID・型別データ・タグ・版が開始と同じ。
- `ids`: 指定の ID の行だけを読む（キャッシュの SQL に `IN` を足す）。

### Kyou の ID

`uuidV5(固定の名前空間, gkill_plugin_vrchat|<起動キー>|<種類>|<…>)`。起動キーはファイルの最初の時刻付きの行の時刻で、
ファイル名や置き場所が変わっても同じ起動なら同じ値になる。種類（world_timeis / world_urlog / player / event）を材料に入れるので、
種類をまたいで ID が重ならない（gkill の索引は同じ ID の Kyou を1件にまとめてしまう）。
残りの材料は追記で変わらないもの（滞在の通し番号・人ごとの入室の回数・出来事の通し番号）にしてあり、
プレイ中に運んだ短い版とあとの完全な版とで、両方にある記録の ID は同じになる。

### 同じ起動のログが複数あるとき

キャッシュは起動キーごとに、その起動のファイルのうち**一番大きいもの**から記録を作る。ログは追記しかされないので、大きいほうが新しい。
一番大きいファイルが消えたら、残ったファイルから作り直す。

### 取り込みはバックグラウンド

`gkill_plugin_codex` と同じく常駐ビルダ + WAL（[ADR-0305](../../../documents/adr/0305-plugin-background-builder-wal.md)）。
差分は `(mtime, size)` で見て、変わったファイルとそれが属する起動だけを作り直す。
`gkill_server generate_plugin_cache` からの単独モード（`--gkill-build-cache`）では同じ構築を同期で1周する。

## プラグイン情報

| 項目 | 値 |
| --- | --- |
| `name` / `executable` | `gkill_plugin_vrchat` |
| `rep_name` | `VRChat` |
| `data_type`（manifest） | `timeis_start`（代表の種類。各 Kyou は `timeis_start` / `urlog` / `kmemo`） |
| `provides` | `timeis` / `urlog` / `kmemo` / `tag` |
| キャッシュ | `$GKILL_HOME/caches/plugin_cache/{userID}/gkill_plugin_vrchat/cache.db` |

## ファイル構成

| ファイル | 役割 |
| --- | --- |
| `main.go` | `sdk.Handler` の配線、`kyouOf`（行 → Kyou と型別データ）、ワード検索 |
| `types.go` | 定数（rep 名・data_type・タグ・記録の種類）、`record`、URLog の URL の組み立て |
| `parse.go` | 出力ログ1つ → 出来事の列（長すぎる行は読み捨てて数える） |
| `records.go` | 出来事の列 → 記録（滞在・同部屋・出来事、ID の導出） |
| `scan.go` | 指定の展開と走査（stat だけ） |
| `cache.go` | SQLite キャッシュ、差分の取り込み、起動キーごとの作り直し |
| `query.go` | 一覧・1件・設定画面の統計（ファイルを開かない） |
| `builder.go` | 常駐ビルダ |
| `html.go` | 設定画面（`sdk/configview`） |
| `config.go` | `config.json` の既定と解釈 |
| `uuid.go` | UUIDv5 |

## テスト

```bash
cd src/plugins/gkill_plugin_vrchat
go test ./...
```

`testdata/fixtures/` のログはすべて合成値（ワールド名・人の名前・usr / wrld ID・時刻は作った値）。実際のログから行を写さないこと。
フォルダ名を `logs` にしたり拡張子を `.log` にしたりしないこと（gkill の `.gitignore` に当たってコミットされず、きれいな作業ツリーでテストが落ちる）。

## 関連資料

- [ADR-0317](../../../documents/adr/0317-plugin-mixed-typed-kinds-and-playing-time.md) — 1本のプラグインが複数の型別の種類を返す・プラグインの TimeIs も playing_time で絞る
- [plugin-system.md](../../../documents/reverse/plugin-system.md) — プラグインの仕組み全体
