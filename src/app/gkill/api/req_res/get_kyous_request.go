package req_res

import "github.com/mt3hr/gkill/src/app/gkill/api/find"

type GetKyousRequest struct {
	SessionID string `json:"session_id"`

	Query *find.FindQuery `json:"query"`

	LocaleName string `json:"locale_name"`
}
