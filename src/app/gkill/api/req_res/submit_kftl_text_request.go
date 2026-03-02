package req_res

type SubmitKFTLTextRequest struct {
	SessionID  string `json:"session_id"`
	KFTLText   string `json:"kftl_text"`
	LocaleName string `json:"locale_name"`
}
