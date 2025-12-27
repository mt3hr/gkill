package req_res

import "time"

type GetTagHistoryByTagIDRequest struct {
	SessionID string `json:"session_id"`

	ID string `json:"id"`

	UpdateTime *time.Time `json:"update_time"`

	RepName *string `json:"rep_name"`

	LocaleName string `json:"locale_name"`
}
