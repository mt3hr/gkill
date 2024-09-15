package req_res

type Account struct {
	UserID string `json:"user_id"`

	IsAdmin bool `json:"is_admin"`

	IsEnable bool `json:"is_enable"`
}
