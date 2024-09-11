// ˅
package user_config

import "context"

// ˄

type ApplicationConfigDAO interface {
	GetAllApplicationConfigs(ctx context.Context) ([]*ApplicationConfig, error)

	GetApplicationConfig(ctx context.Context, userID string, device string) (*ApplicationConfig, error)

	AddApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) (bool, error)

	UpdateApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) (bool, error)

	DeleteApplicationConfig(ctx context.Context, userID string, device string) (bool, error)

	Close(ctx context.Context) error

	// ˅

	// ˄
}

// ˅

// ˄
