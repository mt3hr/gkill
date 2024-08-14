// ˅
package user_config

import "context"

// ˄

type DeviceStructDAO interface {
	GetAllDeviceStructs(ctx context.Context) []*DeviceStruct

	GetDeviceStructs(ctx context.Context, userID string, device string) []*DeviceStruct

	AddDeviceStruct(ctx context.Context, deviceStruct *DeviceStruct) bool

	AddDeviceStructs(ctx context.Context, deviceStructs []*DeviceStruct) bool

	UpdateDeviceStruct(ctx context.Context, deviceStruct *DeviceStruct) bool

	DeleteDeviceStruct(ctx context.Context, id string) bool

	DeleteUsersDeviceStructs(ctx context.Context, userID string) bool

	// ˅

	// ˄
}

// ˅

// ˄
