// ˅
package req_res

// ˄

type UploadGPSLogFilesRequest struct {
	// ˅

	// ˄

	SessionID string `json:"session_id"`

	GPSLogFiles []*FileData `json:"gps_log_files"`

	// ˅

	// ˄
}

// ˅

// ˄
