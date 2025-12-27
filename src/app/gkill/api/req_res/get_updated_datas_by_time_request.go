package req_res

import "time"

type GetUpdatedDatasByTimeRequest struct {
	SessionID string `json:"session_id"`

	LastUpdatedTime time.Time `json:"last_updated_time"`

	LocaleName string `json:"locale_name"`
}
