// ˅
package mi_share_info

import "context"

// ˄

type miShareInfoDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (m *miShareInfoDAOSQLite3Impl) GetAllMiShareInfos(ctx context.Context) ([]*MiShareInfo, error) {
	panic("notImplements")
}

func (m *miShareInfoDAOSQLite3Impl) GetMiShareInfos(ctx context.Context, userID string, device string) ([]*MiShareInfo, error) {
	panic("notImplements")
}

func (m *miShareInfoDAOSQLite3Impl) AddMiShareInfo(ctx context.Context, miShareInfo *MiShareInfo) (bool, error) {
	panic("notImplements")
}

func (m *miShareInfoDAOSQLite3Impl) UpdateMiShareInfo(ctx context.Context, miShareInfo *MiShareInfo) (bool, error) {
	panic("notImplements")
}

func (m *miShareInfoDAOSQLite3Impl) DeleteMiShareInfo(ctx context.Context, id string) (bool, error) {
	panic("notImplements")
}

// ˄
