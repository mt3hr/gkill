package req_res

type GetMisRequest struct {
	SessionID string `json:"session_id"`

	Query string `json:"query"`
}
