// ˅
package reps

import "context"

// ˄

type TagRepository interface {
	FindTags(ctx context.Context, queryJSON string) []*Tag

	Close(ctx context.Context)

	GetTag(ctx context.Context, id string) *Tag

	GetTagsByTagName(ctx context.Context, tagname string) []*Tag

	GetTagsByTargetID(ctx context.Context, target_id string) []*Tag

	UpdateCache(ctx context.Context)

	GetPath(ctx context.Context, id string) string

	GetRepName(ctx context.Context) string

	GetTagHistories(ctx context.Context, id string) []*Tag

	// ˅

	// ˄
}

// ˅

// ˄
