package main

// chatGPTConversation は conversations.json の1会話。
type chatGPTConversation struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	CreateTime float64 `json:"create_time"` // Unix timestamp (float)
	// UpdateTime は会話単位の更新時刻。ChatGPTの書き出しにはメッセージ単位の更新時刻が無く、
	// Kyou は1メッセージ = 1件なので、Kyou の UpdateTime には CreateTime を入れている。
	// (Claude.ai の書き出しはメッセージ単位で updated_at を持つので、あちらはそれを使う)
	UpdateTime float64                 `json:"update_time"`
	Mapping    map[string]*chatGPTNode `json:"mapping"`
	// DefaultModelSlug は会話の既定モデル。メッセージ側に model_slug が無い返事の予備。
	DefaultModelSlug string `json:"default_model_slug"`
}

// chatGPTNode はマッピング内の1ノード。
type chatGPTNode struct {
	ID      string          `json:"id"`
	Message *chatGPTMessage `json:"message"`
	Parent  string          `json:"parent"`
}

// chatGPTMessage は1メッセージ。
type chatGPTMessage struct {
	ID         string         `json:"id"`
	Author     chatGPTAuthor  `json:"author"`
	Content    chatGPTContent `json:"content"`
	CreateTime *float64       `json:"create_time"`
	Status     string         `json:"status"`
	// Metadata のうちこのプラグインが使うのは model_slug（返事を書いたモデル名）だけ。
	Metadata chatGPTMetadata `json:"metadata"`
}

// chatGPTMetadata はメッセージのメタデータ。
type chatGPTMetadata struct {
	ModelSlug string `json:"model_slug"`
}

// chatGPTAuthor はメッセージ送信者。
type chatGPTAuthor struct {
	Role string `json:"role"` // "user" or "assistant" or "system" or "tool"
}

// chatGPTContent はメッセージ内容。
type chatGPTContent struct {
	ContentType string `json:"content_type"`
	Parts       []any  `json:"parts"` // 通常は[]string、画像の場合はobject
}
