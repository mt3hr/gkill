package req_res

import (
	"encoding/json"
	"time"
)

// KyouMCPDTO は MCP レスポンス用の軽量 Kyou DTO
type KyouMCPDTO struct {
	DataType      string               `json:"data_type"`
	RelatedTime   time.Time            `json:"related_time"`
	Tags          []string             `json:"tags,omitempty"`
	Texts         []string             `json:"texts,omitempty"`
	Notifications []NotificationMCPDTO `json:"notifications,omitempty"`
	Payload       interface{}          `json:"payload,omitempty"`
}

type NotificationMCPDTO struct {
	Content          string    `json:"content"`
	NotificationTime time.Time `json:"notification_time,omitempty"`
	IsNotificated    bool      `json:"is_notificated"`
}

type TimeIsPayloadMCPDTO struct {
	Kind      string     `json:"kind"` // "timeis"
	Title     string     `json:"title"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
}

type KmemoPayloadMCPDTO struct {
	Kind    string `json:"kind"` // "kmemo"
	Content string `json:"content"`
}

type KCPayloadMCPDTO struct {
	Kind     string      `json:"kind"` // "kc"
	Title    string      `json:"title"`
	NumValue json.Number `json:"num_value"`
}

type URLogPayloadMCPDTO struct {
	Kind  string `json:"kind"` // "urlog"
	Title string `json:"title"`
	URL   string `json:"url"`
}

type NlogPayloadMCPDTO struct {
	Kind   string      `json:"kind"` // "nlog"
	Title  string      `json:"title"`
	Shop   string      `json:"shop,omitempty"`
	Amount json.Number `json:"amount"`
}

type MiPayloadMCPDTO struct {
	Kind              string     `json:"kind"` // "mi"
	Title             string     `json:"title"`
	IsChecked         bool       `json:"is_checked"`
	BoardName         string     `json:"board_name,omitempty"`
	LimitTime         *time.Time `json:"limit_time,omitempty"`
	EstimateStartTime *time.Time `json:"estimate_start_time,omitempty"`
	EstimateEndTime   *time.Time `json:"estimate_end_time,omitempty"`
}

type LantanaPayloadMCPDTO struct {
	Kind string `json:"kind"` // "lantana"
	Mood int    `json:"mood"`
}

type IDFPayloadMCPDTO struct {
	Kind     string `json:"kind"` // "idf"
	FileName string `json:"file_name"`
}

type GitPayloadMCPDTO struct {
	Kind          string `json:"kind"` // "git_commit_log"
	CommitMessage string `json:"commit_message"`
	Addition      int    `json:"addition,omitempty"`
	Deletion      int    `json:"deletion,omitempty"`
}
