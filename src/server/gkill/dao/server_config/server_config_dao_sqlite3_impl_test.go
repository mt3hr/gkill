package server_config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTempServerConfigDAO(t *testing.T) ServerConfigDAO {
	t.Helper()
	dir, err := os.MkdirTemp("", "server_config_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dao, err := NewServerConfigDAOSQLite3Impl(context.Background(), filepath.Join(dir, "server_config.db"))
	if err != nil {
		t.Fatalf("failed to create server config dao: %v", err)
	}
	t.Cleanup(func() {
		dao.Close(context.Background())
		os.RemoveAll(dir) // best-effort cleanup, may fail on Windows due to WAL lock
	})
	return dao
}

func makeTestServerConfig(device string) *ServerConfig {
	return &ServerConfig{
		EnableThisDevice:     true,
		Device:               device,
		IsLocalOnlyAccess:    false,
		Address:              ":9999",
		EnableTLS:            false,
		URLogTimeout:         10 * time.Second,
		UploadSizeLimitMonth: -1,
		UserDataDirectory:    "",
	}
}

func TestServerConfigAddAndGet(t *testing.T) {
	dao := newTempServerConfigDAO(t)
	ctx := context.Background()

	cfg := makeTestServerConfig("test-device")
	ok, err := dao.AddServerConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("AddServerConfig failed: %v", err)
	}
	if !ok {
		t.Fatal("AddServerConfig returned false")
	}

	got, err := dao.GetServerConfig(ctx, "test-device")
	if err != nil {
		t.Fatalf("GetServerConfig failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetServerConfig returned nil")
	}
	if got.Device != "test-device" {
		t.Errorf("Device = %q, want %q", got.Device, "test-device")
	}
	if got.Address != ":9999" {
		t.Errorf("Address = %q, want %q", got.Address, ":9999")
	}
}

func TestServerConfigGetAll(t *testing.T) {
	dao := newTempServerConfigDAO(t)
	ctx := context.Background()

	for _, device := range []string{"device-a", "device-b"} {
		cfg := makeTestServerConfig(device)
		if _, err := dao.AddServerConfig(ctx, cfg); err != nil {
			t.Fatalf("AddServerConfig failed: %v", err)
		}
	}

	all, err := dao.GetAllServerConfigs(ctx)
	if err != nil {
		t.Fatalf("GetAllServerConfigs failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 configs, got %d", len(all))
	}
}

func TestServerConfigUpdate(t *testing.T) {
	dao := newTempServerConfigDAO(t)
	ctx := context.Background()

	cfg := makeTestServerConfig("dev-upd")
	if _, err := dao.AddServerConfig(ctx, cfg); err != nil {
		t.Fatalf("AddServerConfig failed: %v", err)
	}

	cfg.Address = ":8080"
	ok, err := dao.UpdateServerConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("UpdateServerConfig failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateServerConfig returned false")
	}

	got, err := dao.GetServerConfig(ctx, "dev-upd")
	if err != nil {
		t.Fatalf("GetServerConfig failed: %v", err)
	}
	if got.Address != ":8080" {
		t.Errorf("Address = %q, want %q", got.Address, ":8080")
	}
}

func TestServerConfigDelete(t *testing.T) {
	dao := newTempServerConfigDAO(t)
	ctx := context.Background()

	cfg := makeTestServerConfig("dev-del")
	if _, err := dao.AddServerConfig(ctx, cfg); err != nil {
		t.Fatalf("AddServerConfig failed: %v", err)
	}

	ok, err := dao.DeleteServerConfig(ctx, "dev-del")
	if err != nil {
		t.Fatalf("DeleteServerConfig failed: %v", err)
	}
	if !ok {
		t.Fatal("DeleteServerConfig returned false")
	}

	got, err := dao.GetServerConfig(ctx, "dev-del")
	if err != nil {
		t.Fatalf("GetServerConfig after delete should not error: %v", err)
	}
	if got == nil {
		t.Fatal("GetServerConfig returned nil")
	}
	if got.EnableThisDevice {
		t.Error("deleted device should return default EnableThisDevice=false")
	}
}

// GetDefaultServerConfig（行が無い端末のフォールバック）と定数が一致すること。
// 初回起動ブロックも同じ定数を引くので、既定の待受が2系統に分かれて食い違う事故を防ぐ（ADR-0708）。
func TestGetDefaultServerConfigUsesListenDefaults(t *testing.T) {
	got := GetDefaultServerConfig("any-device")
	if got.Address != DefaultListenAddress {
		t.Errorf("Address = %q, want %q", got.Address, DefaultListenAddress)
	}
	if got.IsLocalOnlyAccess != DefaultIsLocalOnlyAccess {
		t.Errorf("IsLocalOnlyAccess = %v, want %v", got.IsLocalOnlyAccess, DefaultIsLocalOnlyAccess)
	}
	if got.EnableTLS {
		t.Error("EnableTLS = true。既定では証明書が無いので TLS は無効であること")
	}
}

// 行が無い端末を GetServerConfig すると、SQL のフォールバックではなく GetDefaultServerConfig が返ること。
// どちらの経路でも既定はループバック限定になる。
func TestGetServerConfigUnknownDeviceReturnsListenDefaults(t *testing.T) {
	dao := newTempServerConfigDAO(t)
	got, err := dao.GetServerConfig(context.Background(), "unknown-device")
	if err != nil {
		t.Fatalf("GetServerConfig failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetServerConfig returned nil")
	}
	if got.Address != DefaultListenAddress {
		t.Errorf("Address = %q, want %q", got.Address, DefaultListenAddress)
	}
	if got.IsLocalOnlyAccess != DefaultIsLocalOnlyAccess {
		t.Errorf("IsLocalOnlyAccess = %v, want %v", got.IsLocalOnlyAccess, DefaultIsLocalOnlyAccess)
	}
}
