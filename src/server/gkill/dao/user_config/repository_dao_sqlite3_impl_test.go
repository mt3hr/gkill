package user_config

import (
	"context"
	"path/filepath"
	"testing"
)

func newTempRepositoryDAO(t *testing.T) RepositoryDAO {
	t.Helper()
	dir := t.TempDir()
	dao, err := NewRepositoryDAOSQLite3Impl(context.Background(), filepath.Join(dir, "repository.db"))
	if err != nil {
		t.Fatalf("failed to create repository dao: %v", err)
	}
	t.Cleanup(func() { dao.Close(context.Background()) })
	return dao
}

func TestRepositoryGetAllEmpty(t *testing.T) {
	dao := newTempRepositoryDAO(t)
	ctx := context.Background()

	all, err := dao.GetAllRepositories(ctx)
	if err != nil {
		t.Fatalf("GetAllRepositories failed: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 repositories on empty DB, got %d", len(all))
	}
}

func TestRepositoryGetByUserDeviceEmpty(t *testing.T) {
	dao := newTempRepositoryDAO(t)
	ctx := context.Background()

	repos, err := dao.GetRepositories(ctx, "nonexistent", "nodevice")
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repositories for nonexistent user, got %d", len(repos))
	}
}

// Note: AddRepository has complex business validation that requires all repository types
// (kmemo, mi, timeis, lantana, kc, nlog, urlog, directory, etc.) to have exactly one
// UseToWrite=true entry per device. Full CRUD testing requires a realistic setup
// with all repository types and is covered in API integration tests.
