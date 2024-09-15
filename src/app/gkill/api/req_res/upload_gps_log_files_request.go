package req_res

type UploadGPSLogFilesRequest struct {
	SessionID string

	GPSLogFiles []*FileData

	TargetRepName string

	ConflictBehavior FileUploadConflictBehavior
}
