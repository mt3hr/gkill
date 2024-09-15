package req_res

import "time"

type GetPlaingTimeisRequest struct {
	SessionID string `json:"session_id"`

	Time time.Time `json:"time"`
}
