package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	_ "modernc.org/sqlite"
)

// readConversationSummaries は conversation_summaries.db から 会話ID -> タイトル/プロジェクト名 を読む。
//
// あくまで補助。Codex の session_index.jsonl と同じ扱いで、
// 読めなくても取り込みは続ける(呼び出し側が警告だけ出す)。
//
// 読み取り専用で開き、fts5 の仮想テーブル(conversations_fts)には触らない。
// 本家は WAL を使っているので、同期で写したファイルには直近の数件が載っていないことがある
// —— そのときタイトルは最初の発言の冒頭で代用する(loader.go)。
func readConversationSummaries(filePath string) (map[string]conversationMeta, error) {
	// immutable=1 は使わない(WAL の残りを読めなくなる)。mode=ro で開く。
	db, err := sql.Open("sqlite", "file:"+filepath2uri(filePath)+"?mode=ro&_pragma=busy_timeout(3000)")
	if err != nil {
		return nil, fmt.Errorf("error at open conversation summaries %s: %w", filePath, err)
	}
	defer func() { _ = db.Close() }()

	rows, err := db.Query(`SELECT conversation_id, COALESCE(title, ''), COALESCE(workspace_uris, '')
FROM conversation_summaries`)
	if err != nil {
		return nil, fmt.Errorf("error at select conversation_summaries %s: %w", filePath, err)
	}
	defer func() { _ = rows.Close() }()

	metas := map[string]conversationMeta{}
	for rows.Next() {
		conversationID, title, workspaces := "", "", ""
		if err := rows.Scan(&conversationID, &title, &workspaces); err != nil {
			return metas, fmt.Errorf("error at scan conversation_summaries %s: %w", filePath, err)
		}
		if conversationID == "" {
			continue
		}
		metas[strings.ToLower(conversationID)] = conversationMeta{
			ConversationID: strings.ToLower(conversationID),
			Title:          strings.TrimSpace(title),
			Project:        projectNameOf(workspaces),
		}
	}
	return metas, rows.Err()
}

// filepath2uri は Windows のパスを SQLite の file: URI に通る形へ直す。
func filepath2uri(filePath string) string {
	return strings.ReplaceAll(filePath, `\`, "/")
}

// projectNameOf は workspace_uris(JSON配列の文字列)からプロジェクト名を作る。
//
// 中身は ["file:///c:/path/to/repo"] で、パーセントエンコードされている版もある。
// 開いているフォルダの名前だけをチップに出す(絶対パスは端末固有なので出さない)。
// プロジェクトに属さない会話では空文字や空配列になる。
func projectNameOf(workspaceURIs string) string {
	trimmed := strings.TrimSpace(workspaceURIs)
	if trimmed == "" {
		return ""
	}
	var uris []string
	if json.Unmarshal([]byte(trimmed), &uris) != nil {
		// 配列でなく1本のURIが直接入っていることもありうる
		uris = []string{trimmed}
	}
	for _, uri := range uris {
		name := lastPathElement(decodeURIPath(uri))
		if name != "" {
			return name
		}
	}
	return ""
}

// decodeURIPath は file:// の URI からパス部分を取り出してパーセントデコードする。
func decodeURIPath(uri string) string {
	path := strings.TrimSpace(uri)
	path = strings.TrimPrefix(path, "file://")
	if decoded, err := url.PathUnescape(path); err == nil {
		path = decoded
	}
	return path
}

// lastPathElement はパスの末尾の要素を返す。区切りは / と \ の両方。
func lastPathElement(p string) string {
	trimmed := strings.TrimRight(p, `/\`)
	if trimmed == "" {
		return ""
	}
	if i := strings.LastIndexAny(trimmed, `/\`); i >= 0 {
		return trimmed[i+1:]
	}
	return trimmed
}
