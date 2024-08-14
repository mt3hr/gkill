// ˅
package user_config

import "context"

// ˄

type RepStructDAO interface {
	GetAllRepStructs(ctx context.Context) []*RepStruct

	GetRepStructs(ctx context.Context, userID string, device string) []*RepStruct

	AddRepStruct(ctx context.Context, repStruct *RepStruct) bool

	AddRepStructs(ctx context.Context, repStructs []*RepStruct) bool

	UpdateRepStruct(ctx context.Context, repStruct *RepStruct) bool

	DeleteRepStruct(ctx context.Context, id string) bool

	DeleteUsersRepStructs(ctx context.Context, userID string) bool

	// ˅

	// ˄
}

// ˅

// ˄
