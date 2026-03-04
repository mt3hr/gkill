package req_res

type GetServerConfigsRequest struct {
	SessionID string `json:"session_id"`

	LocaleName string `json:"locale_name"`
}
