// ˅
package req_res

// ˄

type UploadFilesRequest struct {
	// ˅

	// ˄

	SessionID string `json:"session_id"`

	Files []*FileData `json:"files"`

	// ˅

	// ˄
}

// ˅

// ˄
