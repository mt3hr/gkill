// ˅
package reps

import "context"

// ˄

type TextRepository interface {
	FindTexts(ctx context.Context, queryJSON string) ([]*Text, error)

	Close(ctx context.Context) error

	GetText(ctx context.Context, id string) (*Text, error)

	GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error)

	UpdateCache(ctx context.Context) error

	GetPath(ctx context.Context, id string) (string, error)

	GetRepName(ctx context.Context) (string, error)

	GetTextHistories(ctx context.Context, id string) ([]*Text, error)

	// ˅

	// ˄
}

// ˅

// ˄
