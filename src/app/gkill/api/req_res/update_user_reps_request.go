package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/user_config"

type UpdateUserRepsRequest struct {
	SessionID string `json:"session_id"`

	TargetUserID string `json:"target_user_id"`

	UpdatedReps []*user_config.Repository `json:"updated_reps"`

	LocaleName string `json:"locale_name"`
}
