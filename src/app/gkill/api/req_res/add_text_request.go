package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type AddTextRequest struct {
	SessionID string `json:"session_id"`

	Text *reps.Text `json:"text"`
}
