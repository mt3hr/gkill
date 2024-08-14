// ˅
package user_config

import "context"

// ˄

type RepTypeStructDAO interface {
	GetAllRepTypeStructs(ctx context.Context) ([]*RepTypeStruct, error)

	GetRepTypeStructs(ctx context.Context, userID string, device string) ([]*RepTypeStruct, error)

	AddRepTypeStruct(ctx context.Context, repTypeStruct *RepTypeStruct) (bool, error)

	AddRepTypeStructs(ctx context.Context, repTypeStructs []*RepTypeStruct) (bool, error)

	UpdateRepTypeStruct(ctx context.Context, repTypeStruct *RepTypeStruct) (bool, error)

	DeleteRepTypeStruct(ctx context.Context, id string) (bool, error)

	DeleteUsersRepTypeStructs(ctx context.Context, userID string) (bool, error)

	// ˅

	// ˄
}

// ˅

// ˄
