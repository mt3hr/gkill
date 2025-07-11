package req_res

import "time"

type GetKCRequest struct {
	SessionID string `json:"session_id"`

	ID string `json:"id"`

	UpdateTime *time.Time `json:"update_time"`

	RepName *string `json:"rep_name"`
}
