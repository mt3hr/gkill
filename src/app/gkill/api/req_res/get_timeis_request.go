package req_res

import "time"

type GetTimeisRequest struct {
	SessionID string `json:"session_id"`

	ID string `json:"id"`

	UpdateTime *time.Time `json:"update_time"`
}
