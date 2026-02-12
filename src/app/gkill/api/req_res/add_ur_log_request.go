package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type AddURLogRequest struct {
	SessionID string `json:"session_id"`

	URLog reps.URLog `json:"urlog"`

	TXID *string `json:"tx_id"`

	LocaleName string `json:"locale_name"`

	AddedKyou *reps.Kyou `json:"added_kyou"`

	WantResponseKyou bool `json:"want_response_kyou"`
}
