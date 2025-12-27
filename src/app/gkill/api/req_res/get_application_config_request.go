package req_res

type GetApplicationConfigRequest struct {
	SessionID string `json:"session_id"`

	LocaleName string `json:"locale_name"`
}
