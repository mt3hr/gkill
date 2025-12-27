package req_res

type GetKFTLTemplatesRequest struct {
	SessionID string `json:"session_id"`

	LocaleName string `json:"locale_name"`
}
