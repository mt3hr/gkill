// ˅
package user_config

import "context"

// ˄

type RepositoryDAO interface {
	GetAllRepositories(ctx context.Context) ([]*Repository, error)

	GetRepositories(ctx context.Context, userID string, device string) ([]*Repository, error)

	AddRepository(ctx context.Context, repository *Repository) (bool, error)

	AddRepositories(ctx context.Context, repository []*Repository) (bool, error)

	UpdateRepository(ctx context.Context, repository *Repository) (bool, error)

	DeleteRepository(ctx context.Context, id string) (bool, error)

	DeleteAllRepositoriesByUser(ctx context.Context, userID string, device string) (bool, error)

	Close(ctx context.Context) error

	// ˅

	// ˄
}

// ˅

// ˄
