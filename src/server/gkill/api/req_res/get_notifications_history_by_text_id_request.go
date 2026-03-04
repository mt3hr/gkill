package req_res

import "time"

type GetNotificationHistoryByNotificationIDRequest struct {
	SessionID string `json:"session_id"`

	ID string `json:"id"`

	UpdateTime *time.Time `json:"update_time"`

	LocaleName string `json:"locale_name"`
}
