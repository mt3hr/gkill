package req_res

type DownloadSkillRequest struct {
	SessionID  string `json:"session_id"`
	LocaleName string `json:"locale_name"`
	Name       string `json:"name"`
}
