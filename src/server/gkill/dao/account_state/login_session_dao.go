package account_state

import "context"

type LoginSessionDAO interface {
	GetAllLoginSessions(ctx context.Context) ([]*LoginSession, error)

	GetLoginSessions(ctx context.Context, userID string, device string) ([]*LoginSession, error)

	GetLoginSession(ctx context.Context, sessionID string) (*LoginSession, error)

	AddLoginSession(ctx context.Context, loginSession *LoginSession) (bool, error)

	UpdateLoginSession(ctx context.Context, loginSession *LoginSession) (bool, error)

	DeleteLoginSession(ctx context.Context, sessionID string) (bool, error)

	// DeleteLoginSessionsByUserID は対象ユーザのログインセッションをすべて削除する。
	// パスワードを設定しなおしたときに、それまでのセッションを失効させるために使う。
	DeleteLoginSessionsByUserID(ctx context.Context, userID string) (bool, error)

	Close(ctx context.Context) error
}
