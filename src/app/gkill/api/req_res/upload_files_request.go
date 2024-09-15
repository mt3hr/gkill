package req_res

type UploadFilesRequest struct {
	SessionID string

	Files []*FileData

	TargetRepName string

	ConflictBehavior FileUploadConflictBehavior
}
