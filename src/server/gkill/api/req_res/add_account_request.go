package req_res

type AddAccountRequest struct {
	SessionID string `json:"session_id"`

	AccountInfo *Account `json:"account_info"`

	DoInitialize bool `json:"do_initialize"`

	LocaleName string `json:"locale_name"`
}
