package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const conversationsFile = "conversations.json"

// loadConversations は conversations.json を直接読み込む（キャッシュ再構築時に使用）。
func loadConversations(pluginDir string) ([]claudeConversation, error) {
	path := filepath.Join(pluginDir, conversationsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("conversations.json が見つかりません。Claude.ai からエクスポートしたファイルを %s に配置してください", path)
		}
		return nil, fmt.Errorf("conversations.json の読み込みに失敗しました: %w", err)
	}

	var convs []claudeConversation
	if err := json.Unmarshal(data, &convs); err != nil {
		return nil, fmt.Errorf("conversations.json のパースに失敗しました: %w", err)
	}
	return convs, nil
}
