package user_config

import (
	"context"
	"path/filepath"
	"testing"
)

func newTempApplicationConfigDAO(t *testing.T) ApplicationConfigDAO {
	t.Helper()
	dir := t.TempDir()
	dao, err := NewApplicationConfigDAOSQLite3Impl(context.Background(), filepath.Join(dir, "app_config.db"))
	if err != nil {
		t.Fatalf("failed to create application config dao: %v", err)
	}
	t.Cleanup(func() { dao.Close(context.Background()) })
	return dao
}

func TestApplicationConfigAddDefault(t *testing.T) {
	dao := newTempApplicationConfigDAO(t)
	ctx := context.Background()

	ok, err := dao.AddDefaultApplicationConfig(ctx, "user1", "device1")
	if err != nil {
		t.Fatalf("AddDefaultApplicationConfig failed: %v", err)
	}
	if !ok {
		t.Fatal("AddDefaultApplicationConfig returned false")
	}

	got, err := dao.GetApplicationConfig(ctx, "user1", "device1")
	if err != nil {
		t.Fatalf("GetApplicationConfig failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetApplicationConfig returned nil")
	}
	if got.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "user1")
	}
	if got.Device != "device1" {
		t.Errorf("Device = %q, want %q", got.Device, "device1")
	}
}

func TestApplicationConfigAddAndGet(t *testing.T) {
	dao := newTempApplicationConfigDAO(t)
	ctx := context.Background()

	cfg := GetDefaultApplicationConfig("user2", "device2")
	cfg.UseDarkTheme = true
	cfg.MiDefaultBoard = "仕事"

	ok, err := dao.AddApplicationConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("AddApplicationConfig failed: %v", err)
	}
	if !ok {
		t.Fatal("AddApplicationConfig returned false")
	}

	got, err := dao.GetApplicationConfig(ctx, "user2", "device2")
	if err != nil {
		t.Fatalf("GetApplicationConfig failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetApplicationConfig returned nil")
	}
	if !got.UseDarkTheme {
		t.Error("UseDarkTheme should be true")
	}
	if got.MiDefaultBoard != "仕事" {
		t.Errorf("MiDefaultBoard = %q, want %q", got.MiDefaultBoard, "仕事")
	}
}

func TestApplicationConfigUpdate(t *testing.T) {
	dao := newTempApplicationConfigDAO(t)
	ctx := context.Background()

	if _, err := dao.AddDefaultApplicationConfig(ctx, "user-upd", "dev-upd"); err != nil {
		t.Fatalf("AddDefaultApplicationConfig failed: %v", err)
	}

	cfg, err := dao.GetApplicationConfig(ctx, "user-upd", "dev-upd")
	if err != nil {
		t.Fatalf("GetApplicationConfig failed: %v", err)
	}

	cfg.UseDarkTheme = true
	ok, err := dao.UpdateApplicationConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("UpdateApplicationConfig failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateApplicationConfig returned false")
	}

	got, err := dao.GetApplicationConfig(ctx, "user-upd", "dev-upd")
	if err != nil {
		t.Fatalf("GetApplicationConfig after update failed: %v", err)
	}
	if !got.UseDarkTheme {
		t.Error("UseDarkTheme should be true after update")
	}
}

func TestApplicationConfigDelete(t *testing.T) {
	dao := newTempApplicationConfigDAO(t)
	ctx := context.Background()

	if _, err := dao.AddDefaultApplicationConfig(ctx, "user-del", "dev-del"); err != nil {
		t.Fatalf("AddDefaultApplicationConfig failed: %v", err)
	}

	ok, err := dao.DeleteApplicationConfig(ctx, "user-del", "dev-del")
	if err != nil {
		t.Fatalf("DeleteApplicationConfig failed: %v", err)
	}
	if !ok {
		t.Fatal("DeleteApplicationConfig returned false")
	}

	// After delete, GetAllApplicationConfigs should have fewer results
	all, err := dao.GetAllApplicationConfigs(ctx)
	if err != nil {
		t.Fatalf("GetAllApplicationConfigs failed: %v", err)
	}
	for _, cfg := range all {
		if cfg.UserID == "user-del" && cfg.Device == "dev-del" {
			t.Error("deleted config still present in GetAll")
		}
	}
}

func TestApplicationConfigGetAll(t *testing.T) {
	dao := newTempApplicationConfigDAO(t)
	ctx := context.Background()

	// Get initial count (may have defaults)
	initial, err := dao.GetAllApplicationConfigs(ctx)
	if err != nil {
		t.Fatalf("GetAllApplicationConfigs (initial) failed: %v", err)
	}
	initialCount := len(initial)

	if _, err := dao.AddDefaultApplicationConfig(ctx, "user-getall-1", "dev-getall-1"); err != nil {
		t.Fatalf("AddDefaultApplicationConfig failed: %v", err)
	}

	all, err := dao.GetAllApplicationConfigs(ctx)
	if err != nil {
		t.Fatalf("GetAllApplicationConfigs failed: %v", err)
	}
	if len(all) <= initialCount {
		t.Errorf("expected more configs after add, initial=%d, got=%d", initialCount, len(all))
	}
}
