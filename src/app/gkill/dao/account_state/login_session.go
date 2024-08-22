// ˅
package account_state

import "time"

// ˄

type LoginSession struct {
	// ˅

	// ˄

	ID string `json:"id"`

	UserID string `json:"user_id"`

	Device string `json:"device"`

	ApplicationName string `json:"application_name"`

	SessionID string `json:"session_id"`

	ClientIPAddress string `json:"client_ip_address"`

	LoginTime time.Time `json:"login_time"`

	ExpirationTime time.Time `json:"expiration_time"`

	IsLocalAppUser bool `json:"is_local_app_user"`

	// ˅

	// ˄
}

// ˅

// ˄
