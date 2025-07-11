package req_res

import "time"

type GetIDFKyouRequest struct {
	SessionID string `json:"session_id"`

	ID string `json:"id"`

	UpdateTime *time.Time `json:"update_time"`

	RepName *string `json:"rep_name"`
}
