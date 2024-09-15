package req_res

type GetNlogsRequest struct {
	SessionID string `json:"session_id"`

	Query string `json:"query"`
}
