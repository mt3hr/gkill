package req_res

type GetGitCommitLogsRequest struct {
	SessionID string `json:"session_id"`

	Query string `json:"query"`
}
