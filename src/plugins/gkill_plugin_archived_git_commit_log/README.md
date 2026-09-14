# gkill_plugin_archived_git_commit_log

zip に固めて保管した Git リポジトリのコミットログを gkill のタイムラインに表示するプラグイン。

作業を終えたリポジトリを zip にして IDF リポジトリへ入れておくと、その中の `.git` は誰も読まず、
コミット履歴がタイムラインから抜け落ちる。このプラグインは **zip を展開せずに** 中の `.git` を読み、
コミット1件を稼働中の Git リポジトリ（native の `git_commit_log` rep）と**同じ形**の記録として返す。

- `data_type` は `git_commit_log`、ID はコミットハッシュ、rep 名は `.git` を含むディレクトリ名、
  時刻はコミッタ日時、利用者名は author 名、アプリ名は `git`
- 画面は native と同じ表示（追加行 / 削除行 / メッセージ）で、推移グラフの集計や MCP の
  `git_commit_log` payload にもそのまま載る
- 稼働中のリポジトリにも同じコミットがあれば、検索結果では1件に畳まれる
- 同じリポジトリを別の時期に固めた zip が複数あっても、コミットは1件ずつ。複数の zip に入っている
  コミットの rep 名は**最新のコミットを持つリポジトリ**のもの（改名したプロジェクトの共通の履歴は新しい名前に付く）

## セットアップ

### 1. ビルド

```bash
cd src/plugins/gkill_plugin_archived_git_commit_log
go build .

# Windows 向けにクロスコンパイルする場合
GOOS=windows GOARCH=amd64 go build -o gkill_plugin_archived_git_commit_log.exe .
# Android(Termux) 向け。go-git も純Go なので NDK も WSL も要らない
CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -o gkill_plugin_archived_git_commit_log .
```

### 2. 配置

```
$GKILL_HOME/plugins/{userID}/gkill_plugin_archived_git_commit_log/
    manifest.json                             # 必須。バイナリに埋め込んであるので --gkill-print-manifest で出せる
    gkill_plugin_archived_git_commit_log      # Linux / macOS / Android
    gkill_plugin_archived_git_commit_log.exe  # Windows
    config.json                               # 初回起動時に自動生成される。既存のものは上書きされない
```

ディレクトリ名・`manifest.json` の `name`・`executable` は**すべて `gkill_plugin_archived_git_commit_log`** で一致させること。
`gkill_plugin_` の接頭辞も必須（Termux 側の配布スクリプトが `pkill -KILL -f gkill_plugin_` で
更新前にプロセスを落としているため、接頭辞が無いと古いバイナリを掴んだまま生き残る）。

`manifest.json` と既定の `config.json` はバイナリ自身から出せる。

```bash
./gkill_plugin_archived_git_commit_log --gkill-print-manifest > manifest.json
./gkill_plugin_archived_git_commit_log --gkill-print-config   > config.json
```

> **PowerShell 5.1 では `>` を使わないこと。** UTF-16LE で書かれ、さらにプラグインの UTF-8 出力を
> CP932 として解釈するので二重に壊れる。壊れた `manifest.json` は
> `plugin_manager.go` が**無言で読み飛ばす**ので、プラグインが消えたようにしか見えない。

置いたら gkill を再起動する（リポジトリの構築時にプラグインを探すため）。

### 3. 対象の zip を指定する

`config.json` の `source_dirs` に、zip の実パスか、zip を置いたフォルダを書く。
設定画面（Kyou の右クリック →「プラグイン設定」）からも編集できる。

```json
{
  "source_dirs": [
    "$HOME/Kyou/Box_Laptop_20221210/myrepo.zip",
    "$HOME/Kyou/Box_Desktop_20180826",
    "D:/backup/repositories/*.zip"
  ],
  "max_git_dir_mb": 256
}
```

| キー | 意味 |
|---|---|
| `source_dirs` | 走査対象。配列でも改行区切りの文字列でもよい。`*` `**` `?` `[]` のワイルドカード、先頭の `~`、環境変数（`$HOME` など）が使える。zip の実パスはその zip だけを、フォルダは配下の `*.zip` を再帰的に読む。既定は空 |
| `max_git_dir_mb` | 1リポジトリの `.git` の合計サイズの上限（MB）。超えるものは読まず、設定画面に理由を出す。既定 256 |

- 変更は**次の検索から反映される**（gkill の再起動は不要）
- zip を指定から外すと、そのコミットは次の走査で消える（他の zip にも入っていれば残る）
- `$HOME` は**リテラルで書いてよい**。プラグイン側が `os.ExpandEnv` する。
  ただし gkill を Windows サービスで動かしている場合は実行アカウントのホームになるので、絶対パスが確実

## 読み方

zip は展開しない。中央ディレクトリを列挙して `.git/**` のうち go-git が読むエントリ
（`HEAD` / `config` / `packed-refs` / `shallow` / `refs/**` / `objects/**`）だけをメモリ上のファイルシステム
（go-billy の memfs）へ流し込み、go-git で開いて全 ref からコミットを辿る。hooks / logs / index は読まない。

| zip の中の形 | rep 名 |
|---|---|
| `name/.git/…`（普通） | `name` |
| `kokko/wiki/.git/…`（入れ子） | `wiki` |
| `.git/…`（ルート直下） | zip のファイル名から拡張子を除いたもの |
| 1つの zip に複数 | それぞれ別の rep 名 |

- `.git/HEAD` が無い zip（ソースだけの snapshot）は読み飛ばす。問題としては報告しない
- `git init` 直後（コミット 0 件）の `.git` は 0 件で正常
- loose オブジェクトと packfile のどちらも読める
- 読み直しの要否は、`.git` 配下のエントリの (名前, CRC32, サイズ) から作った指紋で決める。
  アーカイブは基本不変なので、2回目以降は指紋の比較だけで終わる
- 行数（追加 / 削除）は go-git の `StatsContext` で、native と同じ計算（root commit は空ツリーとの差、merge は親1との差）。
  集計に失敗したコミットは行数 0 で記録し、理由を設定画面に出す

読めないもの:

- `.git` がファイル（worktree / submodule の `gitdir:` ポインタ）
- 分割 zip（`.z01` … と `.zip` の組）
- `max_git_dir_mb` を超える `.git`

## キャッシュ

`$GKILL_HOME/caches/plugin_cache/{userID}/gkill_plugin_archived_git_commit_log/cache.db`（SQLite・WAL）。
取り込みはバックグラウンドで進み、置いた直後の1回目の検索は空が返る。進捗は設定画面に出る。
`gkill clear_cache plugin` で消せる。

| 表 | 内容 |
|---|---|
| `repo` | zip の中の1リポジトリ（指紋・コミット数・読めなかった理由） |
| `commit_log` | コミット1件。同じハッシュは1行。rep 名は、そのコミットを含むリポジトリのうち最新のコミットを持つもの（同着なら zip のパス順）で、構築のたびに決め直す |
| `commit_repo` | コミットがどの zip のどのリポジトリに入っていたか |

## 検索

ワード検索の対象はコミットメッセージ・リポジトリ名・author 名。コミットハッシュは前方一致で当たる。
`rep_types` を `git_commit_log` にした検索（型別リポジトリ経由）では native と同じくメッセージだけが対象。

## 開発

```bash
go test ./...
```

testdata に `.git` は置けない（git が入れ子の `.git` を追跡しない）ので、テストのたびに go-git でリポジトリを作り、
`archive/zip` で固めてから読む（`testutil_test.go`）。
