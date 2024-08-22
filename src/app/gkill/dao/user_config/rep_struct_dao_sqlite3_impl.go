// ˅
package user_config

import "context"

// ˄

type repStructDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (r *repStructDAOSQLite3Impl) GetAllRepStructs(ctx context.Context) ([]*RepStruct, error) {
	panic("notImplements")
}

func (r *repStructDAOSQLite3Impl) GetRepStructs(ctx context.Context, userID string, device string) ([]*RepStruct, error) {
	panic("notImplements")
}

func (r *repStructDAOSQLite3Impl) AddRepStruct(ctx context.Context, repStruct *RepStruct) (bool, error) {
	panic("notImplements")
}

func (r *repStructDAOSQLite3Impl) AddRepStructs(ctx context.Context, repStructs []*RepStruct) (bool, error) {
	panic("notImplements")
}

func (r *repStructDAOSQLite3Impl) UpdateRepStruct(ctx context.Context, repStruct *RepStruct) (bool, error) {
	panic("notImplements")
}

func (r *repStructDAOSQLite3Impl) DeleteRepStruct(ctx context.Context, id string) (bool, error) {
	panic("notImplements")
}

func (r *repStructDAOSQLite3Impl) DeleteUsersRepStructs(ctx context.Context, userID string) (bool, error) {
	panic("notImplements")
}

// ˄
