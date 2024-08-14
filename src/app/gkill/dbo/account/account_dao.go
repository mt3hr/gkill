// ˅
package account

import "context"

// ˄

type AccountDAO interface {
	GetAllAccounts(ctx context.Context) []*Account

	GetAccount(ctx context.Context, userID string) *Account

	AddAccount(ctx context.Context, account *Account) bool

	UpdateAccount(ctx context.Context, account *Account) bool

	DeleteAccount(ctx context.Context, userID string) bool

	// ˅

	// ˄
}

// ˅

// ˄
