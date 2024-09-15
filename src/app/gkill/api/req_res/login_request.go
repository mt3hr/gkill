package req_res

type LoginRequest struct {
	UserID string `json:"user_id"`

	PasswordSha256 string `json:"password_sha256"`
}
