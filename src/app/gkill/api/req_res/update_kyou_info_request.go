package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type UpdateKyouInfoRequest struct {
	SessionID string `json:"session_id"`

	Kyou *reps.IDFKyou `json:"kyou"`
}
