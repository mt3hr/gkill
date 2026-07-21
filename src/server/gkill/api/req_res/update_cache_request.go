package req_res

type UpdateCacheRequest struct {
	SessionID string `json:"session_id"`

	UserIDs []string `json:"user_ids"`

	LocaleName string `json:"locale_name"`
}
