package req_res

type UploadFilesRequest struct {
	SessionID string `json:"session_id"`

	Files []*FileData `json:"files"`

	TargetRepName string `json:"target_rep_name"`

	ConflictBehavior FileUploadConflictBehavior `json:"conflict_behavior"`
}
