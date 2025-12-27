package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type AddKmemoRequest struct {
	SessionID string `json:"session_id"`

	Kmemo *reps.Kmemo `json:"kmemo"`

	TXID *string `json:"tx_id"`

	LocaleName string `json:"locale_name"`
}
