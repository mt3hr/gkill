// ˅
package reps

import "context"

// ˄

type tagRepositorySQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (t *tagRepositorySQLite3Impl) FindTags(ctx context.Context, queryJSON string) ([]*Tag, error) {
	panic("notImplements")
}

func (t *tagRepositorySQLite3Impl) Close(ctx context.Context) error {
	panic("notImplements")
}

func (t *tagRepositorySQLite3Impl) GetTag(ctx context.Context, id string) (*Tag, error) {
	panic("notImplements")
}

func (t *tagRepositorySQLite3Impl) GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error) {
	panic("notImplements")
}

func (t *tagRepositorySQLite3Impl) GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error) {
	panic("notImplements")
}

func (t *tagRepositorySQLite3Impl) UpdateCache(ctx context.Context) error {
	panic("notImplements")
}

func (t *tagRepositorySQLite3Impl) GetPath(ctx context.Context, id string) (string, error) {
	panic("notImplements")
}

func (t *tagRepositorySQLite3Impl) GetRepName(ctx context.Context) (string, error) {
	panic("notImplements")
}

func (t *tagRepositorySQLite3Impl) GetTagHistories(ctx context.Context, id string) ([]*Tag, error) {
	panic("notImplements")
}

// ˄
