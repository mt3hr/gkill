// ˅
package account_state

import "context"

// ˄

type loginSessionDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (l *loginSessionDAOSQLite3Impl) GetAllLoginSessions(ctx context.Context) ([]*LoginSession, error) {
	panic("notImplements")
}

func (l *loginSessionDAOSQLite3Impl) GetLoginSessions(ctx context.Context, userID string, device string) ([]*LoginSession, error) {
	panic("notImplements")
}

func (l *loginSessionDAOSQLite3Impl) AddLoginSession(ctx context.Context, loginSession *LoginSession) (bool, error) {
	panic("notImplements")
}

func (l *loginSessionDAOSQLite3Impl) UpdateLoginSession(ctx context.Context, loginSession *LoginSession) (bool, error) {
	panic("notImplements")
}

func (l *loginSessionDAOSQLite3Impl) DeleteLoginSession(ctx context.Context, id string) (bool, error) {
	panic("notImplements")
}

// ˄
