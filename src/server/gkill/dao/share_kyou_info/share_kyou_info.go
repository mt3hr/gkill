package share_kyou_info

type ShareKyouInfo struct {
	ID string `json:"id"`

	ShareID string `json:"share_id"`

	UserID string `json:"user_id"`

	Device string `json:"device"`

	ShareTitle string `json:"share_title"`

	FindQueryJSON JSONString `json:"find_query_json"`

	ViewType string `json:"view_type"`

	IsShareTimeOnly bool `json:"is_share_time_only"`

	IsShareWithTags bool `json:"is_share_with_tags"`

	IsShareWithTexts bool `json:"is_share_with_texts"`

	IsShareWithTimeIss bool `json:"is_share_with_timeiss"`

	IsShareWithLocations bool `json:"is_share_with_locations"`
}

type JSONString string

func (j *JSONString) UnmarshalJSON(b []byte) error {
	*j = JSONString(b)
	return nil
}

func (j *JSONString) MarshalJSON() ([]byte, error) {
	return []byte(*j), nil
}
