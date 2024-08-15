// ˅
package server_config

import "context"

// ˄

type serverConfigDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (s *serverConfigDAOSQLite3Impl) GetAllServerConfigs(ctx context.Context) ([]*ServerConfig, error) {
	panic("notImplements")
}

func (s *serverConfigDAOSQLite3Impl) GetServerConfig(ctx context.Context, device string) (*ServerConfig, error) {
	panic("notImplements")
}

func (s *serverConfigDAOSQLite3Impl) AddServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error) {
	panic("notImplements")
}

func (s *serverConfigDAOSQLite3Impl) UpdateServerConfig(ctx context.Context, serverConfig *ServerConfig) (bool, error) {
	panic("notImplements")
}

func (s *serverConfigDAOSQLite3Impl) DeleteServerConfig(ctx context.Context, id string) (bool, error) {
	panic("notImplements")
}

// ˄
