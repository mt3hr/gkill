// ˅
package account_state

import "time"

// ˄

type LoginSession struct {
	// ˅

	// ˄

	ID string

	UserID string

	Device string

	ApplicationName string

	SessionID string

	ClientIPAddress string

	LoginTime time.Time

	ExpirationTime time.Time

	IsLocalAppUser bool

	// ˅

	// ˄
}

// ˅

// ˄
