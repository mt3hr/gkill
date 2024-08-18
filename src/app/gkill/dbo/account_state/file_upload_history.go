// ˅
package account_state

import "time"

// ˄

type FileUploadHistory struct {
	// ˅

	// ˄

	UserID string `json:"user_id"`

	Device string `json:"device"`

	FileName string `json:"file_name"`

	FileSizeByte string `json:"file_size_byte"`

	Successed bool `json:"successed"`

	SourceAddress string `json:"source_address"`

	UploadTime time.Time `json:"upload_time"`

	// ˅

	// ˄
}

// ˅

// ˄
