package req_res

type GetGkillInfoRequest struct {
	SessionID string `json:"session_id"`

	LocaleName string `json:"locale_name"`
}
