// ˅
package reps

import "context"

// ˄

type TextRepository interface {
	FindTexts(ctx context.Context, queryJSON string) []*Text

	Close(ctx context.Context)

	GetText(ctx context.Context, id string) *Text

	GetTextsByTargetID(ctx context.Context, target_id string) []*Text

	UpdateCache(ctx context.Context)

	GetPath(ctx context.Context, id string) string

	GetRepName(ctx context.Context) string

	GetTextHistories(ctx context.Context, id string) []*Text

	// ˅

	// ˄
}

// ˅

// ˄
