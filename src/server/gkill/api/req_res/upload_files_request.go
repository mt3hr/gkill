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

	// CreateDevice は登録する IDFKyou の create_device / update_device に刻む端末名。
	// 空ならサーバの端末名（Web の画面は送らない）。MCP の gkill_upload_files は "mcp" を送る。
	CreateDevice string `json:"create_device,omitempty"`

	LocaleName string `json:"locale_name"`
}
