// ˅
package account_state

import "context"

// ˄

type LoginSessionDAO interface {
	GetAllLoginSessions(ctx context.Context) ([]*LoginSession, error)

	GetLoginSessions(ctx context.Context, userID string, device string) ([]*LoginSession, error)

	AddLoginSession(ctx context.Context, loginSession *LoginSession) (bool, error)

	UpdateLoginSession(ctx context.Context, loginSession *LoginSession) (bool, error)

	DeleteLoginSession(ctx context.Context, id string) (bool, error)

	// ˅

	// ˄
}

// ˅

// ˄
