// ˅
package user_config

import "context"

// ˄

type RepTypeStructDAO interface {
	GetAllRepTypeStructs(ctx context.Context) []*RepTypeStruct

	GetRepTypeStructs(ctx context.Context, userID string, device string) []*RepTypeStruct

	AddRepTypeStruct(ctx context.Context, repTypeStruct *RepTypeStruct) bool

	AddRepTypeStructs(ctx context.Context, repTypeStructs []*RepTypeStruct) bool

	UpdateRepTypeStruct(ctx context.Context, repTypeStruct *RepTypeStruct) bool

	DeleteRepTypeStruct(ctx context.Context, id string) bool

	DeleteUsersRepTypeStructs(ctx context.Context, userID string) bool

	// ˅

	// ˄
}

// ˅

// ˄
