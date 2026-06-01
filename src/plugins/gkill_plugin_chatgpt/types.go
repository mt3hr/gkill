package main


// chatGPTConversation は conversations.json の1会話。
type chatGPTConversation struct {
	ID         string                     `json:"id"`
	Title      string                     `json:"title"`
	CreateTime float64                    `json:"create_time"` // Unix timestamp (float)
	UpdateTime float64                    `json:"update_time"`
	Mapping    map[string]*chatGPTNode    `json:"mapping"`
}

// chatGPTNode はマッピング内の1ノード。
type chatGPTNode struct {
	ID      string          `json:"id"`
	Message *chatGPTMessage `json:"message"`
	Parent  string          `json:"parent"`
}

// chatGPTMessage は1メッセージ。
type chatGPTMessage struct {
	ID     string              `json:"id"`
	Author chatGPTAuthor       `json:"author"`
	Content chatGPTContent     `json:"content"`
	CreateTime *float64        `json:"create_time"`
	Status string              `json:"status"`
}

// chatGPTAuthor はメッセージ送信者。
type chatGPTAuthor struct {
	Role string `json:"role"` // "user" or "assistant" or "system" or "tool"
}

// chatGPTContent はメッセージ内容。
type chatGPTContent struct {
	ContentType string   `json:"content_type"`
	Parts       []any    `json:"parts"` // 通常は[]string、画像の場合はobject
}
