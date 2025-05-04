package user_config

import "encoding/json"

type DnoteData struct {
	ID string `json:"id"`

	UserID string `json:"user_id"`

	Device string `json:"device"`

	DnoteJSONData json.RawMessage `json:"dnote_json_data"`
}
