package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type UpdateIDFKyouRequest struct {
	SessionID string `json:"session_id"`

	IDFKyou *reps.IDFKyou `json:"idf_kyou"`

	TXID *string `json:"tx_id"`
}
