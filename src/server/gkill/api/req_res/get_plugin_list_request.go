package req_res

type GetPluginListRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
}
