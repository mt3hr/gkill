package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type UpdateKmemoRequest struct {
	SessionID string `json:"session_id"`

	Kmemo *reps.Kmemo `json:"kmemo"`
}
