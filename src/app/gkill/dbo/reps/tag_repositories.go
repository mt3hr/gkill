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

func (t *TagRepositories) FindTags(ctx context.Context, queryJSON string) []*Tag {
	// ˅

	// ˄
}

func (t *TagRepositories) Close(ctx context.Context) {
	// ˅

	// ˄
}

func (t *TagRepositories) GetTag(ctx context.Context, id string) *Tag {
	// ˅

	// ˄
}

func (t *TagRepositories) GetTagsByTagName(ctx context.Context, tagname string) []*Tag {
	// ˅

	// ˄
}

func (t *TagRepositories) GetTagsByTargetID(ctx context.Context, target_id string) []*Tag {
	// ˅

	// ˄
}

func (t *TagRepositories) UpdateCache(ctx context.Context) {
	// ˅

	// ˄
}

func (t *TagRepositories) GetPath(ctx context.Context, id string) string {
	// ˅

	// ˄
}

func (t *TagRepositories) GetRepName(ctx context.Context) string {
	// ˅

	// ˄
}

func (t *TagRepositories) GetTagHistories(ctx context.Context, id string) []*Tag {
	// ˅

	// ˄
}

// ˅

// ˄
