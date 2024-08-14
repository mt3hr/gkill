// ˅
package user_config

import "context"

// ˄

type TagStructDAO interface {
	GetAllTagStructs(ctx context.Context) []*TagStruct

	GetTagStructs(ctx context.Context, userID string, device string) []*TagStruct

	AddTagStruct(ctx context.Context, tagStruct *TagStruct) bool

	AddTagStructs(ctx context.Context, tagStructs []*TagStruct) bool

	UpdateTagStruct(ctx context.Context, tagStruct *TagStruct) bool

	DeleteTagStruct(ctx context.Context, id string) bool

	DeleteUsersTagStructs(ctx context.Context, userID string) bool

	// ˅

	// ˄
}

// ˅

// ˄
