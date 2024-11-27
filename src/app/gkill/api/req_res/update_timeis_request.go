package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type UpdateTimeisRequest struct {
	SessionID string `json:"session_id"`

	TimeIs *reps.TimeIs `json:"timeis"`
}
