# テストガイド

## 1. 概要

gkill プロジェクトでは約1,400件の自動テストを整備しています。Go バックエンド、Vue 3 フロントエンド、MCP サーバ、Android、Wear OS の各コンポーネントにテストが存在し、データアクセス層から API 統合、UI の E2E テストまで幅広くカバーしています。

### テスト統計

| コンポーネント | テスト数 | テストファイル数 | フレームワーク |
|--------------|---------|----------------|---------------|
| Go バックエンド | ~453 | 46 | Go `testing` |
| フロントエンド ユニット | 669 | 48 | Vitest |
| フロントエンド E2E | 49 | 12 | Playwright |
| MCP サーバ | 226 | 6 | Vitest |
| Android | 2 | 2 | JUnit 4 |
| Wear OS | 9 | 9 | JUnit 4 + MockK |
| **合計** | **~1,400** | **~123** | |

### テスト仕様書

各 `src/` サブディレクトリには `ABOUT_TEST.md` が配置されており、そのフォルダ内のテスト概要を日本語で記載しています。索引は [`src/ABOUT_TEST.md`](../../src/ABOUT_TEST.md) です。

## 2. テスト実行コマンド

### 全テスト一括実行

```bash
npm test
```

このコマンドは以下の全テストを順次実行します：server → client → MCP → Android → Wear OS

### コンポーネント別実行

| コマンド | 対象 | 所要時間目安 |
|---------|------|------------|
| `npm run test_server` | Go バックエンド全体 | 数十秒 |
| `npm run test_client` | フロントエンド（ユニット + E2E） | 1〜2分 |
| `npm run test_client_unit` | フロントエンド ユニットのみ | 数十秒 |
| `npm run test_client_e2e` | フロントエンド E2E のみ | 1分前後 |
| `npm run test_mcp` | MCP サーバ | 数秒 |
| `npm run test_android` | Android | Gradle 依存 |
| `npm run test_wear_os` | Wear OS | Gradle 依存 |

### Go パッケージ単位での実行

```bash
# 特定パッケージのテスト
cd src/server && go test ./gkill/api/...
cd src/server && go test ./gkill/dao/reps/...
cd src/server && go test ./gkill/api/kftl/...

# 全パッケージ（npm run test_server と等価）
cd src/server && go test ./...

# 詳細出力
cd src/server && go test -v ./gkill/dao/reps/...
```

### Vitest の個別実行

```bash
# 特定テストファイルの実行
npx vitest run src/client/__tests__/unit/api/gkill-api.test.ts

# パターンマッチで実行
npx vitest run --reporter=verbose datas

# ウォッチモードで開発中に常時実行
npx vitest watch
```

### Playwright E2E テスト

```bash
# E2E テスト実行（事前にサーバ起動が必要）
npm run test_client_e2e

# 特定ファイル
npx playwright test src/client/__tests__/e2e/login.spec.ts

# ヘッドフルモード（ブラウザ表示）
npx playwright test --headed

# デバッグモード
npx playwright test --debug
```

> **Note:** E2E テストは `localhost:5173` への接続を前提としています。実行前に `npm run dev` でフロントエンド開発サーバを起動してください。

## 3. テストアーキテクチャ

### 3.1 Go バックエンド（`src/server/`）

```
src/server/gkill/
├── api/
│   ├── gkill_server_api_test.go      ← 統合テスト（全エンドポイント）
│   ├── find_filter_test.go            ← 検索フィルタ
│   ├── find/find_query_test.go        ← クエリビルダー
│   ├── gpslogs/gpslogs_test.go        ← GPS ログ解析
│   ├── message/message_test.go        ← メッセージフォーマット
│   ├── kftl/                          ← KFTL パーサ（3ファイル）
│   └── req_res/req_res_test.go        ← JSON 往復テスト
├── dao/
│   ├── gkill_dao_manager_test.go      ← DAO マネージャ
│   ├── account/                       ← アカウント CRUD
│   ├── account_state/                 ← セッション・アップロード履歴
│   ├── server_config/                 ← サーバ設定
│   ├── user_config/                   ← ユーザ設定・リポジトリ定義
│   ├── share_kyou_info/               ← 共有設定
│   ├── gkill_notification/            ← 通知ターゲット
│   ├── hide_files/                    ← ファイル非表示
│   ├── sqlite3impl/                   ← SQLite3 ユーティリティ
│   └── reps/                          ← リポジトリ実装（16ファイル）
│       ├── *_repository_sqlite3_impl_test.go  ← 11データ型
│       ├── cached_and_temp_test.go    ← キャッシュ層・一時層
│       └── cache/                     ← キャッシュ更新
├── dvnf/                              ← DVNF ファイル管理（2ファイル）
└── main/                              ← CLI・エントリポイント（8ファイル）
```

**テスト戦略:**

- **インメモリ SQLite3**: 全 DAO テストはインメモリデータベースを使用し、テスト間の隔離を保証
- **4層リポジトリパターン**: interface → SQLite3 実装 → キャッシュ実装 → 一時実装の各層をそれぞれテスト
- **統合テスト**: `gkill_server_api_test.go` が全11データ型の CRUD を HTTP ハンドラレベルで検証
- **テストヘルパー**: `reps/testhelper_test.go` が共通のテストデータ生成・DB セットアップを提供

### 3.2 フロントエンド ユニット（`src/client/__tests__/unit/`）

```
src/client/__tests__/
├── unit/
│   ├── api/gkill-api.test.ts         ← GkillAPI シングルトン（全メソッド）
│   ├── classes/                       ← ユーティリティ（6ファイル）
│   │   ├── deep-equals.test.ts
│   │   ├── format-date-time.test.ts
│   │   ├── looks-like-url.test.ts
│   │   ├── long-press.test.ts
│   │   ├── save-as.test.ts
│   │   └── delete-gkill-cache.test.ts
│   ├── datas/                         ← データモデル（22ファイル）
│   ├── dnote/                         ← D-note モジュール（5ファイル）
│   ├── kftl/                          ← KFTL パーサ（5ファイル）
│   ├── composables/                   ← Vue Composable（6ファイル）
│   ├── router.test.ts                 ← ルーター（12ルート）
│   ├── i18n-completeness.test.ts      ← i18n 完全性（7ロケール）
│   └── service-worker.test.ts         ← Service Worker
├── e2e/                               ← E2E テスト（後述）
└── helpers/                           ← テストヘルパー
    ├── factory.ts                     ← テストデータファクトリ
    ├── mock-api.ts                    ← API モック
    └── setup-i18n.ts                  ← i18n セットアップ
```

**テスト戦略:**

- **jsdom 環境**: Vitest の jsdom 環境でブラウザ API をシミュレート
- **API モック**: `mock-api.ts` で `GkillAPI` のメソッドをモックし、HTTP 通信なしでテスト
- **ファクトリパターン**: `factory.ts` の `makeKmemo()`, `makeMi()`, `makeTag()` 等でテストデータを生成
- **Vue 3 対応**: `@vue/test-utils` と Vitest の組み合わせで Composable と Vue コンポーネントをテスト

### 3.3 フロントエンド E2E（`src/client/__tests__/e2e/`）

全12ルートを Playwright で検証。各テストでは以下を共通チェック：

- **JS エラー検出**: ページ遷移時にコンソールエラーがないことを検証
- **レスポンシブ対応**: デスクトップとモバイルのビューポートで表示確認
- **インタラクティブ操作**: ボタンクリック、フォーム入力、ダイアログ開閉

| テストファイル | 対象ルート | 主なテスト内容 |
|-------------|-----------|--------------|
| `login.spec.ts` | `/login` | セッション永続化、認証リダイレクト、パスワードマスキング |
| `kftl-dialog.spec.ts` | `/kftl` | KFTL テキスト入力、マルチライン、テンプレート |
| `mi-board.spec.ts` | `/mi` | タスクボード表示、FAB 検出、レスポンシブ |
| `rykv.spec.ts` | `/rykv` | モバイルビューポート、URL 永続化 |
| `mkfl.spec.ts` | `/mkfl` | ファイル管理 |
| `plaing.spec.ts` | `/plaing` | 計画ビュー |
| `settings.spec.ts` | `/settings` | 設定コンテンツ、インタラクティブ操作 |
| `kyou-list.spec.ts` | `/kyou` | レコード一覧 |
| `share-page.spec.ts` | `/shared` | 共有ページ |
| `shared-mi.spec.ts` | `/shared_mi` | 共有タスク |
| `regist-first-account.spec.ts` | `/regist_first_account` | 初回アカウント登録 |
| `set-new-password.spec.ts` | `/set_new_password` | パスワード再設定 |

### 3.4 MCP サーバ（`src/mcp/__tests__/`）

| テストファイル | テスト内容 |
|-------------|-----------|
| `validation.test.mjs` | 6ツールの入力パラメータ検証（必須/型/範囲） |
| `normalization.test.mjs` | 日付・文字列・デフォルト値の正規化 |
| `constants.test.mjs` | ツール名、エラーコード、デフォルト設定値 |
| `tool-handlers.test.mjs` | 各ツールのハンドラ実行ロジック |
| `client.test.mjs` | GkillReadClient（fetch モック、認証、レスポンスパース） |
| `server.test.mjs` | McpServer ライフサイクル、トランスポート管理 |

### 3.5 Android / Wear OS

**Android** (`src/android/`): JUnit 4 + Kotlin
- ユニットテスト（JVM）: 定数検証（サーバURL、ポート、バイナリ名）
- インストルメンテーションテスト: Android フレームワーク統合

**Wear OS** (`src/wear_os/`): JUnit 4 + MockK
- phone_companion（4テスト）: 認証ストア、Activity、API クライアント（MockWebServer）、メッセージハンドリング
- watch_app（5テスト）: Activity、テンプレートキャッシュ、Wear クライアント、データモデル

## 4. テスト設定ファイル

| ファイル | 用途 |
|---------|------|
| `vitest.config.ts` | フロントエンドユニットテスト設定（jsdom, Vue 3, パスエイリアス） |
| `vitest.config.mcp.ts` | MCP サーバテスト設定（Node.js 環境, shebang 除去） |
| `playwright.config.ts` | E2E テスト設定（baseURL, タイムアウト, ブラウザ設定） |
| `src/server/go.mod` | Go テストの依存管理 |
| `src/android/app/build.gradle.kts` | Android テスト設定 |
| `src/wear_os/phone_companion/build.gradle.kts` | Wear OS phone_companion テスト設定 |
| `src/wear_os/watch_app/build.gradle.kts` | Wear OS watch_app テスト設定 |

## 5. テストカバレッジの範囲

### Go バックエンド（29パッケージ全てにテスト有）

```mermaid
graph LR
    subgraph "API 層"
        A1[gkill_server_api_test.go<br/>統合テスト]
        A2[find_query_test.go<br/>クエリフィルタ]
        A3[kftl/ テスト<br/>KFTL パーサ]
        A4[req_res_test.go<br/>JSON 往復]
        A5[gpslogs_test.go<br/>GPS 解析]
        A6[message_test.go<br/>メッセージ]
    end

    subgraph "DAO 層"
        D1[11 リポジトリ<br/>SQLite3 実装]
        D2[キャッシュ/一時<br/>リポジトリ]
        D3[アカウント<br/>セッション]
        D4[設定 DAO<br/>サーバ/ユーザ]
        D5[共有/通知<br/>DAO]
        D6[DAO マネージャ<br/>SQLite3 ユーティリティ]
    end

    subgraph "ユーティリティ"
        U1[dvnf<br/>ファイル管理]
        U2[main/common<br/>CLI/ログ/スレッド]
        U3[エントリポイント<br/>バッチ処理]
    end

    A1 --> D1
    A1 --> D3
```

### フロントエンド

```mermaid
graph LR
    subgraph "ユニットテスト"
        F1[GkillAPI<br/>全メソッド]
        F2[22 データモデル]
        F3[KFTL パーサ]
        F4[D-note モジュール]
        F5[Composable]
        F6[ユーティリティ]
        F7[ルーター / i18n<br/>Service Worker]
    end

    subgraph "E2E テスト"
        E1[12 ルート<br/>全画面]
    end

    F1 --> E1
    F5 --> E1
```

## 6. テストデータの管理

### Go テスト

Go テストではインメモリ SQLite3 データベースを使用します。各テスト関数が独立した DB インスタンスを持ち、テスト間の干渉を防止します。

```go
// テストヘルパーの使用例（概念）
func TestKmemoRepository(t *testing.T) {
    db := setupInMemoryDB(t)  // インメモリ DB を作成
    repo := NewKmemoRepositorySQLite3Impl(db)
    // テスト実行...
    // t.Cleanup() で自動クリーンアップ
}
```

### フロントエンド テスト

テストデータファクトリ (`src/client/__tests__/helpers/factory.ts`) が各データ型のモックオブジェクトを生成します。

```typescript
// ファクトリの使用例（概念）
import { makeKmemo, makeMi, makeTag } from '../helpers/factory'

const kmemo = makeKmemo({ content: 'テストメモ' })
const mi = makeMi({ title: 'テストタスク', is_checked: false })
const tag = makeTag({ tag: 'テストタグ', target_id: kmemo.id })
```

API モック (`src/client/__tests__/helpers/mock-api.ts`) は `GkillAPI` シングルトンのメソッドをスタブに置き換え、ネットワーク通信なしでテストを実行します。

## 7. テスト実行の前提条件

### 全テスト共通

```bash
# Node.js 依存パッケージのインストール（初回 or package.json 変更時）
npm install
```

### Go テスト

- Go 1.26.0 以上
- 追加のセットアップ不要（インメモリ DB 使用のため）

### フロントエンド ユニットテスト

- Node.js 20.15.1 以上
- 追加のセットアップ不要

### フロントエンド E2E テスト

```bash
# Playwright ブラウザのインストール（初回のみ）
npx playwright install

# フロントエンド開発サーバの起動
npm run dev
```

### Android テスト

```bash
# Android SDK と Java JDK が必要
cd src/android && ./gradlew test
```

### Wear OS テスト

```bash
# gradlew を android/ からコピー（初回のみ）
cp src/android/gradlew src/wear_os/
cp src/android/gradlew.bat src/wear_os/
cp -r src/android/gradle src/wear_os/

cd src/wear_os && ./gradlew test
```

## 8. テスト追加のガイドライン

### Go テスト

- テストファイルはソースファイルと同じディレクトリに `*_test.go` として配置（Go の標準慣習）
- `testing.T` を使用し、サブテスト（`t.Run`）でテストケースをグループ化
- DAO テストはインメモリ SQLite3 を使用し、外部依存を排除

### フロントエンド ユニットテスト

- テストファイルは `src/client/__tests__/unit/` 配下に、ソースのディレクトリ構成を反映して配置
- ファイル名は `{module-name}.test.ts` 形式
- `factory.ts` でテストデータを生成、`mock-api.ts` で API をモック

### フロントエンド E2E テスト

- テストファイルは `src/client/__tests__/e2e/` 配下に `{route-name}.spec.ts` 形式で配置
- `helpers.ts` の共通ユーティリティを利用
- 各テストで JS コンソールエラーの検出を組み込む

### MCP サーバ テスト

- テストファイルは `src/mcp/__tests__/` 配下に `{module}.test.mjs` 形式で配置
- `vitest.config.mcp.ts` で Node.js 環境を指定

### 新しいデータ型を追加した場合のテスト

新しいデータ型（例：新しい Kyou 派生型）を追加した場合、以下のテストが必要です：

1. **Go DAO テスト**: `dao/reps/{type}_repository_sqlite3_impl_test.go` — CRUD テスト
2. **Go API 統合テスト**: `gkill_server_api_test.go` にテストケース追加
3. **フロントエンド データモデルテスト**: `__tests__/unit/datas/{type}.test.ts`
4. **フロントエンド API テスト**: `__tests__/unit/api/gkill-api.test.ts` にテストケース追加
5. **KFTL テスト**（対応する場合）: Go 側 `kftl/*_test.go`、TS 側 `__tests__/unit/kftl/*.test.ts`

## 9. トラブルシューティング

### Go テストが失敗する

```bash
# モジュールキャッシュのクリア
cd src/server && go clean -testcache

# 詳細ログ付きで再実行
cd src/server && go test -v -count=1 ./gkill/dao/reps/...
```

### Vitest が失敗する

```bash
# node_modules を再インストール
rm -rf node_modules && npm install

# キャッシュクリア
npx vitest run --reporter=verbose --no-cache
```

### Playwright E2E が失敗する

```bash
# ブラウザの再インストール
npx playwright install --force

# 開発サーバが起動しているか確認
curl http://localhost:5173

# スクリーンショット付きデバッグ
npx playwright test --debug --trace on
```

### Android / Wear OS テストが失敗する

```bash
# Gradle キャッシュのクリア
cd src/android && ./gradlew clean
cd src/wear_os && ./gradlew clean

# gradlew が存在しない場合（Wear OS）
# src/android/ から gradlew, gradlew.bat, gradle/ をコピー
```

## 10. 関連資料

| 資料 | 説明 |
|------|------|
| [`src/ABOUT_TEST.md`](../../src/ABOUT_TEST.md) | テスト仕様書索引（全サブディレクトリへのリンク） |
| [`dev-setup.md`](dev-setup.md) | 開発環境構築手順 |
| [`operations-guide.md`](operations-guide.md) | 運用・デプロイガイド |
| [`api-endpoints.md`](api-endpoints.md) | API エンドポイント一覧（テスト対象の参照） |
| [`program-spec.md`](program-spec.md) | プログラム仕様（テスト対象の内部構造） |
