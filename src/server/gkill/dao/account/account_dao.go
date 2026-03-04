package account

import "context"

type AccountDAO interface {
	GetAllAccounts(ctx context.Context) ([]*Account, error)

	GetAccount(ctx context.Context, userID string) (*Account, error)

	AddAccount(ctx context.Context, account *Account) (bool, error)

	UpdateAccount(ctx context.Context, account *Account) (bool, error)

	DeleteAccount(ctx context.Context, userID string) (bool, error)

	Close(ctx context.Context) error
}
