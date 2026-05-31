package main

import "time"

// claudeConversation は conversations.json の1会話に対応する。
type claudeConversation struct {
	UUID         string         `json:"uuid"`
	Name         string         `json:"name"`
	Summary      string         `json:"summary"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	ChatMessages []claudeMessage `json:"chat_messages"`
}

// claudeMessage は会話内の1メッセージ。
type claudeMessage struct {
	UUID              string    `json:"uuid"`
	Text              string    `json:"text"`
	Sender            string    `json:"sender"` // "human" or "assistant"
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	ParentMessageUUID string    `json:"parent_message_uuid"`
}
