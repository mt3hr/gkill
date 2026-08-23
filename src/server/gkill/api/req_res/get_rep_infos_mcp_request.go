package req_res

// GetRepInfosMCPRequest は /api/get_rep_infos_mcp のリクエスト。
type GetRepInfosMCPRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
}
