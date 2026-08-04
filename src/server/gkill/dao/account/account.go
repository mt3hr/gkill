// Package account はユーザアカウント情報のDAO。
package account

import "time"

// PasswordResetTokenTTL はパスワードリセットトークンの有効期間。
const PasswordResetTokenTTL = 72 * time.Hour

type Account struct {
	UserID string `json:"user_id"`

	// PasswordHash はArgon2idのPHC文字列。パスワード未設定のときはnil。
	// 資格情報そのものなのでレスポンスには絶対に載せない (json:"-")。
	PasswordHash *string `json:"-"`

	IsAdmin bool `json:"is_admin"`

	IsEnable bool `json:"is_enable"`

	PasswordResetToken *string `json:"password_reset_token"`

	// PasswordResetTokenExpiration はリセットトークンの期限。トークンがnilのときはnil。
	PasswordResetTokenExpiration *time.Time `json:"password_reset_token_expiration"`
}

// VerifyPassword はクライアントから受け取った資格情報がこのアカウントのものかを返す。
// パスワードが設定されていないアカウントは常に不一致とする (fail-closed)。
func (a *Account) VerifyPassword(credential string) (bool, error) {
	if a.PasswordHash == nil || *a.PasswordHash == "" {
		return false, nil
	}
	if len(credential) > CredentialMaxLength {
		return false, nil
	}
	return VerifyPassword(*a.PasswordHash, credential)
}

// IsPasswordResetTokenValid は渡されたトークンが有効 (一致していて期限内) かを返す。
// 比較は資格情報と同じくconstant-timeで行う。
func (a *Account) IsPasswordResetTokenValid(token string, now time.Time) bool {
	if a.PasswordResetToken == nil {
		return false
	}
	if a.PasswordResetTokenExpiration != nil && now.After(*a.PasswordResetTokenExpiration) {
		return false
	}
	return constantTimeEquals(*a.PasswordResetToken, token)
}
