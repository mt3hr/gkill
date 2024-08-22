// ˅
package user_config

import "context"

// ˄

type repTypeStructDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (r *repTypeStructDAOSQLite3Impl) GetAllRepTypeStructs(ctx context.Context) ([]*RepTypeStruct, error) {
	panic("notImplements")
}

func (r *repTypeStructDAOSQLite3Impl) GetRepTypeStructs(ctx context.Context, userID string, device string) ([]*RepTypeStruct, error) {
	panic("notImplements")
}

func (r *repTypeStructDAOSQLite3Impl) AddRepTypeStruct(ctx context.Context, repTypeStruct *RepTypeStruct) (bool, error) {
	panic("notImplements")
}

func (r *repTypeStructDAOSQLite3Impl) AddRepTypeStructs(ctx context.Context, repTypeStructs []*RepTypeStruct) (bool, error) {
	panic("notImplements")
}

func (r *repTypeStructDAOSQLite3Impl) UpdateRepTypeStruct(ctx context.Context, repTypeStruct *RepTypeStruct) (bool, error) {
	panic("notImplements")
}

func (r *repTypeStructDAOSQLite3Impl) DeleteRepTypeStruct(ctx context.Context, id string) (bool, error) {
	panic("notImplements")
}

func (r *repTypeStructDAOSQLite3Impl) DeleteUsersRepTypeStructs(ctx context.Context, userID string) (bool, error) {
	panic("notImplements")
}

// ˄
