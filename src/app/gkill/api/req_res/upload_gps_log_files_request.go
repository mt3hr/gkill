package req_res

type UploadGPSLogFilesRequest struct {
	SessionID string `json:"session_id"`

	GPSLogFiles []*FileData `json:"gps_log_files"`

	TargetRepName string `json:"target_rep_name"`

	ConflictBehavior FileUploadConflictBehavior `json:"conflict_behavior"`
}
