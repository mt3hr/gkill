package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type AddNlogRequest struct {
	SessionID string `json:"session_id"`

	Nlog *reps.Nlog `json:"nlog"`

	TXID *string `json:"tx_id"`

	LocaleName string `json:"locale_name"`
}
