package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type UpdateNlogRequest struct {
	SessionID string `json:"session_id"`

	Nlog *reps.Nlog `json:"nlog"`
}
