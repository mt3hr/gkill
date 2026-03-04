package req_res

type UpdateCacheRequest struct {
	UserIDs []string `json:"user_ids"`

	LocaleName string `json:"locale_name"`
}
