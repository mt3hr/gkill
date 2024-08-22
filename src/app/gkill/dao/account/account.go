// ˅
package account

// ˄

type Account struct {
	// ˅

	// ˄

	UserID string `json:"user_id"`

	PasswordSha256 string `json:"password_sha256"`

	IsAdmin bool `json:"is_admin"`

	IsEnable bool `json:"is_enable"`

	PasswordResetToken string `json:"password_reset_token"`

	// ˅

	// ˄
}

// ˅

// ˄
