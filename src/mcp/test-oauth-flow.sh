#!/bin/bash
# =============================================================================
# gkill MCP OAuth フロー手動テスト
# ChatGPT/Claude.ai接続デバッグ用
#
# 使い方:
#   export MCP_HOST="https://<your-mcp-host>"
#   bash src/mcp/test-oauth-flow.sh
#
# サーバー側のstderrログと合わせて確認してください。
# =============================================================================

HOST="${MCP_HOST:-http://localhost:8808}"
echo "=== gkill MCP OAuth Flow Test ==="
echo "Target: $HOST"
echo ""

# ---------------------------------------------------------------------------
# Step 1: POST /mcp (未認証) → 401 + WWW-Authenticate
# ---------------------------------------------------------------------------
echo "--- Step 1: POST /mcp (unauthenticated) → expect 401 ---"
STEP1=$(curl -s -w "\n%{http_code}" -X POST "$HOST/mcp" \
  -H "Content-Type: application/json" \
  -D /dev/stderr \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' \
  2>&1)
echo "$STEP1"
echo ""

# ---------------------------------------------------------------------------
# Step 2: GET /.well-known/oauth-protected-resource
# ---------------------------------------------------------------------------
echo "--- Step 2: GET /.well-known/oauth-protected-resource ---"
PRM=$(curl -s "$HOST/.well-known/oauth-protected-resource")
echo "$PRM" | python3 -m json.tool 2>/dev/null || echo "$PRM"
echo ""

# Check authorization_servers points to the right host
echo "  → authorization_servers should point to: $HOST"
echo ""

# ---------------------------------------------------------------------------
# Step 3: GET /.well-known/oauth-protected-resource/mcp (Claude.ai variant)
# ---------------------------------------------------------------------------
echo "--- Step 3: GET /.well-known/oauth-protected-resource/mcp ---"
PRM2=$(curl -s "$HOST/.well-known/oauth-protected-resource/mcp")
echo "$PRM2" | python3 -m json.tool 2>/dev/null || echo "$PRM2"
echo ""

# ---------------------------------------------------------------------------
# Step 4: GET /.well-known/oauth-authorization-server
# ---------------------------------------------------------------------------
echo "--- Step 4: GET /.well-known/oauth-authorization-server ---"
META=$(curl -s "$HOST/.well-known/oauth-authorization-server")
echo "$META" | python3 -m json.tool 2>/dev/null || echo "$META"
echo ""

# Extract endpoints from metadata
AUTH_EP=$(echo "$META" | python3 -c "import sys,json; print(json.load(sys.stdin)['authorization_endpoint'])" 2>/dev/null)
TOKEN_EP=$(echo "$META" | python3 -c "import sys,json; print(json.load(sys.stdin)['token_endpoint'])" 2>/dev/null)
REG_EP=$(echo "$META" | python3 -c "import sys,json; print(json.load(sys.stdin)['registration_endpoint'])" 2>/dev/null)

echo "  → authorization_endpoint: $AUTH_EP"
echo "  → token_endpoint:         $TOKEN_EP"
echo "  → registration_endpoint:  $REG_EP"
echo ""
echo "  ★ Check: これらのURLは $HOST で始まっていますか？"
echo "    localhost になっている場合、MCP_OAUTH_ISSUER の設定漏れです。"
echo ""

# ---------------------------------------------------------------------------
# Step 5: POST /oauth/register (DCR)
# ---------------------------------------------------------------------------
echo "--- Step 5: POST /oauth/register (Dynamic Client Registration) ---"
REG=$(curl -s "$HOST/oauth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "client_name": "test-cli",
    "redirect_uris": ["http://localhost:9999/callback"],
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "token_endpoint_auth_method": "none"
  }')
echo "$REG" | python3 -m json.tool 2>/dev/null || echo "$REG"
CLIENT_ID=$(echo "$REG" | python3 -c "import sys,json; print(json.load(sys.stdin)['client_id'])" 2>/dev/null)
echo ""
echo "  → client_id: $CLIENT_ID"
echo "  ★ Check: client_id_issued_at が含まれていますか？"
echo ""

# ---------------------------------------------------------------------------
# Step 5b: POST /register (Claude.ai fallback path)
# ---------------------------------------------------------------------------
echo "--- Step 5b: POST /register (Claude.ai fallback) ---"
REG2=$(curl -s -o /dev/null -w "%{http_code}" "$HOST/register" \
  -H "Content-Type: application/json" \
  -d '{"redirect_uris": ["http://localhost/cb"], "client_name": "test-fallback"}')
echo "  HTTP status: $REG2 (expect 201)"
echo ""

# ---------------------------------------------------------------------------
# Step 6: GET /oauth/authorize (login form)
# ---------------------------------------------------------------------------
echo "--- Step 6: GET /oauth/authorize (login form) ---"
# Generate PKCE
CODE_VERIFIER=$(python3 -c "import secrets; print(secrets.token_urlsafe(32))" 2>/dev/null || echo "test-verifier-that-is-long-enough-for-pkce-validation-requirement")
CODE_CHALLENGE=$(echo -n "$CODE_VERIFIER" | openssl dgst -sha256 -binary | openssl base64 -A | tr '+/' '-_' | tr -d '=')

AUTH_URL="$HOST/oauth/authorize?response_type=code&client_id=${CLIENT_ID:-test-client}&redirect_uri=http://localhost:9999/callback&code_challenge=$CODE_CHALLENGE&code_challenge_method=S256&scope=gkill:read&state=test123&resource=$HOST/mcp"
echo "  URL: $AUTH_URL"

AUTH_RESP=$(curl -s -o /dev/null -w "%{http_code}" "$AUTH_URL")
echo "  HTTP status: $AUTH_RESP (expect 200)"
echo ""

# ---------------------------------------------------------------------------
# Step 6b: GET /authorize (Claude.ai fallback)
# ---------------------------------------------------------------------------
echo "--- Step 6b: GET /authorize (Claude.ai fallback) ---"
FALLBACK_URL="$HOST/authorize?response_type=code&client_id=test&redirect_uri=http://localhost/cb&code_challenge=$CODE_CHALLENGE&code_challenge_method=S256"
FALLBACK_RESP=$(curl -s -o /dev/null -w "%{http_code}" "$FALLBACK_URL")
echo "  HTTP status: $FALLBACK_RESP (expect 200)"
echo ""

# ---------------------------------------------------------------------------
# Step 7: POST /oauth/authorize (submit login) → redirect with code
# ---------------------------------------------------------------------------
echo "--- Step 7: POST /oauth/authorize (login submit) ---"
echo "  ★ Note: ここではgkillのユーザーID/パスワードが必要です"
echo "  自動テストではスキップします（ブラウザで実行してください）"
echo ""
echo "  手動テスト用URL（ブラウザで開く）:"
echo "  $AUTH_URL"
echo ""

# ---------------------------------------------------------------------------
# Step 8: POST /oauth/token (code exchange) — 手動テスト用テンプレート
# ---------------------------------------------------------------------------
echo "--- Step 8: POST /oauth/token (template) ---"
echo "  認可コードを取得した後、以下を実行:"
echo ""
echo "  curl -X POST $HOST/oauth/token \\"
echo "    -H 'Content-Type: application/x-www-form-urlencoded' \\"
echo "    -d 'grant_type=authorization_code&code=<AUTH_CODE>&code_verifier=$CODE_VERIFIER&client_id=${CLIENT_ID:-test-client}&redirect_uri=http://localhost:9999/callback&resource=$HOST/mcp'"
echo ""

# ---------------------------------------------------------------------------
# Step 9: POST /mcp with Bearer token — テンプレート
# ---------------------------------------------------------------------------
echo "--- Step 9: POST /mcp with Bearer token (template) ---"
echo "  アクセストークン取得後、以下を実行:"
echo ""
echo "  # initialize"
echo "  curl -X POST $HOST/mcp \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'Authorization: Bearer <ACCESS_TOKEN>' \\"
echo "    -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2024-11-05\",\"capabilities\":{},\"clientInfo\":{\"name\":\"test\",\"version\":\"1.0\"}}}'"
echo ""
echo "  # tools/list"
echo "  curl -X POST $HOST/mcp \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'Authorization: Bearer <ACCESS_TOKEN>' \\"
echo "    -d '{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}'"
echo ""
echo "  # tools/call (gkill_get_all_tag_names)"
echo "  curl -X POST $HOST/mcp \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'Authorization: Bearer <ACCESS_TOKEN>' \\"
echo "    -d '{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"gkill_get_all_tag_names\",\"arguments\":{}}}'"
echo ""

# ---------------------------------------------------------------------------
# Summary: チェックリスト
# ---------------------------------------------------------------------------
echo "=========================================="
echo "  デバッグチェックリスト"
echo "=========================================="
echo ""
echo "  OAuth Discovery:"
echo "  [ ] Step 1: 401 + WWW-Authenticate ヘッダーが返る"
echo "  [ ] Step 2: protected-resource metadata に正しい authorization_servers"
echo "  [ ] Step 4: OAuth metadata の全URLが $HOST で始まっている"
echo "  [ ] Step 5: DCR成功、client_id_issued_at が含まれる"
echo ""
echo "  認証フロー:"
echo "  [ ] Step 6: ログインフォームが表示される"
echo "  [ ] Step 7: ログイン成功 → redirect_uri に code パラメータ付きでリダイレクト"
echo "  [ ] Step 8: トークン交換成功 → access_token, refresh_token を取得"
echo ""
echo "  MCP通信:"
echo "  [ ] Step 9: initialize → protocolVersion, serverInfo が返る"
echo "  [ ] Step 9: tools/list → 6ツールが返る"
echo "  [ ] Step 9: tools/call → 正常レスポンス（gkillバックエンド接続確認）"
echo ""
echo "  ChatGPT固有:"
echo "  [ ] ChatGPTのコールバックURL が redirect_uris に含まれる"
echo "  [ ] ChatGPTが resource パラメータを送信している（サーバーログ確認）"
echo "  [ ] ChatGPT側でツール一覧が表示される"
echo "  [ ] ChatGPTからツール呼出が成功する"
echo ""
echo "  よくあるエラー:"
echo "  - 'Couldn't reach the MCP server' → MCP_OAUTH_ISSUER がlocalhost"
echo "  - OAuth認証後も401 → トークン交換失敗（PKCE/resource不一致）"
echo "  - ツール呼出エラー → gkillバックエンド未起動/認証エラー"
echo "  - 'Internal error' → gkillバックエンドとの通信失敗"
echo ""
