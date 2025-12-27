package req_res

type GetAllRepNamesRequest struct {
	SessionID string `json:"session_id"`

	LocaleName string `json:"locale_name"`
}
