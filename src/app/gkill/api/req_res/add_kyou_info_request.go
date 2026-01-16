package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type AddKyouInfoRequest struct {
	SessionID string `json:"session_id"`

	Kyou *reps.IDFKyou `json:"kyou"`

	TXID *string `json:"tx_id"`

	LocaleName string `json:"locale_name"`

	AddedKyou *reps.Kyou `json:"added_kyou"`

	WantResponseKyou bool `json:"want_response_kyou"`
}
