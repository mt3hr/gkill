package req_res

type GetPluginContentHTMLRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
	// RepName はKyouのrep_name。フロントはKyou.rep_nameをそのまま渡す。
	RepName string `json:"rep_name"`
	KyouID  string `json:"kyou_id"`
}
