package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

const conversationsFile = "conversations.json"

// isNumberedConversationFile は分割形式（conversations-000.json など）かどうかを判定する。
// Claude.ai の実物は ZIP の名前だけが conversations-000.zip と分割されていて中身は
// conversations.json だが、中身まで分割されたときにも読めるよう受け付ける。
func isNumberedConversationFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "conversations-") && strings.HasSuffix(lower, ".json")
}

// isSingleConversationFile は conversations.json かどうかを判定する。
func isSingleConversationFile(name string) bool {
	return strings.EqualFold(name, conversationsFile)
}

// findConversationEntries は開いた ZIP 群から会話ファイルのエントリを集める。
//
// **アーカイブ単位**で分割形式（conversations-NNN.json）を優先し、その ZIP に無ければ
// conversations.json を採る（gkill_plugin_chatgpt と同じ規則）。
// 返す順は Path 順（決定的。同じ会話 ID の同着は後勝ちになる）。
func findConversationEntries(set *sdk.SourceSet) []sdk.SourceEntry {
	if set == nil {
		return nil
	}
	numberedArchives := map[string]bool{}
	for _, entry := range set.Entries() {
		if isNumberedConversationFile(entry.Name) {
			numberedArchives[entry.ArchivePath] = true
		}
	}
	entries := make([]sdk.SourceEntry, 0, len(set.Entries()))
	for _, entry := range set.Entries() {
		switch {
		case isNumberedConversationFile(entry.Name):
			entries = append(entries, entry)
		case isSingleConversationFile(entry.Name) && !numberedArchives[entry.ArchivePath]:
			entries = append(entries, entry)
		}
	}
	slices.SortFunc(entries, func(a, b sdk.SourceEntry) int { return cmp.Compare(a.Path, b.Path) })
	return entries
}

// loadConversations はエントリの会話を全部読む（キャッシュ再構築時に使用）。
// 複数の ZIP（conversations-000.zip, conversations-001.zip …）を指定できるので、
// 読めたものを連結して返す。
func loadConversations(entries []sdk.SourceEntry) ([]claudeConversation, error) {
	var all []claudeConversation
	for _, entry := range entries {
		convs, err := readConversationEntry(entry)
		if err != nil {
			return nil, err
		}
		all = append(all, convs...)
	}
	return all, nil
}

// readConversationEntry は ZIP のエントリ1つを伸長ストリームのまま読む。展開はしない。
func readConversationEntry(entry sdk.SourceEntry) ([]claudeConversation, error) {
	reader, err := entry.Open()
	if err != nil {
		return nil, fmt.Errorf("%s の読み込みに失敗しました: %w", entry.Path, err)
	}
	defer func() { _ = reader.Close() }()
	convs, err := decodeConversationArray(reader)
	if err != nil {
		return nil, fmt.Errorf("%s のパースに失敗しました: %w", entry.Path, err)
	}
	return convs, nil
}

// decodeConversationArray はトップレベルが配列の JSON を要素ごとにデコードする。
//
// 配列を丸ごと Decode しない。json.Decoder は値1つを内部バッファに全部載せてから
// デコードするので、丸ごとだと生バイトぶん（実物の conversations.json は 203MB）の
// メモリが構造体の上に余計に要る。要素ごとなら1会話ぶんで済む。
func decodeConversationArray(reader io.Reader) ([]claudeConversation, error) {
	decoder := json.NewDecoder(reader)
	if err := expectDelim(decoder, '['); err != nil {
		return nil, err
	}
	var convs []claudeConversation
	for decoder.More() {
		var conv claudeConversation
		if err := decoder.Decode(&conv); err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}
	if err := expectDelim(decoder, ']'); err != nil {
		return nil, err
	}
	return convs, nil
}

// expectDelim は次のトークンが want の区切りであることを確かめる。
func expectDelim(decoder *json.Decoder, want json.Delim) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if delim, ok := token.(json.Delim); !ok || delim != want {
		return fmt.Errorf("unexpected token %v (want %v)", token, want)
	}
	return nil
}

// dedupeConversations は同じ会話 UUID が複数のエントリ・ZIP にあるとき、updated_at が
// 新しい版を1つ残す。同じなら後（Path 順で後ろの ZIP）が勝つ。
// メッセージは残した版のものだけになる（古い書き出しにしか無いメッセージは持ち越さない）。
// 一方にしか無い会話はそのまま残る（UUID の和集合）。初出順を保ち、空 UUID は触らない。
//
// これが「同じフォルダに新旧の書き出しが並んだとき、どう畳むか」の決定（ADR-0303 が要求）。
func dedupeConversations(convs []claudeConversation) []claudeConversation {
	indexByID := make(map[string]int, len(convs))
	result := convs[:0]
	for _, conv := range convs {
		if conv.UUID == "" {
			result = append(result, conv)
			continue
		}
		if index, exist := indexByID[conv.UUID]; exist {
			if !conv.UpdatedAt.Before(result[index].UpdatedAt) {
				result[index] = conv
			}
			continue
		}
		indexByID[conv.UUID] = len(result)
		result = append(result, conv)
	}
	return result
}
