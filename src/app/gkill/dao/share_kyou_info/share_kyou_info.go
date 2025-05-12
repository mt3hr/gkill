package share_kyou_info

type ShareKyouInfo struct {
	ID string `json:"id"`

	UserID string `json:"user_id"`

	Device string `json:"device"`

	ShareTitle string `json:"share_title"`

	IsShareDetail bool `json:"is_share_detail"`

	ShareID string `json:"share_id"`

	FindQueryJSON JSONString `json:"find_query_json"`
}

type JSONString string

func (j *JSONString) UnmarshalJSON(b []byte) error {
	*j = JSONString(b)
	return nil
}

func (j *JSONString) MarshalJSON() ([]byte, error) {
	return []byte(*j), nil
}
