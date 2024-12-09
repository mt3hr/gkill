package req_res

import "time"

type GetReKyouRequest struct {
	SessionID string `json:"session_id"`

	ID string `json:"id"`

	UpdateTime *time.Time `json:"update_time"`
}
