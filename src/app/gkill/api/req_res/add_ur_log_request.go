package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type AddURLogRequest struct {
	SessionID string `json:"session_id"`

	URLog *reps.URLog `json:"urlog"`

	TXID *string `json:"tx_id"`
}
