# tools

gkill プロジェクト用のユーティリティスクリプト。

## ファイル一覧

| ファイル | 内容 |
|---|---|
| `dev.mjs` | `npm run dev` の実体。`--api=<url>` で接続先gkill_serverを指定してViteを起動する |
| `license_getter.ps1` | 依存ライブラリのライセンス一覧を `LICENSES_DEPENDENCE` ファイルに出力する PowerShell スクリプト |

## dev.mjs

Vite開発サーバーを起動するラッパー。Viteは未知のCLIオプションをエラーにするため、`--api` / `--api-target` をここで取り除き、`GKILL_API_PROXY_TARGET` 環境変数に変換してViteへ渡す（`vite.config.ts` の `server.proxy` がこれを読む）。`--api` 以外の引数はそのままViteへ素通しする。

```bash
npm run dev                                  # http://localhost:9999 (既定)
npm run dev -- --api=http://127.0.0.1:19999
npm run dev -- --api=19999                   # ポート番号だけなら 127.0.0.1 を補完
npm run dev -- --api=19999 --port 5180
```

優先順位は `--api` > 環境変数 `GKILL_API_PROXY_TARGET` > 既定値。

## license_getter.ps1

Go モジュール（`src/server/`）と Node.js プロジェクト（プロジェクトルート）の依存ライセンスを収集し、`LICENSES_DEPENDENCE` ファイルを生成する。

```powershell
# プロジェクトルートから実行
cd src/tools
./license_getter.ps1
```

**出力先:** `../../LICENSES_DEPENDENCE`（プロジェクトルートの `LICENSES_DEPENDENCE` ファイル）

**必要なツール:**
- `go-licenses` (`go install github.com/google/go-licenses@latest`)
- Node.js + npm (`npm install` 済み)
