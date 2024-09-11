// ˅
package user_config

import "context"

// ˄

type TagStructDAO interface {
	GetAllTagStructs(ctx context.Context) ([]*TagStruct, error)

	GetTagStructs(ctx context.Context, userID string, device string) ([]*TagStruct, error)

	AddTagStruct(ctx context.Context, tagStruct *TagStruct) (bool, error)

	AddTagStructs(ctx context.Context, tagStructs []*TagStruct) (bool, error)

	UpdateTagStruct(ctx context.Context, tagStruct *TagStruct) (bool, error)

	DeleteTagStruct(ctx context.Context, id string) (bool, error)

	DeleteUsersTagStructs(ctx context.Context, userID string) (bool, error)

	Close(ctx context.Context) error

	// ˅

	// ˄
}

// ˅

// ˄
