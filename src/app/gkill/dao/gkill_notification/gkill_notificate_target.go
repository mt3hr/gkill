package gkill_notification

type GkillNotificateTarget struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	PublicKey    string     `json:"public_key"`
	Subscription JSONString `json:"subscription"`
}

type JSONString string

func (j *JSONString) UnmarshalJSON(b []byte) error {
	*j = JSONString(b)
	return nil
}

func (j *JSONString) MarshalJSON() ([]byte, error) {
	return []byte(*j), nil
}
