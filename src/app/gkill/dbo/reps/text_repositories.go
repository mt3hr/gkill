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

func (t *TextRepositories) FindTexts(ctx context.Context, queryJSON string) ([]*Text, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TextRepositories) Close(ctx context.Context) error {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TextRepositories) GetText(ctx context.Context, id string) (*Text, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TextRepositories) GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TextRepositories) UpdateCache(ctx context.Context) error {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TextRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TextRepositories) GetRepName(ctx context.Context) (string, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TextRepositories) GetTextHistories(ctx context.Context, id string) ([]*Text, error) {
	// ˅
	panic("notImplements")
	// ˄
}

// ˅

// ˄
