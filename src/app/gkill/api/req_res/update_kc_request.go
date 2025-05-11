package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type UpdateKCRequest struct {
	SessionID string `json:"session_id"`

	KC *reps.KC `json:"kc"`
}
