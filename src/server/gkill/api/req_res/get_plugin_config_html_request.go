package req_res

type GetPluginConfigHTMLRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
	// RepName はKyouのrep_name。フロントはKyou.rep_nameをそのまま渡す。
	RepName string `json:"rep_name"`
}
