// ˅
package user_config

import "context"

// ˄

type deviceStructDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (d *deviceStructDAOSQLite3Impl) GetAllDeviceStructs(ctx context.Context) ([]*DeviceStruct, error) {
	panic("notImplements")
}

func (d *deviceStructDAOSQLite3Impl) GetDeviceStructs(ctx context.Context, userID string, device string) ([]*DeviceStruct, error) {
	panic("notImplements")
}

func (d *deviceStructDAOSQLite3Impl) AddDeviceStruct(ctx context.Context, deviceStruct *DeviceStruct) (bool, error) {
	panic("notImplements")
}

func (d *deviceStructDAOSQLite3Impl) AddDeviceStructs(ctx context.Context, deviceStructs []*DeviceStruct) (bool, error) {
	panic("notImplements")
}

func (d *deviceStructDAOSQLite3Impl) UpdateDeviceStruct(ctx context.Context, deviceStruct *DeviceStruct) (bool, error) {
	panic("notImplements")
}

func (d *deviceStructDAOSQLite3Impl) DeleteDeviceStruct(ctx context.Context, id string) (bool, error) {
	panic("notImplements")
}

func (d *deviceStructDAOSQLite3Impl) DeleteUsersDeviceStructs(ctx context.Context, userID string) (bool, error) {
	panic("notImplements")
}

// ˄
