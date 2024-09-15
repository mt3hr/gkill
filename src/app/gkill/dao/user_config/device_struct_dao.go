package user_config

import "context"

type DeviceStructDAO interface {
	GetAllDeviceStructs(ctx context.Context) ([]*DeviceStruct, error)

	GetDeviceStructs(ctx context.Context, userID string, device string) ([]*DeviceStruct, error)

	AddDeviceStruct(ctx context.Context, deviceStruct *DeviceStruct) (bool, error)

	AddDeviceStructs(ctx context.Context, deviceStructs []*DeviceStruct) (bool, error)

	UpdateDeviceStruct(ctx context.Context, deviceStruct *DeviceStruct) (bool, error)

	DeleteDeviceStruct(ctx context.Context, id string) (bool, error)

	DeleteUsersDeviceStructs(ctx context.Context, userID string) (bool, error)

	Close(ctx context.Context) error
}
