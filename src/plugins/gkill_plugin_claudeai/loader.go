package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const conversationsFile = "conversations.json"

// isConversationFile はデータソースから拾うファイル名かどうかを判定する。
func isConversationFile(name string) bool {
	return strings.EqualFold(name, conversationsFile)
}

// findConversationFiles はデータソースから conversations.json を集める。
func findConversationFiles(src expandedSource) []string {
	return collectSourceFiles(src, isConversationFile)
}

// loadConversations は見つかった conversations.json をすべて読み込む（キャッシュ再構築時に使用）。
// 複数フォルダを指定できるので、読めたものを連結して返す。
func loadConversations(src expandedSource) ([]claudeConversation, error) {
	paths := findConversationFiles(src)
	if len(paths) == 0 {
		return nil, fmt.Errorf("conversations.json が見つかりません。Claude.ai からエクスポートしたファイルを、データソースに指定したフォルダへ配置してください")
	}

	var all []claudeConversation
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s の読み込みに失敗しました: %w", filepath.Base(path), err)
		}
		var convs []claudeConversation
		if err := json.Unmarshal(data, &convs); err != nil {
			return nil, fmt.Errorf("%s のパースに失敗しました: %w", filepath.Base(path), err)
		}
		all = append(all, convs...)
	}
	return all, nil
}
