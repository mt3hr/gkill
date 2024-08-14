// ˅
package reps

import "context"

// ˄

type TextRepositories struct {
	// ˅

	// ˄

	textRepositories []TextRepository

	// ˅

	// ˄
}

func (t *TextRepositories) FindTexts(ctx context.Context, queryJSON string) []*Text {
	// ˅

	// ˄
}

func (t *TextRepositories) Close(ctx context.Context) {
	// ˅

	// ˄
}

func (t *TextRepositories) GetText(ctx context.Context, id string) *Text {
	// ˅

	// ˄
}

func (t *TextRepositories) GetTextsByTargetID(ctx context.Context, target_id string) []*Text {
	// ˅

	// ˄
}

func (t *TextRepositories) UpdateCache(ctx context.Context) {
	// ˅

	// ˄
}

func (t *TextRepositories) GetPath(ctx context.Context, id string) string {
	// ˅

	// ˄
}

func (t *TextRepositories) GetRepName(ctx context.Context) string {
	// ˅

	// ˄
}

func (t *TextRepositories) GetTextHistories(ctx context.Context, id string) []*Text {
	// ˅

	// ˄
}

// ˅

// ˄
