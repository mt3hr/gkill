package req_res

import "github.com/mt3hr/gkill/src/app/gkill/dao/reps"

type UpdateURLogRequest struct {
	SessionID string `json:"session_id"`

	URLog *reps.URLog `json:"urlog"`

	ReGetURLogContent bool `json:"re_get_urlog_content"`

	TXID *string `json:"tx_id"`

	LocaleName string `json:"locale_name"`

	UpdatedKyou *reps.Kyou `json:"updated_kyou"`

	WantResponseKyou bool `json:"want_response_kyou"`
}
