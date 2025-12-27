package req_res

type GetNotificationsByTargetIDRequest struct {
	SessionID string `json:"session_id"`

	TargetID string `json:"target_id"`

	LocaleName string `json:"locale_name"`
}
