package req_res

type UploadFilesRequest struct {
	SessionID string `json:"session_id"`

	Files []*FileData `json:"files"`

	TargetRepName string `json:"target_rep_name"`

	ConflictBehavior FileUploadConflictBehavior `json:"conflict_behavior"`

	// CreateApp は登録する IDFKyou の create_app / update_app に刻むアプリ名。
	// 空なら "gkill"（Web の画面は送らない）。MCP の gkill_upload_files が自分のアプリ名
	// （gkill_mcp_write / gkill_mcp_readwrite）を送り、画面からのアップロードと区別できるようにする。
	CreateApp string `json:"create_app"`

	LocaleName string `json:"locale_name"`
}
