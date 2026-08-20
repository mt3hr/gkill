# gkill_plugin_fitbit

Google Takeout の Fitbit / Google Health のデータを**日別に集計**して gkill のタイムラインに表示するプラグイン。
**1日1指標を1件の数値記録**として返すので、Dnote の推移グラフでそのまま集計できる。

心拍だけで2,200万サンプルあるため、サンプル1件ずつを記録にすることはしない。
日別に畳んでから返す（実測: 861日 × 34指標 = 19,709件）。

## セットアップ

### 1. ビルド

```bash
cd src/plugins/gkill_plugin_fitbit
go build -o gkill_plugin_fitbit .
# Windows の場合
go build -o gkill_plugin_fitbit.exe .
```

### 2. 配置

```
$GKILL_HOME/plugins/{userID}/gkill_plugin_fitbit/
├── manifest.json           # このディレクトリの manifest.json をコピー
├── gkill_plugin_fitbit     # ビルドしたバイナリ（.exe は自動補完）
└── config.json             # 取り込み元の指定（初回起動時に自動生成）
```

キャッシュは gkill のキャッシュディレクトリに作られる。

```
$GKILL_HOME/caches/plugin_cache/{userID}/gkill_plugin_fitbit/cache.db
```

`manifest.json` と既定の `config.json` はバイナリに埋め込んであるので、配置先で吐かせられる。

```bash
./gkill_plugin_fitbit --gkill-print-manifest > manifest.json
./gkill_plugin_fitbit --gkill-print-config   > config.json
```

### 3. 取り込み元のフォルダを指定する

**Google Takeout の ZIP は展開せず、そのままフォルダに置く。**

```
~/Kyou/GoogleTakeout_<端末>_<日付>/
  takeout-20260808T230152Z-1-001.zip
  takeout-20260808T230152Z-1-002.zip   ← 分割されていればそのまま並べる
```

初回起動時に `manifest.json` と同じフォルダに `config.json` が自動生成される。既存ファイルは上書きされない。

```json
{
  "source_dirs": ["~/Kyou/GoogleTakeout_*"],
  "timezone": "Asia/Tokyo",
  "metrics": [],
  "scan_workers": 0
}
```

| キー | 意味 |
|---|---|
| `source_dirs` | 取り込み元。**ZIP だけを読む**（展開済みのフォルダは対象外）。フォルダを指定するとその下の `*.zip` を再帰的に探し、ZIP の中の `Google Health/Physical Activity_GoogleData` を自動的に見つける。ZIP を直接指定してもよい。`* ** ? []` のワイルドカード、先頭の `~`、環境変数が使える |
| `timezone` | **「この日はどの日か」を決めるタイムゾーン**（既定 `Asia/Tokyo`）。サンプルの時刻はUTCなので、これが無いと日の境目が決まらない。変更すると集計をやり直す |
| `metrics` | 取り込む指標のキー。空なら全部。キーは設定画面の一覧を参照 |
| `scan_workers` | 同時に読むファイル数。0 なら自動（CPU数の半分、最大4） |

編集は**次の検索から反映される**（gkill の再起動は不要）。

> **Windows サービスで動かしている場合の注意**
> `GkillServer` は LocalSystem で動くことがある。`~` は実行アカウントのホーム
> （LocalSystem なら `C:\Windows\system32\config\systemprofile`）を指すので、
> サービス配下では**絶対パスで書くのが確実**。

## 取り込む内容

対象は `Google Health/Physical Activity_GoogleData` 配下の CSV。34指標を取り込む。

| 種類 | 指標 |
|---|---|
| 日内で合計 | 歩数 / 距離 / 消費カロリー / 活動カロリー / 上った階数 / 上昇高度 / 活動時間(軽度・中程度・高強度) / アクティブゾーン時間(全体・脂肪燃焼・有酸素・ピーク) / 心肺負荷 |
| 日内で平均・最大・最小 | 心拍数(日平均・最大・最小) / 皮膚温 / 血中酸素 |
| 分数として件数を数える | 座位時間 / 低活動時間 / 中活動時間 / 高活動時間 |
| その日の値をそのまま | 安静時心拍数 / 体調スコア / 心拍変動(HRV) / ノンレム時心拍数 / 呼吸数 / 血中酸素(Fitbit日平均) / 推定VO2Max / 心肺負荷比 / 睡眠時皮膚温 / 体重 / 身長 |

**取り込まないもの**: `*_readme.txt`、カテゴリ値だけのもの（`moods`・`micro_stillness`・`time_in_heart_rate_zone`）、
値がCSVに埋め込まれたJSONになっているもの（`body_response_event`・`daily_heart_rate_zones`）、
生のセンサー特徴量（`continuous_eda`・`micro_motion`）、座標（`gps_location` は位置情報プラグインの担当）。

**睡眠時間は対象外**。Takeout では `Google Health/Sleep Score/` と `Global Export Data/sleep-*.json` にあり、
今回の対象フォルダには入っていない。指標テーブルはデータ駆動なので、`metrics.go` に1行足せば対応できる。

## 動作の要点

- **記録のID** は `(指標キー, 現地日付)` から決まる。値やタイムゾーンには依存しないので、
  作り直しても同じIDになり、タイムゾーンを変えても記録が二重に増えるのではなく既存が更新される
- **関連時刻は現地日の正午**。0時にすると、閲覧側のブラウザが1時間西のタイムゾーンだと前日にずれる
- **更新時刻は取り込み元ファイルの mtime の最大値**。`time.Now()` にすると
  再起動のたびに全件が更新扱いになり、PWAのキャッシュが荒れる
- **タグ**は `fitbit` と指標名を付ける。gkill 本体がプラグインのタグをタグ一覧に載せるようになったので、
  rykv の既定の絞り込み「タグ無し」から漏れる問題は起きない
- **数値記録として返す**（`data_type` は `kc`、`provides` に `kc`）。
  これにより Dnote の推移グラフで `KCTitleEqualPredicate` + `AggregateSumKCNumValue` がそのまま使える。
  プラグインの記録は読み取り専用なので、編集・削除メニューは出ない
- **初回の取り込みはバックグラウンド**で行い、検索は「今キャッシュにあるぶん」を即座に返す。
  gkill のハンドラは「30秒以内」ではなく「数十ミリ秒」で返る必要があるため
  （死活確認の期限は5秒で、超えるとプロセスが殺される）。
  進捗は設定画面で確認できる。新しいファイルから処理するので、直近のデータが先に見えるようになる
- **キャッシュの差分更新**はエントリ単位。エントリ1つが1日に寄与する部分集計を持ち、
  変化したエントリの寄与だけを差し替えて日次の値を畳み直す。
  心拍は1日1ファイルなので、現地の1日はUTCの2ファイルにまたがる
- **変化したかどうかは CRC32 で見る**。Takeout は書き出し時刻を全エントリに
  同じ値で入れるので、エントリの更新時刻は中身が変わっても動かない。
  CRC32 は ZIP の中央ディレクトリに入っているので、読むのに伸長は要らない
- **書き出しをまたいで合算しない**。「ZIP を含むフォルダ + ZIP名の書き出し時刻」を
  1つの取り込み世代とし、日が重なったときは新しい世代の値だけを使う。
  同じ世代（分割された `-001` `-002` …）は合算する。
  これが無いと、古い書き出しを消さずに新しいのを置いたとき歩数が2倍になる
- **タイムゾーンのデータを埋め込んでいる**（`time/tzdata`）。Windows にはシステムの
  タイムゾーンデータベースが無く、埋め込まないと日付が静かに9時間ずれる

## 実測（271MB の ZIP / 展開後 3.7GB / 対象1,964エントリ / 約2,400万行）

| | 時間 |
|---|---|
| 初回の取り込み | 約155秒 |
| 2回目（変更なし） | 1秒未満 |

取り込み結果は 861日ぶん・34指標の 19,709件。

初回が展開済みフォルダ（約51秒）より遅いのは伸長のぶん。バックグラウンドで走り、
取り込めた日から順に見えるので、検索が待たされることはない。

## プラグイン情報

| 項目 | 値 |
|---|---|
| `rep_name` | `Fitbit` |
| `data_type` | `kc` |
| `provides` | `kc`, `tag` |
| プロトコルバージョン | `1` |
| 最小 gkill バージョン | `1.1.7` |

## ファイル構成

| ファイル | 内容 |
|---|---|
| `main.go` | エントリポイント、SDK ハンドラ登録、記録への変換、ワード検索 |
| `config.go` | `config.json` の読み直しと解釈 |
| `metrics.go` | 指標レジストリ、ファイル名の接頭辞解決、列の解決 |
| `loader.go` | 取り込み元のZIPを開く・CSV1本ぶんの部分集計 |
| `timeparse.go` | 高速なタイムスタンプ解析と現地日付のバケット |
| `builder.go` | バックグラウンドの取り込み |
| `cache.go` | SQLite3 キャッシュ（部分集計と差分更新） |
| `query.go` | キャッシュからの読み出し |
| `render.go` | 詳細 HTML（1日1指標のカードと時刻別グラフ） |
| `html.go` | 設定画面の HTML |
| `uuid.go` | 記録IDの導出 |
| `types.go` | 型定義と定数 |

## テスト

```bash
cd src/plugins/gkill_plugin_fitbit
go test ./...
```

`npm test` は `npm run test_plugins` を含むため、このテストも自動実行される。

## 関連資料

- プラグイン SDK: [`src/server/gkill/plugin/README.md`](../../server/gkill/plugin/README.md)
- プラグインシステム全体: [`src/plugins/README.md`](../README.md)
