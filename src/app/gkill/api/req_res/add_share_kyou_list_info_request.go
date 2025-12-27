package req_res

type AddShareKyouListInfoRequest struct {
	SessionID string `json:"session_id"`

	ShareKyouListInfo *ShareKyouListInfo `json:"share_kyou_list_info"`

	LocaleName string `json:"locale_name"`
}
