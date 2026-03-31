# gkill MCP サーバー セットアップガイド

本資料では、gkillのMCPサーバーをセットアップし、Claude（Claude.ai / Claude Code / Claude Desktop）から利用できるようにする手順を説明します。

## 目次

1. [前提条件](#1-前提条件)
2. [gkill_server の起動](#2-gkill_server-の起動)
3. [パスワードSHA256ハッシュの生成](#3-パスワードsha256ハッシュの生成)
4. [接続方式の選択](#4-接続方式の選択)
5. [stdio方式: Claude Desktop でのセットアップ](#5-stdio方式-claude-desktop-でのセットアップ)
6. [stdio方式: Claude Code でのセットアップ](#6-stdio方式-claude-code-でのセットアップ)
7. [HTTP方式: Claude.ai Connectors でのセットアップ](#7-http方式-claudeai-connectors-でのセットアップ)
8. [HTTP方式: Claude Code (リモートMCP) でのセットアップ](#8-http方式-claude-code-リモートmcp-でのセットアップ)
9. [動作確認](#9-動作確認)
10. [トラブルシューティング](#10-トラブルシューティング)
11. [環境変数リファレンス](#11-環境変数リファレンス)
12. [MCPサーバー種別の選び方](#12-mcpサーバー種別の選び方)

---

## 1. 前提条件

### 必要なソフトウェア

| ソフトウェア | バージョン | 用途 |
|---|---|---|
| Node.js | 20.19 以上（24.x 推奨） | MCPサーバーの実行 |
| gkill_server | 最新ビルド | バックエンドAPI |
| npm | Node.js に付属 | 依存パッケージ管理 |

### 事前準備

1. gkillリポジトリをクローン済みであること

```bash
git clone https://github.com/mt3hr/gkill.git
cd gkill
```

2. npm依存パッケージをインストール済みであること

```bash
npm install
```

3. gkill_server がビルド済みであること（まだの場合）

```bash
npm run install_server
```

4. gkill_server を少なくとも1回起動して初期設定が完了していること（初回起動で `admin` ユーザーが作成されます）

---

## 2. gkill_server の起動

MCPサーバーは gkill_server のAPIを呼び出すため、gkill_server が起動している必要があります。

### 基本的な起動

```bash
gkill_server
```

デフォルトでは `http://127.0.0.1:9999` で起動します。

### TLS有効の場合

TLSが有効な場合は `https://127.0.0.1:9999` になります。MCPサーバーからの接続時に自己署名証明書を使っている場合は、環境変数 `GKILL_INSECURE=true` の設定が必要です（後述）。

### リモートからアクセスする場合

gkill_server がリモートマシンで動いている場合は、そのIPアドレスまたはホスト名を `GKILL_BASE_URL` に指定します。

---

## 3. パスワードSHA256ハッシュの生成

MCPサーバーはgkill_serverへの認証にSHA256ハッシュ化されたパスワードを使用します。以下のいずれかの方法で生成してください。

### Linux / macOS / Git Bash (Windows)

```bash
echo -n "あなたのパスワード" | sha256sum
# 出力例: a1b2c3d4e5f6... *-
# ハッシュ部分（スペースより前）をコピー
```

### PowerShell (Windows)

```powershell
$bytes = [System.Text.Encoding]::UTF8.GetBytes("あなたのパスワード")
$hash = [System.Security.Cryptography.SHA256]::Create().ComputeHash($bytes)
($hash | ForEach-Object { $_.ToString("x2") }) -join ""
```

### Python

```bash
python3 -c "import hashlib; print(hashlib.sha256('あなたのパスワード'.encode()).hexdigest())"
```

> **補足**: 初期状態では `admin` ユーザーのパスワードは空文字列です。空文字列のSHA256ハッシュは `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` です。

---

## 4. 接続方式の選択

MCPサーバーは2つのトランスポートモードを提供しています。利用するクライアントに合わせて選択してください。

| 方式 | 対応クライアント | ネットワーク | 認証 |
|---|---|---|---|
| **stdio** | Claude Desktop, Claude Code (ローカル) | ローカルのみ | 環境変数で認証情報を渡す |
| **HTTP** | Claude.ai Connectors, Claude Code (リモート), ChatGPT | リモート可 | OAuth 2.1 (自動) |

### 判断フローチャート

```
MCPサーバーとクライアントは同じマシン上？
├─ はい → stdio方式を推奨（設定が簡単）
│  ├─ Claude Desktop → セクション5へ
│  └─ Claude Code → セクション6へ
└─ いいえ → HTTP方式が必要
   ├─ Claude.ai → セクション7へ
   └─ Claude Code (リモート) → セクション8へ
```

---

## 5. stdio方式: Claude Desktop でのセットアップ

Claude Desktop はローカルのMCPサーバーとstdioで通信します。

### 5.1 設定ファイルの場所

| OS | パス |
|---|---|
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Linux | `~/.config/Claude/claude_desktop_config.json` |

### 5.2 設定の記述

`claude_desktop_config.json` を開き（なければ新規作成）、以下のように記述します。

#### Read/Write統合サーバーの場合（推奨）

```json
{
  "mcpServers": {
    "gkill-readwrite": {
      "command": "node",
      "args": [
        "/path/to/gkill/src/mcp/gkill-readwrite-server.mjs"
      ],
      "env": {
        "GKILL_BASE_URL": "http://127.0.0.1:9999",
        "GKILL_USER": "admin",
        "GKILL_PASSWORD_SHA256": "<SHA256ハッシュ>"
      }
    }
  }
}
```

#### Windows の場合の注意点

Windows では `command` にNode.jsのフルパスを指定し、`args` にもフルパスを使います。

```json
{
  "mcpServers": {
    "gkill-readwrite": {
      "command": "C:\\Program Files\\nodejs\\node.exe",
      "args": [
        "C:\\Users\\<ユーザー名>\\Git\\gkill\\src\\mcp\\gkill-readwrite-server.mjs"
      ],
      "env": {
        "GKILL_BASE_URL": "http://127.0.0.1:9999",
        "GKILL_USER": "admin",
        "GKILL_PASSWORD_SHA256": "<SHA256ハッシュ>"
      }
    }
  }
}
```

> **Tip**: Node.js のパスは `where node`（Windows）または `which node`（macOS/Linux）で確認できます。

#### TLS + 自己署名証明書の場合

gkill_server がTLS（自己署名証明書）を使っている場合は、以下の環境変数を追加します。

```json
{
  "mcpServers": {
    "gkill-readwrite": {
      "command": "node",
      "args": ["..."],
      "env": {
        "GKILL_BASE_URL": "https://127.0.0.1:9999",
        "GKILL_USER": "admin",
        "GKILL_PASSWORD_SHA256": "<SHA256ハッシュ>",
        "GKILL_INSECURE": "true",
        "NODE_TLS_REJECT_UNAUTHORIZED": "0"
      }
    }
  }
}
```

#### プロキシ環境の場合

企業プロキシ等がある環境で、gkill_server がローカルの場合はプロキシを除外します。

```json
{
  "env": {
    "GKILL_BASE_URL": "http://127.0.0.1:9999",
    "NO_PROXY": "127.0.0.1,localhost",
    "HTTP_PROXY": "",
    "HTTPS_PROXY": "",
    "ALL_PROXY": ""
  }
}
```

### 5.3 反映

設定を保存した後、Claude Desktop を再起動します。MCPサーバーが正しく認識されると、チャット画面のツールアイコンに gkill のツール一覧が表示されます。

---

## 6. stdio方式: Claude Code でのセットアップ

Claude Code ではプロジェクトルートに `.mcp.json` ファイルを配置してMCPサーバーを設定します。

### 6.1 `.mcp.json` の作成

プロジェクトルート（`gkill/`）に `.mcp.json` を作成します。

```json
{
  "mcpServers": {
    "gkill-readwrite": {
      "command": "node",
      "args": ["src/mcp/gkill-readwrite-server.mjs"],
      "env": {
        "GKILL_BASE_URL": "http://127.0.0.1:9999",
        "GKILL_USER": "admin",
        "GKILL_PASSWORD_SHA256": "<SHA256ハッシュ>"
      }
    }
  }
}
```

> **注意**: `.mcp.json` は `.gitignore` に追加することを推奨します。パスワードのSHA256ハッシュが含まれるため、リポジトリにコミットしないでください。

### 6.2 Windows の場合

```json
{
  "mcpServers": {
    "gkill-readwrite": {
      "command": "C:\\Program Files\\nodejs\\node.exe",
      "args": ["C:\\Users\\<ユーザー名>\\Git\\gkill\\src\\mcp\\gkill-readwrite-server.mjs"],
      "env": {
        "GKILL_BASE_URL": "http://127.0.0.1:9999",
        "GKILL_USER": "admin",
        "GKILL_PASSWORD_SHA256": "<SHA256ハッシュ>"
      }
    }
  }
}
```

### 6.3 TLS + 自己署名証明書 + プロキシ除外の場合

```json
{
  "mcpServers": {
    "gkill-readwrite": {
      "command": "node",
      "args": ["src/mcp/gkill-readwrite-server.mjs"],
      "env": {
        "GKILL_BASE_URL": "https://192.168.x.x:9999",
        "GKILL_USER": "admin",
        "GKILL_PASSWORD_SHA256": "<SHA256ハッシュ>",
        "GKILL_INSECURE": "true",
        "NODE_TLS_REJECT_UNAUTHORIZED": "0",
        "NO_PROXY": "192.168.x.x,localhost,127.0.0.1",
        "HTTP_PROXY": "",
        "HTTPS_PROXY": "",
        "ALL_PROXY": ""
      }
    }
  }
}
```

### 6.4 反映

Claude Code を再起動するか、`/mcp` コマンドで MCP サーバーの状態を確認します。

```
/mcp
```

接続中のMCPサーバーとツール一覧が表示されれば成功です。

---

## 7. HTTP方式: Claude.ai Connectors でのセットアップ

Claude.ai からリモートでgkillのMCPサーバーに接続する方式です。OAuth 2.1認証が自動で行われます。

### 7.1 ネットワーク要件

Claude.ai のサーバーからMCPサーバーにHTTPでアクセスできる必要があります。以下のいずれかの方法でMCPサーバーを公開してください。

| 方法 | 難易度 | 説明 |
|---|---|---|
| **Cloudflare Tunnel（推奨）** | 低 | 無料。グローバルIP不要。ポート開放不要 |
| VPS上で直接起動 | 中 | グローバルIPが必要 |
| リバースプロキシ (nginx等) | 高 | 既存のWebサーバーがある場合 |

### 7.2 Cloudflare Tunnel を使う場合

#### 7.2.1 cloudflared のインストール

```bash
# macOS
brew install cloudflare/cloudflare/cloudflared

# Windows (winget)
winget install Cloudflare.cloudflared

# Linux
# https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/
```

#### 7.2.2 Cloudflare にログイン

```bash
cloudflared tunnel login
```

ブラウザが開くので、Cloudflareアカウントでログインし、トンネルに使うドメインを選択します。

#### 7.2.3 トンネルの作成

```bash
cloudflared tunnel create gkill-mcp
```

トンネルIDが表示されるので控えておきます。

#### 7.2.4 DNSレコードの追加

```bash
cloudflared tunnel route dns gkill-mcp mcp.example.com
```

`mcp.example.com` の部分は、自分のドメインのサブドメインに置き換えてください。

#### 7.2.5 トンネル設定ファイルの作成

`~/.cloudflared/config.yml` を作成します。

```yaml
tunnel: <トンネルID>
credentials-file: ~/.cloudflared/<トンネルID>.json

ingress:
  - hostname: mcp.example.com
    service: http://localhost:8810
  - service: http_status:404
```

ポート番号はMCPサーバーの種別に合わせてください（Read: 8808, Write: 8809, ReadWrite: 8810）。

#### 7.2.6 トンネルの起動

```bash
cloudflared tunnel run gkill-mcp
```

### 7.3 MCPサーバーの起動（HTTP モード）

#### PowerShell (Windows)

```powershell
$env:GKILL_BASE_URL = "http://127.0.0.1:9999"
$env:GKILL_USER = "admin"
$env:GKILL_PASSWORD_SHA256 = "<SHA256ハッシュ>"
$env:MCP_TRANSPORT = "http"
$env:MCP_PORT = "8810"
$env:MCP_OAUTH_ISSUER = "https://mcp.example.com"
npm run mcp:gkill-readwrite-http
```

#### Bash (Linux / macOS / Git Bash)

```bash
GKILL_BASE_URL="http://127.0.0.1:9999" \
GKILL_USER="admin" \
GKILL_PASSWORD_SHA256="<SHA256ハッシュ>" \
MCP_TRANSPORT="http" \
MCP_PORT="8810" \
MCP_OAUTH_ISSUER="https://mcp.example.com" \
npm run mcp:gkill-readwrite-http
```

> **重要**: `MCP_OAUTH_ISSUER` には、Claude.ai からアクセス可能な公開URLを指定してください。`http://localhost:...` のままだとOAuth認証が失敗します。

#### TLS + 自己署名証明書の場合

gkill_server がTLS（自己署名証明書）で動いている場合は、追加で以下を設定します。

```bash
GKILL_INSECURE="true"
```

### 7.4 Claude.ai での接続設定

1. [Claude.ai](https://claude.ai) にログイン
2. **Settings** → **Connectors** → **Add custom connector**
3. 以下を入力:
   - **Name**: `gkill` など任意の名前
   - **Remote MCP Server URL**: `https://mcp.example.com/mcp`
   - **Client ID**: 空欄のまま（DCRで自動登録されます）
   - **Client Secret**: 空欄のまま
4. **Add** をクリック
5. OAuth認可画面がブラウザで開くので、gkill のユーザーID・パスワードでログイン
6. 認可が完了すると、Claude.ai のチャットでgkillのツールが利用可能になります

### 7.5 接続の確認

Claude.ai のチャットで以下のように話しかけてみてください:

```
gkillのタグ一覧を取得してください
```

`gkill_get_all_tag_names` ツールが呼び出され、タグ一覧が返ってくれば成功です。

---

## 8. HTTP方式: Claude Code (リモートMCP) でのセットアップ

Claude Code からリモートのMCPサーバーに接続する場合は、HTTP方式を使います。

### 8.1 MCPサーバーの起動

セクション7.2〜7.3 と同じ手順でMCPサーバーをHTTPモードで起動し、公開URLを用意します。

### 8.2 Claude Code での設定

Claude Code で以下のコマンドを実行します:

```
/mcp add --transport http gkill-readwrite https://mcp.example.com/mcp
```

OAuth認証フローが自動的に開始されます。ブラウザでgkillにログインして認可を完了してください。

---

## 9. 動作確認

### 9.1 MCPサーバーの起動確認（HTTPモード）

```bash
# Protected Resource Metadata が返ること
curl https://mcp.example.com/.well-known/oauth-protected-resource

# OAuth Authorization Server Metadata が返ること
curl https://mcp.example.com/.well-known/oauth-authorization-server
```

### 9.2 Claude Desktop / Claude Code での確認

Claude Desktop では、チャット画面左下のツールアイコン（ハンマーマーク）をクリックして、gkill のツール一覧が表示されることを確認します。

Claude Code では `/mcp` コマンドでMCPサーバーの接続状態を確認します。

### 9.3 ツールの動作確認

以下のプロンプトで各ツールの動作を確認できます:

| 確認内容 | プロンプト例 |
|---|---|
| Read系の基本動作 | 「gkillのタグ一覧を教えて」 |
| データ取得 | 「今日のgkillの記録を見せて」 |
| メモ作成 | 「gkillにテストメモを追加して」 |
| タスク作成 | 「gkillに『資料作成』というタスクを追加して」 |

---

## 10. トラブルシューティング

### MCPサーバーが起動しない

| 症状 | 原因 | 対処 |
|---|---|---|
| `MODULE_NOT_FOUND` エラー | npm依存パッケージ未インストール | `npm install` を実行 |
| `ECONNREFUSED` エラー | gkill_server が起動していない | gkill_server を起動してから再試行 |
| `ERR000002` / 認証エラー | ユーザー名またはパスワードが間違っている | `GKILL_USER` / `GKILL_PASSWORD_SHA256` を確認 |

### Claude Desktop でツールが表示されない

1. `claude_desktop_config.json` のJSON構文エラーがないか確認（末尾カンマ等）
2. `command` のNode.jsパスが正しいか確認
3. `args` のMCPサーバースクリプトのパスが正しいか確認
4. Claude Desktop を完全に終了してから再起動

### Claude.ai Connectors でOAuth認証が失敗する

| 症状 | 原因 | 対処 |
|---|---|---|
| 「サーバーに接続できません」 | MCPサーバーにインターネットからアクセスできない | Cloudflare Tunnel等でMCPサーバーを公開する |
| OAuth画面が表示されない | `MCP_OAUTH_ISSUER` が未設定または不正 | 公開URLを正しく設定する |
| ログイン後にエラー | `GKILL_BASE_URL` が間違っている | gkill_serverのURLを確認 |

### TLS関連のエラー

| 症状 | 原因 | 対処 |
|---|---|---|
| `UNABLE_TO_VERIFY_LEAF_SIGNATURE` | 自己署名証明書 | `GKILL_INSECURE=true` と `NODE_TLS_REJECT_UNAUTHORIZED=0` を設定 |
| `CERT_HAS_EXPIRED` | 証明書の期限切れ | 証明書を再発行するか、上記の設定で一時的に回避 |

### Claude Code で `/mcp` に表示されるがツールが使えない

1. permissionsでMCPツールが許可されているか確認（初回利用時にプロンプトが出ます）
2. `.claude/settings.json` の `permissions.allow` にMCPツール名が含まれているか確認

---

## 11. 環境変数リファレンス

### gkill接続

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `GKILL_BASE_URL` | はい | `http://127.0.0.1:9999` | gkill_serverのURL |
| `GKILL_USER` | はい | — | gkillのユーザーID |
| `GKILL_PASSWORD_SHA256` | はい※ | — | パスワードのSHA256ハッシュ（hex） |
| `GKILL_PASSWORD` | はい※ | — | 平文パスワード（`_SHA256`の代替。MCPサーバー内でハッシュ化） |
| `GKILL_SESSION_ID` | いいえ | — | 既存セッションIDを直接指定（ログインスキップ） |
| `GKILL_LOCALE` | いいえ | `ja` | ロケール（ja, en, zh, ko, es, fr, de） |
| `GKILL_INSECURE` | いいえ | `false` | `true` でTLS証明書検証をスキップ |

※ `GKILL_PASSWORD_SHA256` と `GKILL_PASSWORD` はどちらか一方を指定。SHA256版を推奨。

### トランスポート（HTTP モード用）

| 変数名 | 必須 | デフォルト | 説明 |
|---|---|---|---|
| `MCP_TRANSPORT` | いいえ | `stdio` | `stdio` または `http` |
| `MCP_PORT` | いいえ | `8808` | HTTPサーバーのポート番号 |
| `MCP_OAUTH_ISSUER` | HTTP時はい | `http://localhost:<port>` | OAuthメタデータのissuer URL。リモート接続時はクライアントがアクセス可能な公開URLを指定 |

### アクセスログ

| 変数名 | デフォルト | 説明 |
|---|---|---|
| `MCP_LOG` | `info` | アクセスログレベル（`debug`, `info`, `warn`, `error`） |

---

## 12. MCPサーバー種別の選び方

gkillは3種類のMCPサーバーを提供しています。用途に応じて選んでください。

| サーバー | ファイル | ツール数 | npm スクリプト | 用途 |
|---|---|---|---|---|
| **Read専用** | `gkill-read-server.mjs` | 7 | `mcp:gkill-read` / `mcp:gkill-read-http` | 閲覧のみ。データを変更したくない場合 |
| **Write専用** | `gkill-write-server.mjs` | 14 | `mcp:gkill-write` / `mcp:gkill-write-http` | 書き込み中心。Read便利ツール3つ付属 |
| **Read/Write統合** | `gkill-readwrite-server.mjs` | 18 | `mcp:gkill-readwrite` / `mcp:gkill-readwrite-http` | 全機能。迷ったらこれ |

### デフォルトポート

| サーバー | stdio | HTTPデフォルトポート |
|---|---|---|
| Read | — | 8808 |
| Write | — | 8809 |
| ReadWrite | — | 8810 |

### 推奨

- **個人利用**: Read/Write統合サーバー1つで十分です
- **共有環境**: Read専用を公開し、Write専用は管理者のみアクセス可能にすると安全です
- **Claude.ai + Claude Code 併用**: HTTP方式（Claude.ai用）とstdio方式（Claude Code用）を別々に起動できます。同じgkill_serverに対して複数のMCPサーバーを同時に起動しても問題ありません

---

## 付録: 設定例まとめ

### 最小構成（ローカル / Claude Desktop / Read/Write統合）

```json
{
  "mcpServers": {
    "gkill-readwrite": {
      "command": "node",
      "args": ["/path/to/gkill/src/mcp/gkill-readwrite-server.mjs"],
      "env": {
        "GKILL_BASE_URL": "http://127.0.0.1:9999",
        "GKILL_USER": "admin",
        "GKILL_PASSWORD_SHA256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
      }
    }
  }
}
```

### フル構成（TLS + プロキシ除外 / Claude Desktop）

```json
{
  "mcpServers": {
    "gkill-readwrite": {
      "command": "node",
      "args": ["/path/to/gkill/src/mcp/gkill-readwrite-server.mjs"],
      "env": {
        "GKILL_BASE_URL": "https://192.168.x.x:9999",
        "GKILL_USER": "admin",
        "GKILL_PASSWORD_SHA256": "<SHA256ハッシュ>",
        "GKILL_LOCALE": "ja",
        "GKILL_INSECURE": "true",
        "NODE_TLS_REJECT_UNAUTHORIZED": "0",
        "NO_PROXY": "192.168.x.x,localhost,127.0.0.1",
        "HTTP_PROXY": "",
        "HTTPS_PROXY": "",
        "ALL_PROXY": ""
      }
    }
  }
}
```

### HTTP方式起動コマンド（Claude.ai 用 / Bash）

```bash
GKILL_BASE_URL="http://127.0.0.1:9999" \
GKILL_USER="admin" \
GKILL_PASSWORD_SHA256="<SHA256ハッシュ>" \
MCP_TRANSPORT="http" \
MCP_PORT="8810" \
MCP_OAUTH_ISSUER="https://mcp.example.com" \
node src/mcp/gkill-readwrite-server.mjs
```
