package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type AddReKyouRequest struct {
	SessionID string `json:"session_id"`

	ReKyou *reps.ReKyou `json:"rekyou"`
}
