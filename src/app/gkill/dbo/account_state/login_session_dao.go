// ˅
package account_state

import "context"

// ˄

type LoginSessionDAO interface {
	GetAllLoginSessions(ctx context.Context) []*LoginSession

	GetLoginSessions(ctx context.Context, userID string, device string) []*LoginSession

	AddLoginSession(ctx context.Context, loginSession *LoginSession) bool

	UpdateLoginSession(ctx context.Context, loginSession *LoginSession) bool

	DeleteLoginSession(ctx context.Context, id string) bool

	// ˅

	// ˄
}

// ˅

// ˄
