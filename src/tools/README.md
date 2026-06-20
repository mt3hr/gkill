# tools

gkill プロジェクト用のユーティリティスクリプト。

## ファイル一覧

| ファイル | 内容 |
|---|---|
| `license_getter.ps1` | 依存ライブラリのライセンス一覧を `LICENSES_DEPENDENCE` ファイルに出力する PowerShell スクリプト |

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
