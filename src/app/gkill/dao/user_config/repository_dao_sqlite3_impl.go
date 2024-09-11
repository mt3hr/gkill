// ˅
package user_config

import "context"

// ˄

type repositoryDAOSQLite3Impl struct {
	// ˅

	// ˄
}

// ˅
func (r *repositoryDAOSQLite3Impl) GetAllRepositories(ctx context.Context) ([]*Repository, error) {
	panic("notImplements")
}

func (r *repositoryDAOSQLite3Impl) GetRepositories(ctx context.Context, userID string, device string) ([]*Repository, error) {
	panic("notImplements")
}

func (r *repositoryDAOSQLite3Impl) AddRepository(ctx context.Context, repository *Repository) (bool, error) {
	panic("notImplements")
}

func (r *repositoryDAOSQLite3Impl) AddRepositories(ctx context.Context, repository []*Repository) (bool, error) {
	panic("notImplements")
}

func (r *repositoryDAOSQLite3Impl) UpdateRepository(ctx context.Context, repository *Repository) (bool, error) {
	panic("notImplements")
}

func (r *repositoryDAOSQLite3Impl) DeleteRepository(ctx context.Context, id string) (bool, error) {
	panic("notImplements")
}

func (r *repositoryDAOSQLite3Impl) DeleteAllRepositoriesByUser(ctx context.Context, userID string, device string) (bool, error) {
	panic("notImplements")
}

// ˄
