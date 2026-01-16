package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type UpdateLantanaRequest struct {
	SessionID string `json:"session_id"`

	Lantana *reps.Lantana `json:"lantana"`

	TXID *string `json:"tx_id"`

	LocaleName string `json:"locale_name"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`

	WantResponseKyou bool `json:"want_response_kyou"`
}
