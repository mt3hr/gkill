package req_res

type LogoutRequest struct {
	SessionID string `json:"session_id"`

	LocaleName string `json:"locale_name"`

	CloseDatabase bool `json:"close_database"`
}
