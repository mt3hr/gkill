package req_res

type OpenFileRequest struct {
	SessionID string `json:"session_id"`

	TargetID string `json:"target_id"`
}
