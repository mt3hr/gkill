package req_res

type GenerateTLSFileRequest struct {
	SessionID string `json:"session_id"`

	LocaleName string `json:"locale_name"`
}
