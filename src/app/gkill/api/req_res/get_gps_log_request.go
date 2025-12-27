package req_res

import "time"

type GetGPSLogRequest struct {
	SessionID string `json:"session_id"`

	StartDate time.Time `json:"start_date"`

	EndDate time.Time `json:"end_date"`

	LocaleName string `json:"locale_name"`
}
