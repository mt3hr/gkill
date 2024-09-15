package req_res

type GetURLogsRequest struct {
	SessionID string `json:"session_id"`

	Query string `json:"query"`
}
