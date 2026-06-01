package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const conversationsFile = "conversations.json"

// findConversationFiles は pluginDir 内の会話JSONファイルパスを返す。
// 新形式（conversations-000.json など）を優先し、なければ旧形式（conversations.json）を返す。
func findConversationFiles(pluginDir string) ([]string, error) {
	numbered, err := filepath.Glob(filepath.Join(pluginDir, "conversations-*.json"))
	if err != nil {
		return nil, fmt.Errorf("ファイル検索に失敗しました: %w", err)
	}
	if len(numbered) > 0 {
		sort.Strings(numbered)
		return numbered, nil
	}
	single := filepath.Join(pluginDir, conversationsFile)
	if _, err := os.Stat(single); err == nil {
		return []string{single}, nil
	}
	return nil, nil
}

// loadConversations は pluginDir 内の conversations*.json を読み込む。
// 新形式（conversations-000.json, conversations-001.json...）と
// 旧形式（conversations.json）の両方に対応する。
func loadConversations(pluginDir string) ([]chatGPTConversation, error) {
	paths, err := findConversationFiles(pluginDir)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("conversations.json または conversations-NNN.json が見つかりません。ChatGPT からエクスポートしたZIPを解凍し、JSONファイルを %s に配置してください", pluginDir)
	}

	var all []chatGPTConversation
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s の読み込みに失敗しました: %w", filepath.Base(path), err)
		}
		var convs []chatGPTConversation
		if err := json.Unmarshal(data, &convs); err != nil {
			return nil, fmt.Errorf("%s のパースに失敗しました: %w", filepath.Base(path), err)
		}
		all = append(all, convs...)
	}
	return all, nil
}

// getMessages はMapping から create_time 昇順でメッセージを取り出す。
// 実エクスポートには children フィールドが含まれないため、ツリー走査は行わず
// タイムスタンプでソートする。
func getMessages(conv *chatGPTConversation) []chatGPTMessage {
	if conv.Mapping == nil {
		return nil
	}
	type msgWithTime struct {
		msg  chatGPTMessage
		time float64
	}
	var all []msgWithTime
	for _, node := range conv.Mapping {
		if node.Message == nil {
			continue
		}
		m := *node.Message
		if m.Author.Role == "system" || m.Author.Role == "tool" {
			continue
		}
		if extractText(&m) == "" {
			continue
		}
		t := float64(0)
		if m.CreateTime != nil {
			t = *m.CreateTime
		}
		all = append(all, msgWithTime{msg: m, time: t})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].time < all[j].time })
	msgs := make([]chatGPTMessage, len(all))
	for i, a := range all {
		msgs[i] = a.msg
	}
	return msgs
}

// extractText はメッセージからテキストを取り出す。
func extractText(m *chatGPTMessage) string {
	if m == nil {
		return ""
	}
	var parts []string
	for _, part := range m.Content.Parts {
		switch v := part.(type) {
		case string:
			if s := strings.TrimSpace(v); s != "" {
				parts = append(parts, s)
			}
		}
	}
	return strings.Join(parts, "\n")
}
