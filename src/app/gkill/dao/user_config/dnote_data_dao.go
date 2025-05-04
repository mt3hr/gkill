package user_config

import "context"

type DnoteDataDAO interface {
	GetAllDnoteDatas(ctx context.Context) ([]*DnoteData, error)

	GetDnoteData(ctx context.Context, userID string, device string) ([]*DnoteData, error)

	AddDnoteData(ctx context.Context, dnoteData *DnoteData) (bool, error)

	DeleteDnoteData(ctx context.Context, id string) (bool, error)

	DeleteUsersDnoteData(ctx context.Context, userID string) (bool, error)

	Close(ctx context.Context) error
}
