# gkill_plugin_gemini

Gemini の会話履歴を gkill タイムラインに表示するプラグイン。Google Takeout の ZIP を解凍せずにそのまま読み込み、1往復を「自分のプロンプト」と「Gemini の返答」の2件の Kyou にして gkill に統合する。

## セットアップ

### 1. データファイルの準備

**Takeout のトップレベルの「Gemini」には会話が入らない**（Gem と予定アクションの設定だけ）。会話は「マイ アクティビティ」の中にある。

1. [Google Takeout](https://takeout.google.com) で「選択をすべて解除」
2. **「マイ アクティビティ」** にチェックを入れる
3. その行の「すべてのアクティビティ データが含まれます」→「選択をすべて解除」→ **「Gemini アプリ」** だけにチェック
4. 同じ行の **「複数の形式」** →「アクティビティの記録」を HTML から **JSON** に変える
5. 書き出しを作成し、届いた ZIP を**解凍せず**そのまま置く

[Gemini アプリ アクティビティ](https://myactivity.google.com/product/gemini)が無効・一時停止のままだと、書き出しても中身が空になる。

Fitbit・タイムラインも同じ書き出しに入れておけば、1つの ZIP を `gkill_plugin_fitbit` / `gkill_plugin_google_locationhistory` と共用できる（3本とも取り込み元の既定は同じ `~/Kyou/GoogleTakeout_*`）。

**HTML 形式では読まない。** HTML のマイアクティビティを見つけたら、設定画面の「走査で見つかった問題」に「JSON で書き出し直してください」と出す（黙って0件にはしない）。

### 2. ビルド

```bash
cd src/plugins/gkill_plugin_gemini
go build -o gkill_plugin_gemini .
# Windows の場合
go build -o gkill_plugin_gemini.exe .
```

### 3. 配置

```
$GKILL_HOME/plugins/{userID}/gkill_plugin_gemini/
├── manifest.json           # このディレクトリの manifest.json をコピー
├── gkill_plugin_gemini     # ビルドしたバイナリ（.exe は自動補完）
└── config.json             # データソース設定（初回起動時に自動生成される）
```

キャッシュは gkill のキャッシュディレクトリに作られる。
`gkill_server clear_cache plugin <all|user_id...>` で削除でき、次回起動時に作り直される。

```
$GKILL_HOME/caches/plugin_cache/{userID}/gkill_plugin_gemini/cache.db
```

### 4. データソースのフォルダを指定する

プラグインの初回起動時に、`manifest.json` と同じフォルダに `config.json` が
**自動生成される**。既定で `~/Kyou/GoogleTakeout_*` を見るので、そこに ZIP を置くなら編集は要らない。

```json
{
  "_comment": "書式の説明（読み飛ばされるので消してよい）",
  "_example_source_dirs": ["~/Kyou/GoogleTakeout_*", "~/Downloads/takeout-*.zip"],
  "source_dirs": [
    "~/Kyou/GoogleTakeout_*"
  ]
}
```

- 配列でも文字列（改行区切り）でも書ける
- **ZIP しか読まない。解凍しないこと。** 解凍したフォルダは読まず、設定画面の「走査で見つかった問題」に「ZIP ではないので読みません」と出る
- ワイルドカード `*` `**` `?` `[]` が使える。マッチしたフォルダは再帰的に走査して `*.zip` を探す。ZIP を直接指定してもよい
- 先頭の `~` と環境変数（`$HOME` など）を展開する
  （Windows サービスとして動かす場合は実行アカウントのホームになるため、絶対パスが確実）
- `source_dirs` を空にすると、プラグインフォルダ直下を見る
- `_` で始まるキーは書式の説明用。プラグインは読まないので消してよい
- 編集は次の検索から反映される（gkill の再起動は不要）

取り込みはバックグラウンドで進むので、**置いた直後の1回目の検索は空が返る**。進捗は設定画面に出る。

新しい書き出しの ZIP を同じフォルダに足すと、同じ往復は新しいほうだけを使う（古い ZIP を消さなくても二重にならない）。

### manifest.json / config.json の出力

`manifest.json` はバイナリに埋め込まれている。配置先で吐かせることもできる。
既定の `config.json` も同様に吐ける（通常は起動時に自動生成されるので不要）。

```bash
./gkill_plugin_gemini --gkill-print-manifest > manifest.json
./gkill_plugin_gemini --gkill-print-config   > config.json
```

## 取り込み元の形

`Takeout/マイ アクティビティ/Gemini アプリ/マイアクティビティ.json`（英語アカウントでは `Takeout/My Activity/Gemini Apps/MyActivity.json`）が JSON 配列で、**1要素 = 1往復**。新しい順に並ぶ。

| キー | 中身 |
|---|---|
| `header` / `products[]` / `activityControls[]` | 製品名。ロケールで変わるので `Gemini` の部分一致で判定する |
| `title` | `<接頭辞>: <プロンプト>`。接頭辞もロケールで変わる。接頭辞を持たない記録（操作の記録）はプロンプトを持たない |
| `time` | RFC 3339 の UTC（ミリ秒つき）。ゾーン付きなので `time.Local` に依存しない |
| `details[]` | `{name, url}`。どちらも会話 URL（`https://gemini.google.com/app/<id>`）。**会話 ID の唯一の出どころ** |
| `safeHtmlItem[]` | `{html}` = Gemini の返答。1記録に2つ以上入ることがあるので順に連結する |
| `subtitles[]` | 注記（音声の有無・添付の件数など）とその参照先 |
| `attachedFiles[]` / `imageFile` | 同じ ZIP フォルダに並ぶ添付ファイル名 |

**ファイル名でもフォルダ名でもエントリを選ばない。** Takeout はどちらもアカウントの言語で訳すので、`MyActivity.json` を探すと日本語環境で黙って0件になる。訳されないのは製品名の `Gemini` だけなので、「親フォルダ名が `Gemini` で始まる `.json`」という位置で選び、Gemini のものかは中身（`header` / `products` / `activityControls` / 会話 URL）で確かめる。

**書き出しにメッセージ ID が無い。** Kyou の ID は会話 ID・時刻・役割・連番から UUIDv5 で導出する。連番は書き出し（世代）ごとに数えるので、書き出しを足しても ID は動かない。**この導出を変えると全 Kyou の ID が変わり、利用者が付けたタグが全部外れる。**

添付（音声・画像・ZIP）の中身は読まない。名前と注記を本文の折りたたみに並べるだけ。

## プラグイン情報

| 項目 | 値 |
|---|---|
| `rep_name` | `Gemini` |
| `data_type` | `gemini_conversation` |
| プロトコルバージョン | `1` |
| 最小 gkill バージョン | `1.1.11` |

## ファイル構成

| ファイル | 内容 |
|---|---|
| `main.go` | エントリポイント、SDK ハンドラ登録、検索の絞り込み |
| `source.go` | ZIP 内のエントリの選別（位置で選ぶ）・差分の署名・設定画面に出す問題の選別 |
| `loader.go` | 要素ごとの流し読み・1往復への変換・世代をまたぐ畳み込み |
| `htmltext.go` | 返答の HTML を markdown 風の平文へ写す（chatview は生 HTML を受けないため） |
| `cache.go` | SQLite3 キャッシュ（テーブル定義・差分判定・バッチ取り込み） |
| `builder.go` | 常駐バックグラウンドビルダ |
| `uuid.go` | Kyou ID の導出（UUIDv5） |
| `render.go` | 会話詳細の HTML（SDK の `chatview` へ写す。ラベルは「あなた」/「Gemini」） |
| `html.go` | 設定画面の HTML（SDK の `configview` へ写す） |
| `types.go` | マイアクティビティの書き出し形式の型定義 |
| `manifest.json` | プラグインメタ情報 |
| `go.mod` | 独立 Go モジュール |

## 関連資料

- 取り込み元の選び方と却下案: [ADR-0321](../../../documents/adr/0321-gemini-activity-from-my-activity.md)
- ZIP だけを読む理由: [ADR-0310](../../../documents/adr/0310-chat-export-plugins-read-zip.md)
- 差分判定に CRC32 と Size を使う理由: [ADR-0303](../../../documents/adr/0303-plugin-cache-use-crc32-and-size.md)
- 常駐ビルダ: [ADR-0305](../../../documents/adr/0305-plugin-background-builder-wal.md)
- プラグイン SDK: [`src/server/gkill/plugin/README.md`](../../server/gkill/plugin/README.md)
- プラグインシステム全体: [`src/plugins/README.md`](../README.md)
