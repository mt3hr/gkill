# gkill_plugin_google_locationhistory

Google Takeout のロケーション履歴を、gkill の**位置情報ログ**として読み込むプラグイン。
地図の表示と、地図による絞り込み（緯度・経度・半径）で使える。

**記録（Kyou）は作らない。** タイムラインには何も出ない。

## セットアップ

### 1. ビルド

```bash
cd src/plugins/gkill_plugin_google_locationhistory
go build -o gkill_plugin_google_locationhistory .
# Windows の場合
go build -o gkill_plugin_google_locationhistory.exe .
```

### 2. 配置

```
$GKILL_HOME/plugins/{userID}/gkill_plugin_google_locationhistory/
├── manifest.json
├── gkill_plugin_google_locationhistory
└── config.json               # 初回起動時に自動生成
```

キャッシュは `$GKILL_HOME/caches/plugin_cache/{userID}/gkill_plugin_google_locationhistory/cache.db`。

```bash
./gkill_plugin_google_locationhistory --gkill-print-manifest > manifest.json
./gkill_plugin_google_locationhistory --gkill-print-config   > config.json
```

### 3. 取り込み元のフォルダを指定する

**Google Takeout の ZIP は展開せず、そのままフォルダに置く。**

```
~/Kyou/GoogleTakeout_<端末>_<日付>/
  takeout-20260808T230152Z-1-001.zip
  takeout-20260808T230152Z-1-002.zip   ← 分割されていればそのまま並べる
```

```json
{
  "source_dirs": ["~/Kyou/GoogleTakeout_*"],
  "accuracy_max_meters": 100,
  "sources": [],
  "include_fitbit_gps": true,
  "visit_points": false,
  "max_points": 1000000
}
```

| キー | 意味 |
|---|---|
| `source_dirs` | 取り込み元。**ZIP だけを読む**（展開済みのフォルダは対象外）。フォルダを指定するとその下の `*.zip` を再帰的に探し、ZIP の中の `タイムライン/` や `Google Health/` を自動的に見つける。ZIP を直接指定してもよい。ワイルドカード・`~`・環境変数が使える |
| `accuracy_max_meters` | これより粗い測位を捨てる（既定100m。0以下で無効）。**精度が分からない点は残す** |
| `sources` | 測位の出所の許可リスト（`GPS` / `WIFI` / `WIFI_ONLY` / `CELL` / `UNKNOWN` / `FITBIT`）。空なら絞らない |
| `include_fitbit_gps` | ワークアウトのトラックを含めるか |
| `visit_points` | 滞在地・移動区間の端点も点として出すか。既定 `false`（生の測位より桁違いに粗く、混ぜると軌跡が読めなくなる） |
| `max_points` | 返す点数の上限 |

編集は**次の取得から反映される**（gkill の再起動は不要）。

> **設定画面への行きかた**
> gkill の設定ダイアログは、いまのところ**記録のコンテキストメニュー**からしか開けない。
> このプラグインは記録を作らないので、当面は `config.json` を直接編集する。

## 対応している形式

形式は**ファイル名ではなく中身**で判定するので、ZIP の中がどう並んでいても拾える。

| 形式 | 判定 | 状態 |
|---|---|---|
| `Timeline Edits.json` | `timelineEdits` | 対応。`rawSignal.signal.position` を読む |
| 端末からの書き出し `location-history.json` | `timelinePath` | 対応。`"35.1234°, 139.1234°"` と分オフセット |
| 旧ロケーション履歴 `Records.json` | `locations` + `latitudeE7`/`timestampMs` | 対応。数百MB〜GBになるので流し読みする |
| ワークアウトのトラック `gps_location_*.csv` | ヘッダが `timestamp,latitude,longitude` | 対応 |
| セマンティックロケーション履歴 `YYYY_MONTH.json` | `timelineObjects` | **未対応**（認識だけして設定画面に出す） |

セマンティックロケーション履歴を中途半端に読まないのは意図的。
座標の密度があるのは `activitySegment` の経路だが、書き出し時期によって座標の入れ方が違い、
`placeVisit.location` だけを読むと1日2点程度しか出ず、「読めているのに中身が薄い」状態に気づけなくなる。

## 動作の要点

- **重複除去は読み出し時**に `(時刻, 緯度, 経度)` で行う。ファイルを跨ぐので書き込み時にはできない
  （ファイル単位の差分更新が、他のファイルにもある点を消してしまう）。
  ワークアウトのトラックは全行が2つのデバイス名で2重に書き出されるので、ここが効く
- **昇順で返す**。降順に直すのは gkill 側の集約の仕事
- **走査はバックグラウンド**で行い、取得は「いまキャッシュにあるぶん」を即座に返す。
  gkill のハンドラは1呼び出し30秒・死活確認は5秒で打ち切られ、超えるとプロセスごと殺される
- **形式の判定結果はエントリ単位で覚える**。位置情報ではないと分かったエントリも覚えるので、
  走査のたびに数千エントリの先頭を伸長し直さずに済む。変化したかどうかは CRC32 で見る
  （Takeout は書き出し時刻を全エントリに同じ値で入れるので、更新時刻は使えない）
- **書き出しの順位付けはしない**。読み出し時の重複除去が別の書き出しの同じ点を1つに畳むので、
  古い書き出しを残したままでも二重にならない。むしろ Google は古いデータを間引くので、
  古い書き出しを残しておくと消えた期間が保たれる
- **記録は1件も返さない**（`emits_kyou: false`）。「記録保管場所」の一覧に出ないが、
  位置情報ログは rep の選択状態と無関係に常に使われる
- **ページングする**。gkill 側の受信バッファは1レスポンス32MBで、
  フル `Records.json` の数百万点は1回では返しきれない

## 実測（271MB の ZIP / 展開後 3.7GB / タイムライン1ファイル + トラック10ファイル）

| | 時間 |
|---|---|
| 初回スキャン | 約20秒 |
| 2回目（変更なし） | 1秒未満 |
| 点の取得 | 1秒未満 |

取り込み 15,470点 → 精度フィルタ後 15,372点 → 重複除去後 **9,107点**（2024-04-18 〜 2026-06-21）。

## プラグイン情報

| 項目 | 値 |
|---|---|
| `rep_name` | `GoogleLocation` |
| `provides` | `gpslog` |
| プロトコルバージョン | `1` |
| 最小 gkill バージョン | `1.1.7` |

## ファイル構成

| ファイル | 内容 |
|---|---|
| `main.go` | エントリポイント、SDK ハンドラ登録、設定の解釈 |
| `formats.go` | フォーマットのレジストリと中身による判定 |
| `parsers.go` | 各フォーマットの読み取り |
| `source.go` | 取り込み元のZIPを開く |
| `cache.go` | SQLite3 キャッシュ（差分更新・フィルタ・重複除去・ページング） |
| `html.go` | 設定画面の HTML |
| `types.go` | 型定義と定数 |

## テスト

```bash
cd src/plugins/gkill_plugin_google_locationhistory
go test ./...
```

## 関連資料

- プラグイン SDK: [`src/server/gkill/plugin/README.md`](../../server/gkill/plugin/README.md)
- プラグインシステム全体: [`src/plugins/README.md`](../README.md)
