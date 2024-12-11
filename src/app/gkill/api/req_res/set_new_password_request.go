package req_res

type SetNewPasswordRequest struct {
	UserID string `json:"user_id"`

	ResetToken string `json:"reset_token"`

	NewPasswordSha256 string `json:"new_password_sha256"`
}
