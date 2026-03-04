package req_res

type ReloadRepositoriesRequest struct {
	SessionID string `json:"session_id"`

	LocaleName string `json:"locale_name"`

	ClearThumbCache bool `json:"clear_thumb_cache"`
}
