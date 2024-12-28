package reps

import (
	"context"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/api/find"
)

type TagRepository interface {
	FindTags(ctx context.Context, query *find.FindQuery) ([]*Tag, error)

	Close(ctx context.Context) error

	GetTag(ctx context.Context, id string, updateTime *time.Time) (*Tag, error)

	GetTagsByTagName(ctx context.Context, tagname string) ([]*Tag, error)

	GetTagsByTargetID(ctx context.Context, target_id string) ([]*Tag, error)

	UpdateCache(ctx context.Context) error

	GetPath(ctx context.Context, id string) (string, error)

	GetRepName(ctx context.Context) (string, error)

	GetTagHistories(ctx context.Context, id string) ([]*Tag, error)

	AddTagInfo(ctx context.Context, tag *Tag) error

	GetAllTagNames(ctx context.Context) ([]string, error)

	GetAllTags(ctx context.Context) ([]*Tag, error)
}
