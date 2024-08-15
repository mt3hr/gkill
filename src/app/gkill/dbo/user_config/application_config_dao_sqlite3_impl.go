// ˅
package user_config

import "context"

// ˄

type applicationConfigDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (a *applicationConfigDAOSQLite3Impl) GetAllApplicationConfigs(ctx context.Context) ([]*ApplicationConfig, error) {
	panic("notImplements")
}

func (a *applicationConfigDAOSQLite3Impl) GetApplicationConfig(ctx context.Context, userID string, device string) (*ApplicationConfig, error) {
	panic("notImplements")
}

func (a *applicationConfigDAOSQLite3Impl) AddApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) (bool, error) {
	panic("notImplements")
}

func (a *applicationConfigDAOSQLite3Impl) UpdateApplicationConfig(ctx context.Context, applicationConfig *ApplicationConfig) (bool, error) {
	panic("notImplements")
}

func (a *applicationConfigDAOSQLite3Impl) DeleteApplicationConfig(ctx context.Context, userID string, device string) (bool, error) {
	panic("notImplements")
}

// ˄
