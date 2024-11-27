package reps

import (
	"context"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
)

type TextRepository interface {
	FindTexts(ctx context.Context, query *find.FindQuery) ([]*Text, error)

	Close(ctx context.Context) error

	GetText(ctx context.Context, id string) (*Text, error)

	GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error)

	UpdateCache(ctx context.Context) error

	GetPath(ctx context.Context, id string) (string, error)

	GetRepName(ctx context.Context) (string, error)

	GetTextHistories(ctx context.Context, id string) ([]*Text, error)

	AddTextInfo(ctx context.Context, text *Text) error
}
