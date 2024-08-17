// ˅
package reps

import "context"

// ˄

type TagRepositories struct {
	// ˅

	// ˄

	tagRepositories []TagRepository

	// ˅

	// ˄
}

func (t *TagRepositories) FindTags(ctx context.Context, queryJSON string) ([]*Tag, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TagRepositories) Close(ctx context.Context) error {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TagRepositories) GetTag(ctx context.Context, id string) (*Tag, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TagRepositories) GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TagRepositories) GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TagRepositories) UpdateCache(ctx context.Context) error {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TagRepositories) GetPath(ctx context.Context, id string) (string, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TagRepositories) GetRepName(ctx context.Context) (string, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TagRepositories) GetTagHistories(ctx context.Context, id string) ([]*Tag, error) {
	// ˅
	panic("notImplements")
	// ˄
}

func (t *TagRepositories) AddTagInfo(ctx context.Context, tag *Tag) error {
	// ˅
	panic("notImplements")
	// ˄
}

// ˅

// ˄
