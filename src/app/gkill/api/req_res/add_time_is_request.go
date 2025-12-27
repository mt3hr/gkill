package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type AddTimeIsRequest struct {
	SessionID string `json:"session_id"`

	TimeIs *reps.TimeIs `json:"timeis"`

	TXID *string `json:"tx_id"`

	LocaleName string `json:"locale_name"`
}
