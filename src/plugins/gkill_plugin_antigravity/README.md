# gkill_plugin_antigravity

Google Antigravity のチャットログを gkill タイムラインに表示するプラグイン。
**自分の発言で1件、それに対する一連の応答で1件**として扱う。
ChatGPT / Claude.ai / Claude Code / Codex プラグインと同じく、タイムラインに「あなた」と「応答」が交互に並ぶ。

応答はツール呼び出しを挟んで何度も分かれるが、**次に自分が発言するまでをまとめて1件**にする。
応答は**最後のテキストを本文**にし、それより前のテキスト・thinking・ツール呼び出し・システム発の注記は
「作業の経過」1つに畳む（Codex / Claude Code と同じ組み方。1会話で数百回に及ぶツール呼び出しを並べると会話が読めないため）。
描画は SDK の `chatview` が担い、ほかの AI チャット系プラグインと見た目を揃えている
（[ADR-0315](../../../documents/adr/0315-chat-plugins-render-through-sdk-chatview.md)）。

読む場所と会話IDの採り方は [ADR-0320](../../../documents/adr/0320-antigravity-read-brain-transcript-and-id-from-uuid-dir.md)。

## セットアップ

### 1. ビルド

```bash
cd src/plugins/gkill_plugin_antigravity
go build -o gkill_plugin_antigravity .
# Windows の場合
go build -o gkill_plugin_antigravity.exe .
```

### 2. 配置

```
$GKILL_HOME/plugins/{userID}/gkill_plugin_antigravity/
├── manifest.json              # このディレクトリの manifest.json をコピー
├── gkill_plugin_antigravity   # ビルドしたバイナリ（.exe は自動補完）
└── config.json                # データソースのフォルダ指定（初回起動時に自動生成。次項参照）
```

キャッシュは gkill のキャッシュディレクトリに作られる。

```
$GKILL_HOME/caches/plugin_cache/{userID}/gkill_plugin_antigravity/cache.db
```

`manifest.json` はバイナリに埋め込まれているので、配置先で吐かせることもできる。
配布スクリプトはバイナリだけを配り、`manifest.json` が無ければこれで生成する。バイナリと必ず一致する。
既定の `config.json` も同様に吐ける（通常は起動時に自動生成されるので不要）。

```bash
./gkill_plugin_antigravity --gkill-print-manifest > manifest.json
./gkill_plugin_antigravity --gkill-print-config   > config.json
```

### 3. データソースのフォルダを指定する

プラグインの初回起動時に、`manifest.json` と同じフォルダに `config.json` が
**自動生成される**。これを編集してデータソースを指定する。既存ファイルは上書きされない。
既定値は `~/.gemini/antigravity` なので、Antigravity を入れた端末で gkill を動かすなら触らなくてよい。

```json
{
  "source_dirs": [
    "~/.gemini/antigravity/brain",
    "~/.gemini/antigravity/conversation_summaries.db"
  ]
}
```

| キー | 意味 |
|---|---|
| `source_dirs` | 走査対象。配列でも改行区切りの文字列でもよい。`*` `**` `?` `[]` のワイルドカード、先頭の `~`、環境変数（`$HOME` など）が使える。空にすると `~/.gemini/antigravity` を見る |

- フォルダを指定すると再帰的に走査して `transcript_full.jsonl` と `conversation_summaries.db` を探す
- `conversation_summaries.db` も指定しておくと、会話タイトルとプロジェクト名が出る（無くても動く）
- 変更は**次の検索から反映される**（gkill の再起動は不要）
- `$HOME` は**リテラルで書いてよい**。プラグイン側が `os.ExpandEnv` する。
  ただし gkill を Windows サービスで動かしている場合は実行アカウントのホームになるので、
  確実に指定したいなら絶対パスにする

複数の端末で Antigravity を使っているなら、各端末のログを dvnf で1か所へ集めたフォルダを指す
（例: `~/Kyou/Antigravity_*`）。**会話IDのフォルダはそのまま保つこと**（後述）。

## 取り込む内容

読むのは `brain/<会話ID>/.system_generated/logs/transcript_full.jsonl` だけ。1行が1ステップで、
`step_index` の昇順に並んでいる。

| レコードの `type` | Kyou での扱い |
|---|---|
| `USER_INPUT` | 人間の発言 = 1 Kyou（`source` は `USER_EXPLICIT`） |
| `PLANNER_RESPONSE` | 応答。`content` が空でない最後のものが本文、それより前は作業の経過。`thinking` は思考、`tool_calls` はツール呼び出し（`toolSummary` があればそれを、無ければパスやコマンドを200文字に要約）、`model` はチップ |
| `SYSTEM_MESSAGE` / `CHECKPOINT` / `CONVERSATION_HISTORY` / `EPHEMERAL_MESSAGE` / `ERROR_MESSAGE` | 1行の注記（200文字で切る）。同じ本文の注記は1件に畳む |
| 上記以外（知らない種別） | 種別の名前を見出しにした注記。**捨てない**（捨てるとログの形が変わった日に会話が静かに消える） |

**保持しないもの**: ツールの実行結果（`GENERIC` / `VIEW_FILE` / `LIST_DIRECTORY` / `GREP_SEARCH` /
`RUN_COMMAND` / `SEARCH_WEB` / `CODE_ACTION` / `GENERATE_IMAGE` の `content`）。
これらは `PLANNER_RESPONSE.tool_calls` の件数とほぼ1対1で続くツールの出力で、
ファイル閲覧1件で KB 級から数十 KB になる（Codex プラグインと同じ方針）。`error` があるときだけ1行の注記にする。
添付（`media` の `uri`）も端末のパスなので持たない。

**読まないもの**: 省略版の `transcript.jsonl` と `logs/chunks/` 配下の写し。
どちらも同じ会話の別の形なので、読むと会話が二重になる。種別はファイル名だけで決めているので自動的に外れる。

### 会話IDは「直近の UUID フォルダ名」から採る

`transcript_full.jsonl` のレコードには**会話IDがどのキーにも入っていない**。ファイル名も全会話で同じなので、
会話IDはフォルダ名から採るしかない。そこで「そのファイルから見て直近の、UUID の形をした祖先フォルダ名」を会話IDにする。
1つの規則で2つの置き方の両方に効く。

```
<ホーム>/.gemini/antigravity/brain/<会話ID>/.system_generated/logs/transcript_full.jsonl   生の置き場
<ホーム>/Kyou/Antigravity_<端末>_<日付>/<会話ID>/transcript_full.jsonl                      同期先
```

UUID の祖先フォルダが無いファイルは**読まずに設定画面の警告へ出す**。
パスから作った代用IDを振ると、フォルダ名が変わっただけで別の会話として二重に出てしまう。

同じ会話IDのファイルが複数見つかったら（生の置き場と同期先の両方を指定したとき）、
**一番大きいファイルだけ**を読む。ログは追記のみなので一番大きいものが最新
（`gkill_plugin_vrchat` の「同じ起動のログが2か所」と同じ規則）。

### 会話タイトルとプロジェクト名

`transcript_full.jsonl` には入っていない。`conversation_summaries.db`（SQLite）の
`conversation_summaries` テーブルから `title` と `workspace_uris` を読む。

- `workspace_uris` は `["file:///c:/path/to/repo"]` の形。開いているフォルダの名前だけをチップに出す（絶対パスは出さない）
- 本家は WAL を使っているので、**同期で写したファイルには直近の数件が載っていないことがある**。
  タイトルが取れない会話は**最初の人間の発言の冒頭**で代用する
- 読めなくても取り込みは続ける（Codex の `session_index.jsonl` と同じ「あれば使う任意の索引」扱い）
- タイトルとプロジェクト名は `search_text` に焼き込まず `kyou_cache` の別カラムで持つので、
  あとからタイトルが付いても会話を読み直さない。ワード検索では照合のときに連結する

## 動作の要点

### Kyou の ID は合成する

レコードに id が無いので、`UUIDv5(名前空間, "<会話ID>/<発言者>/<その単位の先頭レコードの step_index>")` で作る。

- `step_index` は Antigravity が振る追記のみの通番なので、会話が伸びても**既存の Kyou の ID が動かない**
- 発言者を種に混ぜているので、人間の発言と応答の ID が衝突しない
- ファイルの置き場所を材料にしていないので、生の置き場と同期先のどちらから読んでも同じ ID になる

**名前空間（`uuid.go` の `antigravityNamespace`）は永久に変えないこと。**
変えると全 Kyou の ID が作り直され、gkill 側で付けたタグやテキストの紐付けが全部切れる。

### 最初の発言より前のステップを捨てない

会話の先頭には `CONVERSATION_HISTORY` や `SYSTEM_MESSAGE` が並ぶことがある。これは
**最初の応答の経過の先頭**へ入れる。`USER_INPUT` が1件も無い会話（自動で始まった作業）も応答1件にする ――
0件にはしない。

### 取り込みは常駐ビルダで進める

重い構築は常駐ビルダ + WAL + バッチ commit で行う（[ADR-0305](../../../documents/adr/0305-plugin-background-builder-wal.md)）。
ハンドラは全部「ビルダを起こして、今キャッシュにあるぶんを即返す」ので、**初回の検索が0件なのは仕様**。

- 差分は transcript 単位の `(mtime, size)`。変わった会話だけ Kyou を作り直す
- `conversation_summaries.db` は差分を見ずに毎回読む。SQLite はページ単位で書くので、
  タイトルが変わってもファイルのサイズは動かず、秒精度の mtime も同じ秒なら動かない
- 単独モード `--gkill-build-cache`（`gkill_server generate_plugin_cache` が使う）は
  stdio ループに入らず同期で1周する

### ワード検索

判定は SDK の `Query.MatchText`（gkill 本体と同じ規則）。gkill 本体はプラグインが返した Kyou の本文を
持たないので、**ここが唯一の判定**。対象は本文・経過のテキスト・thinking・ツール名と要約・モデル名
（`search_text`）と、会話タイトル・プロジェクト名。
`FindQuery.IDs` が来ていれば SQL で先に絞る（[ADR-0318](../../../documents/adr/0318-pass-find-query-to-plugins.md)）。

## テスト

```bash
cd src/plugins/gkill_plugin_antigravity && go test ./...
```

`testdata/` は合成の会話2件。落としてはいけないテストは
[../ABOUT_TEST.md](../ABOUT_TEST.md) にまとめてある。

## 関連ドキュメント

- [プラグインシステム全体](../../../documents/reverse/plugin-system.md)
- [プラグイン一覧と配置](../README.md)
- [プラグイン SDK](../../server/gkill/plugin/README.md)
