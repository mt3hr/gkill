package req_res

type GetTextHistoryByTextIDRequest struct {
	SessionID string `json:"session_id"`

	ID string `json:"id"`
}
