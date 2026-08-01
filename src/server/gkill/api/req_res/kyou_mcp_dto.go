package req_res

import (
	"encoding/json"
	"time"
)

// KyouMCPDTO は MCP レスポンス用の軽量 Kyou DTO
type KyouMCPDTO struct {
	ID            string               `json:"id,omitempty"`
	DataType      string               `json:"data_type"`
	RelatedTime   time.Time            `json:"related_time"`
	Tags          []string             `json:"tags,omitempty"`
	Texts         []string             `json:"texts,omitempty"`
	Notifications []NotificationMCPDTO `json:"notifications,omitempty"`
	TimeIs        []TimeIsMCPDTO       `json:"timeis,omitempty"`
	Payload       any          `json:"payload,omitempty"`
}

// TimeIsMCPDTO は attached TimeIs（Plaing TimeIs）用DTO
type TimeIsMCPDTO struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
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
	IsImage  bool   `json:"is_image"`
	IsVideo  bool   `json:"is_video"`
	IsAudio  bool   `json:"is_audio"`
	RepName  string `json:"rep_name"`
	MimeType string `json:"mime_type,omitempty"`
	// FilePath はファイルの絶対パス。同一マシンからのリクエストのときだけ埋める。
	FilePath string `json:"file_path,omitempty"`
}

// PluginPayloadMCPDTO はプラグインが提供するKyouのペイロード。
// プラグインKyouの本文はgkill側に保存されておらず、プラグインから
// コンテンツHTMLとして取り出すしかないので、その取得に必要な
// rep_name / kyou_id をペイロードに含める（idfのrep_name/file_nameと同じ考え方）。
type PluginPayloadMCPDTO struct {
	Kind        string `json:"kind"` // "plugin"
	DataType    string `json:"data_type"`
	RepName     string `json:"rep_name"`
	KyouID      string `json:"kyou_id"`
	PluginName  string `json:"plugin_name,omitempty"`
	Description string `json:"description,omitempty"`
}

type GitPayloadMCPDTO struct {
	Kind          string `json:"kind"` // "git_commit_log"
	CommitMessage string `json:"commit_message"`
	Addition      int    `json:"addition,omitempty"`
	Deletion      int    `json:"deletion,omitempty"`
}
