package reps

import "context"

type TagRepository interface {
	FindTags(ctx context.Context, queryJSON string) ([]*Tag, error)

	Close(ctx context.Context) error

	GetTag(ctx context.Context, id string) (*Tag, error)

	GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error)

	GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error)

	UpdateCache(ctx context.Context) error

	GetPath(ctx context.Context, id string) (string, error)

	GetRepName(ctx context.Context) (string, error)

	GetTagHistories(ctx context.Context, id string) ([]*Tag, error)

	AddTagInfo(ctx context.Context, tag *Tag) error

	GetAllTagNames(ctx context.Context) ([]string, error)
}
