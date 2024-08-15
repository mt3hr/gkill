// ˅
package account

import "context"

// ˄

type accountDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (a *accountDAOSQLite3Impl) GetAllAccounts(ctx context.Context) ([]*Account, error) {
	panic("notImplements")
}
func (a *accountDAOSQLite3Impl) GetAccount(ctx context.Context, userID string) (*Account, error) {
	panic("notImplements")
}
func (a *accountDAOSQLite3Impl) AddAccount(ctx context.Context, account *Account) (bool, error) {
	panic("notImplements")
}
func (a *accountDAOSQLite3Impl) UpdateAccount(ctx context.Context, account *Account) (bool, error) {
	panic("notImplements")
}
func (a *accountDAOSQLite3Impl) DeleteAccount(ctx context.Context, userID string) (bool, error) {
	panic("notImplements")
}

// ˄
