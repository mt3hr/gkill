package user_config

import (
	"context"
	"encoding/json"
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

func TestApplicationConfigPlaingTimeIsJSONDataRoundTrip(t *testing.T) {
	dao := newTempApplicationConfigDAO(t)
	ctx := context.Background()

	if def := GetDefaultApplicationConfig("user-pt", "dev-pt"); def.PlaingTimeIsJSONData == nil {
		t.Fatal("GetDefaultApplicationConfig().PlaingTimeIsJSONData should not be nil")
	}

	if _, err := dao.AddDefaultApplicationConfig(ctx, "user-pt", "dev-pt"); err != nil {
		t.Fatalf("AddDefaultApplicationConfig failed: %v", err)
	}

	cfg, err := dao.GetApplicationConfig(ctx, "user-pt", "dev-pt")
	if err != nil {
		t.Fatalf("GetApplicationConfig failed: %v", err)
	}

	saved := json.RawMessage(`{"plaing_timeis_find_kyou_query":{"reps":["rep1"]}}`)
	cfg.PlaingTimeIsJSONData = &saved
	ok, err := dao.UpdateApplicationConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("UpdateApplicationConfig failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateApplicationConfig returned false")
	}

	got, err := dao.GetApplicationConfig(ctx, "user-pt", "dev-pt")
	if err != nil {
		t.Fatalf("GetApplicationConfig after update failed: %v", err)
	}
	if got.PlaingTimeIsJSONData == nil {
		t.Fatal("PlaingTimeIsJSONData should not be nil after update")
	}
	if string(*got.PlaingTimeIsJSONData) != string(saved) {
		t.Errorf("PlaingTimeIsJSONData = %s, want %s", string(*got.PlaingTimeIsJSONData), string(saved))
	}
}

func TestApplicationConfigSavedFindQueryJSONDataRoundTrip(t *testing.T) {
	dao := newTempApplicationConfigDAO(t)
	ctx := context.Background()

	if def := GetDefaultApplicationConfig("user-sfq", "dev-sfq"); def.SavedFindQueryJSONData == nil {
		t.Fatal("GetDefaultApplicationConfig().SavedFindQueryJSONData should not be nil")
	}

	if _, err := dao.AddDefaultApplicationConfig(ctx, "user-sfq", "dev-sfq"); err != nil {
		t.Fatalf("AddDefaultApplicationConfig failed: %v", err)
	}

	cfg, err := dao.GetApplicationConfig(ctx, "user-sfq", "dev-sfq")
	if err != nil {
		t.Fatalf("GetApplicationConfig failed: %v", err)
	}

	saved := json.RawMessage(`{"saved_rykv_find_kyou_querys":[{"id":"id1","title":"仕事","find_kyou_query":{"words":["メモ"]}}],"saved_mi_find_kyou_querys":[]}`)
	cfg.SavedFindQueryJSONData = &saved
	ok, err := dao.UpdateApplicationConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("UpdateApplicationConfig failed: %v", err)
	}
	if !ok {
		t.Fatal("UpdateApplicationConfig returned false")
	}

	got, err := dao.GetApplicationConfig(ctx, "user-sfq", "dev-sfq")
	if err != nil {
		t.Fatalf("GetApplicationConfig after update failed: %v", err)
	}
	if got.SavedFindQueryJSONData == nil {
		t.Fatal("SavedFindQueryJSONData should not be nil after update")
	}
	if string(*got.SavedFindQueryJSONData) != string(saved) {
		t.Errorf("SavedFindQueryJSONData = %s, want %s", string(*got.SavedFindQueryJSONData), string(saved))
	}

	// DEVICE='ALL' で保存されるため、別デバイスの設定行を作った後でも同じ値が見えること
	cfgOtherDevice := got
	cfgOtherDevice.Device = "dev-sfq-other"
	if _, err := dao.UpdateApplicationConfig(ctx, cfgOtherDevice); err != nil {
		t.Fatalf("UpdateApplicationConfig (other device) failed: %v", err)
	}
	gotOther, err := dao.GetApplicationConfig(ctx, "user-sfq", "dev-sfq-other")
	if err != nil {
		t.Fatalf("GetApplicationConfig (other device) failed: %v", err)
	}
	if gotOther.SavedFindQueryJSONData == nil || string(*gotOther.SavedFindQueryJSONData) != string(saved) {
		t.Error("SavedFindQueryJSONData should be shared across devices (DEVICE='ALL')")
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
