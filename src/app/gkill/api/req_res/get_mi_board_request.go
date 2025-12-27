package req_res

type GetMiBoardRequest struct {
	SessionID string `json:"session_id"`

	LocaleName string `json:"locale_name"`
}
