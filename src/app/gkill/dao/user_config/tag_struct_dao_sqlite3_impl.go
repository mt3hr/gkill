// ˅
package user_config

import "context"

// ˄

type tagStructDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (t *tagStructDAOSQLite3Impl) GetAllTagStructs(ctx context.Context) ([]*TagStruct, error) {
	panic("notImplements")
}

func (t *tagStructDAOSQLite3Impl) GetTagStructs(ctx context.Context, userID string, device string) ([]*TagStruct, error) {
	panic("notImplements")
}

func (t *tagStructDAOSQLite3Impl) AddTagStruct(ctx context.Context, tagStruct *TagStruct) (bool, error) {
	panic("notImplements")
}

func (t *tagStructDAOSQLite3Impl) AddTagStructs(ctx context.Context, tagStructs []*TagStruct) (bool, error) {
	panic("notImplements")
}

func (t *tagStructDAOSQLite3Impl) UpdateTagStruct(ctx context.Context, tagStruct *TagStruct) (bool, error) {
	panic("notImplements")
}

func (t *tagStructDAOSQLite3Impl) DeleteTagStruct(ctx context.Context, id string) (bool, error) {
	panic("notImplements")
}

func (t *tagStructDAOSQLite3Impl) DeleteUsersTagStructs(ctx context.Context, userID string) (bool, error) {
	panic("notImplements")
}

// ˄
