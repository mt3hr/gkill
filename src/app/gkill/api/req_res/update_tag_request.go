package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type UpdateTagRequest struct {
	SessionID string `json:"session_id"`

	Tag *reps.Tag `json:"tag"`

	TXID *string `json:"tx_id"`

	LocaleName string `json:"locale_name"`
}
