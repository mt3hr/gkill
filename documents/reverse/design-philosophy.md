# gkill 設計思想

## 1. プロジェクトの動機・目的

### 背景: rykv からのリプレイス

gkill は旧ライフログシステム群（rykv, dnote, kftl, kmemo, mi, lantana, nlog, timeis, urlog 等）を1つのアプリケーションに統合するリプレイスプロジェクトとして開始された。

**旧システムの課題:**
- ソースコード老朽化（JavaScript → TypeScript への移行が必要）
- 製造当時の知識・技術不足による難読コード
- データの更新・削除機能が存在しない（CRUD の U/D が未実装）
- 同一 DB を参照する複数アプリケーションが分散（リソース消費大、操作の手間、起動待機時間）
- 似通ったコードが各アプリケーションに存在（保守コスト高）

**改修の目的:**
- アプリケーションの1本化（専有リソース削減）
- データの更新・削除機能の実現
- TypeScript リプレイスによる可読性・メンテナンス性向上
- アカウント機能の新設（アクセス制御、マルチユーザ対応）

### 根底にある思想: 「記録」のためのアプリケーション

> 「記録によって'よくなっている'気がする」

gkill は「速攻の記録」を重視する:
- ボタン1つで記録
- Enter キー1回でメモ帳起動
- キーボードだけで記録可能（保存ボタンへの腕移動不要）
- KFTL テキストベース入力で操作を最小化

「景色」と「状況」を意識した設計:
- 「いつ、どこで、私が、どのような状況で、何をしたのか」を振り返れること
- 時系列ビュー + 多軸検索（キーワード、日時、場所、状況、データ型、デバイス）

## 2. 中核概念「Kyou」

全データ型の基底エンティティとして「Kyou（きょう）」を設計。共通フィールドを持つことで、異なるデータ型を統一的に扱える。

### 共通フィールド（全 Kyou データ型に存在）

| カラム | 説明 |
|-------|------|
| `IS_DELETED` | 論理削除フラグ |
| `ID` | UUID（主キーではない — Append-Only のため） |
| `RELATED_TIME` | 関連日時（記録の時系列表示に使用） |
| `CREATE_TIME` | 作成日時 |
| `CREATE_APP` | 作成アプリケーション名 |
| `CREATE_USER` | 作成ユーザ名 |
| `CREATE_DEVICE` | 作成デバイス名 |
| `UPDATE_TIME` | 更新日時 |
| `UPDATE_APP` | 更新アプリケーション名 |
| `UPDATE_USER` | 更新ユーザ名 |
| `UPDATE_DEVICE` | 更新デバイス名 |

### データ型一覧と用途

| データ型 | テーブル名 | 用途 | 固有フィールド |
|---------|-----------|------|---------------|
| Kmemo | KMEMO | テキストメモ | CONTENT |
| KC | KC | 数値記録 | TITLE, NUM_VALUE |
| Lantana | LANTANA | 気分値（0-10） | MOOD |
| Mi | MI | タスク管理 | TITLE, IS_CHECKED, BOARD_NAME, LIMIT_TIME, ESTIMATE_START_TIME, ESTIMATE_END_TIME |
| Nlog | NLOG | 支出記録 | SHOP, TITLE, AMOUNT |
| URLog | URLOG | ブックマーク | URL, TITLE, DESCRIPTION, FAVICON_IMAGE, THUMBNAIL_IMAGE |
| TimeIs | TIMEIS | タイムスタンプ | TITLE, START_TIME, END_TIME |
| Tag | TAG | タグ | TARGET_ID, TAG |
| Text | TEXT | テキスト注釈 | TARGET_ID, TEXT |
| Notification | NOTIFICATION | 通知 | TARGET_ID, NOTIFICATION_TIME, CONTENT, IS_NOTIFICATED |
| ReKyou | REKYOU | リポスト | TARGET_ID |
| IDFKyou | IDF | ファイル | TARGET_REP_NAME, TARGET_FILE |

### メタ情報の設計

Tag, Text, Notification は `TARGET_ID` で任意の Kyou に紐づけ可能。
これにより、全データ型に共通のメタ情報付与機能を提供。

## 3. Append-Only DAO

### 設計意図

旧システムの最大の課題であった「更新・削除ができない」問題を、**Append-Only 方式**で解決。

```
データ追加: INSERT（新規レコード）
データ更新: INSERT（同一IDで新しいレコードを追加、UPDATE_TIMEで識別）
データ削除: INSERT（同一IDでIS_DELETED=TRUEのレコードを追加）
```

**ID 列に主キー制約を持たない**ことがこの設計のポイント。同一 ID のレコードが複数行存在し、UPDATE_TIME が最新のものが有効データとなる。

### メリット

- **完全な変更履歴の保持**: 全変更がレコードとして残る
- **ロールバック不要**: 過去の状態は常に参照可能
- **並行性の担保**: 排他制御なしに複数端末から書き込み可能
- **シンプルな実装**: UPDATE/DELETE 文が不要、INSERT のみ

### 最新データの取得

```sql
-- 同一IDで UPDATE_TIME が最新のレコードを取得
SELECT * FROM KMEMO
WHERE ID = ?
ORDER BY UPDATE_TIME DESC
LIMIT 1
```

### 論理削除

```sql
-- IS_DELETED = TRUE のレコードをINSERTすることで削除を表現
INSERT INTO KMEMO (IS_DELETED, ID, ..., UPDATE_TIME, ...)
VALUES (TRUE, ?, ..., datetime('now'), ...)
```

## 4. Repository 4層パターン

> クラス図の詳細は [class-diagrams.md](class-diagrams.md)、実装仕様は [program-spec.md](program-spec.md) を参照。

各データ型に対して4層の実装を持つ:

```
┌─────────────────────────────────────┐
│  Repository Interface               │  ← Go interface
│  (kmemo_repository.go)              │
├─────────────────────────────────────┤
│  SQLite3 Implementation             │  ← DB直接操作
│  (kmemo_repository_sqlite3_impl.go) │
├─────────────────────────────────────┤
│  Cached Implementation              │  ← メモリキャッシュ
│  (kmemo_repository_cached_sqlite3_impl.go)
│  (kmemo_repository_sqlite3_impl_local_cached.go)
├─────────────────────────────────────┤
│  Temp Repository                    │  ← トランザクション用
│  (kmemo_temp_repository.go)         │
│  (kmemo_repository_temp_sqlite3_impl.go)
└─────────────────────────────────────┘
```

### 設計意図

- **Interface 分離**: 実装の差し替え可能性（テスト、将来のDB変更）
- **キャッシュ層**: 読み取りパフォーマンスの最適化
- **一時リポジトリ**: KFTL パース時のトランザクション（一括追加→コミット/ロールバック）

### GkillRepositories（集約アクセスポイント）

`GkillRepositories` 構造体が全リポジトリを統合管理。API ハンドラはこの構造体経由でデータにアクセスする。

## 5. 単一バイナリ配信（Go embed）

```
Vue 3 SPA (フロントエンド)
    ↓ npm run build
  dist/ ディレクトリ
    ↓ コピー
  src/server/gkill/api/embed/html/
    ↓ //go:embed
  Go バイナリに埋め込み
    ↓
  単一実行ファイルとして配信
```

### 設計意図

- **配布の容易さ**: 1ファイルを配置するだけでサーバ+フロントエンドが動作
- **依存関係の排除**: Node.js や Web サーバの別途インストールが不要
- **2つのデプロイモード**: HTTP サーバモード（`gkill_server`）とデスクトップアプリモード（`gkill`）

## 6. アカウント・セッション管理

### 改修前の課題

- アクセス制御なし（ローカル接続なら誰でも閲覧可能）
- 単一ユーザ利用想定
- PC とスマホでコンフィグファイルが異なる

### 改修後の設計

- **アカウント機能**: ユーザ ID + パスワード SHA256 でログイン
- **セッション管理**: UUID セッション ID、30日有効期限、有効期限チェック（ERR000373で期限切れセッションを拒否）
- **ユーザ別設定**: 同一サーバで異なるアカウントが異なるリポジトリ・設定を利用
- **管理者権限**: `IS_ADMIN` フラグでサーバ設定変更権限を制御
- **ログインレート制限**: IP単位で15分間に10回までのログイン試行制限（ERR000374）。スライディングウィンドウ方式

## 7. KFTL テキストベース記録

### 設計意図

> 「画面からの記録だと指の動きが多くて大変なのでもっと楽にしてほしい」

KFTL（Key Fairy Textbase Lifelogger）は、テキスト入力のみで全データ型を記録可能にする仕組み。
行頭のプレフィックス文字でデータ型を指定し、続く行で詳細情報を入力する。

### パース設計

```
テキスト全体 (KFTLStatement)
  ↓ 行分割
各行 (KFTLStatementLine)
  ↓ プレフィックス判定 (kftlFactory)
型別リクエスト (KFTLRequest)
  ↓ 実行
Repository への保存
```

### i18n 対応のプレフィックス

プレフィックス文字は i18n キーとして定義されており、ロケールによって変更可能。
サーバ側パーサは日本語プレフィックスと ASCII プレフィックスの**両方**を常に受理する。

| 機能 | 日本語 | ASCII |
|------|--------|-------|
| タグ | `。` | `#` |
| テキストブロック | `ーー` | `--` |
| 関連時刻 | `？` | `?` |
| 区切り | `、` | `,` |
| 次秒区切り | `、、` | `,,` |
| 保存（入力終了） | `！` | `!` |
| Mi（タスク） | `ーみ` | `/mi` |
| Lantana（気分） | `ーら` | `/mood` |
| Nlog（支出） | `ーん` | `/expense` |
| KC（数値） | `ーか` | `/num` |
| URLog（URL） | `ーう` | `/url` |
| TimeIs（打刻） | `ーち` | `/timeis` |
| TimeIs 開始 | `ーた` | `/start` |
| TimeIs 終了 | `ーえ` | `/end` |
| TimeIs 終了（存在すれば） | `ーいえ` | `/end?` |
| TimeIs タグ終了 | `ーたえ` | `/endt` |
| TimeIs タグ終了（存在すれば） | `ーいたえ` | `/endt?` |

## 8. フロントエンド設計

### 技術選定

- Vue 3 + Composition API（旧 JavaScript → TypeScript リプレイス）
- Vuetify 3（Material Design UI）
- PWA 対応（Service Worker でオフライン利用可能）

### Page → View → Dialog 階層

```
Page（ルートページ、Vue Router で遷移）
  └── View（メインコンテンツ、インライン表示）
       └── Dialog（モーダル操作、Vuetify の v-dialog）
```

### Composable パターン

各コンポーネントのロジック（リアクティブ状態、メソッド）は `use-*.ts` に分離:
- `.vue` ファイル: テンプレート + スタイルのみ
- `use-*.ts`: ビジネスロジック

### Props / Emits 分離

多くのコンポーネントで Props と Emits を別ファイルに分離し、型安全性と再利用性を確保。

## 9. 画面別機能・役割

| 画面 | 役割 | 主な操作 |
|------|------|---------|
| KFTL | テキストベース記録 | 全データ型の追加、TimeIs の開始/終了 |
| Rykv | ライフログ閲覧 | 検索、閲覧、編集、削除 |
| DNote | 日毎サマリ・集計 | 集計条件設定、グラフ表示 |
| Mi | タスク管理 | タスクの追加/編集/チェック、ボード管理 |
| Nlog | 支出確認（Rykv内） | 支出の追加/編集、金額集計 ※専用ルートなし |
| TimeIs | 時間管理 | 稼働中タイマーの表示/終了 ※専用ルート `/plaing` |
| Lantana | 気分値記録（Rykv内） | 気分値の入力（ダイアログ） ※専用ルートなし |
| URLog | URL 記録（Rykv内） | ブックマークレットからの記録 ※専用ルートなし |

## 10. 国際化設計

- vue-i18n で7言語対応（ja, en, zh, ko, es, fr, de）
- 主言語は日本語（`ja.json` が定義元）
- サーバ側も `go-i18n/v2` でメッセージを国際化
- KFTL プレフィックス文字もロケール依存（将来の多言語対応を考慮）

## 11. マルチプラットフォーム展開

```
         gkill_server (Go)
        ┌────────┴────────┐
   ブラウザ/PWA      go-astilectron
   (HTTP サーバ)     (デスクトップアプリ)
        │
   ┌────┴────┐
 Android   Wear OS
 (WebView)  (KFTL 入力)
```

- **Android**: gkill_server バイナリを APK に同梱、WebView で表示
- **Wear OS**: Pixel Watch から KFTL テンプレートベースの記録（Wearable Data Layer 経由）
- **MCP Server**: AI エージェントからの読み取りアクセス
