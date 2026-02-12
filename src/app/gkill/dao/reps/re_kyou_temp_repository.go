package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
)

type ReKyouTempRepository interface {
	FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)

	GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error)

	GetKyouHistories(ctx context.Context, id string) ([]Kyou, error)

	GetPath(ctx context.Context, id string) (string, error)

	UpdateCache(ctx context.Context) error

	GetRepName(ctx context.Context) (string, error)

	Close(ctx context.Context) error

	FindReKyou(ctx context.Context, query *find.FindQuery) ([]ReKyou, error)

	GetReKyou(ctx context.Context, id string, updateTime *time.Time) (*ReKyou, error)

	GetReKyouHistories(ctx context.Context, id string) ([]ReKyou, error)

	AddReKyouInfo(ctx context.Context, rekyou ReKyou, txID string, userID string, device string) error

	GetReKyousAllLatest(ctx context.Context) ([]ReKyou, error)

	GetRepositoriesWithoutReKyouRep(ctx context.Context) (*GkillRepositories, error)

	GetKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]Kyou, error)

	GetReKyousByTXID(ctx context.Context, txID string, userID string, device string) ([]ReKyou, error)

	DeleteByTXID(ctx context.Context, txID string, userID string, device string) error

	UnWrapTyped() ([]ReKyouTempRepository, error)

	UnWrap() ([]Repository, error)
}
