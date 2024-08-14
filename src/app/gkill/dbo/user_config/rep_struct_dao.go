// ˅
package user_config

import "context"

// ˄

type RepStructDAO interface {
	GetAllRepStructs(ctx context.Context) ([]*RepStruct, error)

	GetRepStructs(ctx context.Context, userID string, device string) ([]*RepStruct, error)

	AddRepStruct(ctx context.Context, repStruct *RepStruct) (bool, error)

	AddRepStructs(ctx context.Context, repStructs []*RepStruct) (bool, error)

	UpdateRepStruct(ctx context.Context, repStruct *RepStruct) (bool, error)

	DeleteRepStruct(ctx context.Context, id string) (bool, error)

	DeleteUsersRepStructs(ctx context.Context, userID string) (bool, error)

	// ˅

	// ˄
}

// ˅

// ˄
