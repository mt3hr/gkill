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

// IsPasswordResetTokenExpired は渡されたトークンが「このアカウントのものだが期限切れ」かを返す。
//
// IsPasswordResetTokenValid が偽を返した理由が期限切れなのかを区別するために使う。
// 期限切れだと分からないと、利用者にはリンクが壊れているようにしか見えず、
// 再発行を頼めばよいことに気づけない。
// トークンが一致しないときは偽を返すので、トークンを知らない者が
// 「そのアカウントにリセットトークンがあるか」を探る手がかりにはならない。
func (a *Account) IsPasswordResetTokenExpired(token string, now time.Time) bool {
	if a.PasswordResetToken == nil {
		return false
	}
	if !constantTimeEquals(*a.PasswordResetToken, token) {
		return false
	}
	return a.PasswordResetTokenExpiration != nil && now.After(*a.PasswordResetTokenExpiration)
}
