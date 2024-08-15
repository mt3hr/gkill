// ˅
package reps

import "context"

// ˄

type textRepositorySQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (t *textRepositorySQLite3Impl) FindTexts(ctx context.Context, queryJSON string) ([]*Text, error) {
	panic("notImplements")
}

func (t *textRepositorySQLite3Impl) Close(ctx context.Context) error {
	panic("notImplements")
}

func (t *textRepositorySQLite3Impl) GetText(ctx context.Context, id string) (*Text, error) {
	panic("notImplements")
}

func (t *textRepositorySQLite3Impl) GetTextsByTargetID(ctx context.Context, target_id string) ([]*Text, error) {
	panic("notImplements")
}

func (t *textRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	panic("notImplements")
}

func (t *textRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	panic("notImplements")
}

func (t *textRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	panic("notImplements")
}

func (t *textRepositorySQLite3Impl) GetTextHistories(ctx context.Context, id string) ([]*Text, error) {
	panic("notImplements")
}

// ˄
