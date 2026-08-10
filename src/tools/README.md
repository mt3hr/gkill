# tools

gkill プロジェクト用のユーティリティスクリプト。

## ファイル一覧

| ファイル | 内容 |
|---|---|
| `dev.mjs` | `npm run dev` の実体。`--api=<url>` で接続先gkill_serverを指定してViteを起動する |
| `license_getter.mjs` | `npm run license_getter` の実体。依存ライブラリのライセンス一覧を `LICENSES_DEPENDENCE` に出力する |
| `verify_docs.mjs` | `npm run verify_docs` の実体。資料の件数・リンク・Mermaid・マニュアル鮮度等の機械検査 |
| `build_manuals.mjs` | `resources/manual_src/` から `resources/manual/` を生成する |
| `manual_build.mjs` | マニュアル生成の共通ロジック（`build_manuals.mjs` / `verify_docs.mjs` から利用） |
| `manual_a11y.mjs` | マニュアルのアクセシビリティ検査ヘルパー |
| `manual_ascii_fix.mjs` | マニュアル内の文字化け・ASCII 修正ユーティリティ |
| `extract_manual_src.mjs` | 既存 `resources/manual/` から `manual_src/` を抽出する移行ツール |
| `verify_release_artifacts.mjs` | `npm run release` 成果物の検証 |
| `test_plugins.mjs` | `npm run test_plugins` の実体。`src/plugins/` 配下の各 Go モジュールのテストを実行 |

## dev.mjs

Vite開発サーバーを起動するラッパー。Viteは未知のCLIオプションをエラーにするため、`--api` / `--api-target` をここで取り除き、`GKILL_API_PROXY_TARGET` 環境変数に変換してViteへ渡す（`vite.config.ts` の `server.proxy` がこれを読む）。`--api` 以外の引数はそのままViteへ素通しする。

```bash
npm run dev                                  # http://localhost:9999 (既定)
npm run dev -- --api=http://127.0.0.1:19999
npm run dev -- --api=19999                   # ポート番号だけなら 127.0.0.1 を補完
npm run dev -- --api=19999 --port 5180
```

優先順位は `--api` > 環境変数 `GKILL_API_PROXY_TARGET` > 既定値。

## license_getter.mjs

依存パッケージのライセンス一覧をリポジトリルートの `LICENSES_DEPENDENCE` に生成する。4セクション構成:

| セクション | ソース | 内容 |
|---|---|---|
| Go Modules | `src/server` + `src/plugins` 配下の全 go.mod（5モジュール自動発見） | 推移依存含む本文全文。`path@version` でモジュール横断の重複排除。リポジトリ内 replace（自プロジェクト）は除外 |
| Node.js Modules | ルート `package-lock.json` (v3) | ルート package.json の dependencies から到達可能な本番依存のみ（devDependencies はビルドツールチェーンであり配布物に含まれないため対象外）。本文全文 |
| Android (Gradle) | `src/android` の `releaseRuntimeClasspath` | 推移依存のライセンス名+URL（POM から抽出、無ければ親 POM を追跡）。テスト専用依存は配布物に入らないため対象外 |
| Wear OS (Gradle) | `src/wear_os` の phone_companion + watch_app（同上） | 同上（2プロジェクトを GAV で重複排除して統合） |

```bash
npm run license_getter                 # 全セクション収集
npm run license_getter -- --skip-gradle  # Gradle 環境が無い環境用（該当セクションは未収集の注記のみ）
```

**前提:**
- `go` が PATH にあること（`go mod tidy` は実行しない。go.mod / go.sum への副作用なし）
- `npm ci`（または `npm i`）済み
- Gradle 部のみ JDK + Android SDK（無ければ `--skip-gradle`）

出力は全セクション成功後に一括書き出し（途中失敗で既存ファイルを壊さない）、エントリはソート済みで2回実行しても diff が出ない。
