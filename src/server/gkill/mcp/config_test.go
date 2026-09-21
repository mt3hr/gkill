package mcp

// config.go（設定ファイルの生成・読み込み・優先順位）の検査。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func envMap(values map[string]string) Getenv {
	return func(key string) string { return values[key] }
}

func TestLoadOrCreateConfig(t *testing.T) {
	t.Run("creates the file with defaults when it does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "configs", ConfigFileName)
		cfg, created, err := LoadOrCreateConfig(path)
		expectNoError(t, err)
		expectTrue(t, created, "not created")
		expectEqual(t, cfg.Gkill.BaseURL, DefaultGkillBaseURL)
		expectEqual(t, cfg.LogLevel, "access")
		expectEqual(t, cfg.Transport, "stdio")
		expectEqual(t, cfg.Servers["read"].Port, 8808)
		expectEqual(t, cfg.Servers["write"].Port, 8809)
		expectEqual(t, cfg.Servers["readwrite"].Port, 8810)
		data, err := os.ReadFile(path)
		expectNoError(t, err)
		mustContain(t, string(data), `"base_url": "http://127.0.0.1:9999"`)
		mustContain(t, string(data), `"log_level": "access"`)
		// 2回目は生成しない
		_, createdAgain, err := LoadOrCreateConfig(path)
		expectNoError(t, err)
		expectTrue(t, !createdAgain, "created twice")
	})

	t.Run("does not rewrite an existing file and fills missing keys with defaults", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ConfigFileName)
		expectNoError(t, os.WriteFile(path, []byte(`{"gkill":{"base_url":"https://gkill.example.test"},"log_level":"debug"}`), 0o600))
		cfg, created, err := LoadOrCreateConfig(path)
		expectNoError(t, err)
		expectTrue(t, !created, "created over an existing file")
		expectEqual(t, cfg.Gkill.BaseURL, "https://gkill.example.test")
		expectEqual(t, cfg.Gkill.Locale, "ja")
		expectEqual(t, cfg.LogLevel, "debug")
		expectEqual(t, cfg.Servers["read"].Port, 8808)
		data, _ := os.ReadFile(path)
		expectEqual(t, string(data), `{"gkill":{"base_url":"https://gkill.example.test"},"log_level":"debug"}`)
	})

	t.Run("a broken file stops the startup instead of falling back to defaults", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), ConfigFileName)
		expectNoError(t, os.WriteFile(path, []byte(`{"gkill": `), 0o600))
		_, _, err := LoadOrCreateConfig(path)
		expectErrorContains(t, err, "parse mcp config")
	})
}

func TestResolveSettings(t *testing.T) {
	t.Run("defaults come from the file when no env or flag is given", func(t *testing.T) {
		settings, err := ResolveSettings(DefaultConfig(), "readwrite", FlagOverrides{}, envMap(nil))
		expectNoError(t, err)
		expectEqual(t, settings.Transport, "stdio")
		expectEqual(t, settings.LogLevelName, "access")
		expectEqual(t, settings.Port, 8810)
		expectEqual(t, settings.BindAddr, "0.0.0.0")
		expectEqual(t, settings.Client.BaseURL, DefaultGkillBaseURL)
		expectEqual(t, settings.Client.Locale, "ja")
		expectEqual(t, int64(settings.Client.FetchTimeout), int64(120*time.Second))
		expectEqual(t, settings.MaxFileBytes, int64(8*1024*1024))
		expectEqual(t, int64(settings.FileLinkTTL), int64(time.Hour))
		expectEqual(t, settings.OAuthIssuer, "")
	})

	t.Run("environment variables override the file", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Gkill.User = "fileuser"
		settings, err := ResolveSettings(cfg, "read", FlagOverrides{}, envMap(map[string]string{
			"GKILL_BASE_URL":             "https://127.0.0.1:9999",
			"GKILL_USER":                 "envuser",
			"GKILL_PASSWORD_SHA256":      "abc",
			"GKILL_INSECURE":             "true",
			"GKILL_FETCH_TIMEOUT_MS":     "5000",
			"GKILL_MCP_MAX_FILE_BYTES":   "1048576",
			"GKILL_MCP_FILE_LINK_TTL_MS": "60000",
			"MCP_TRANSPORT":              "http",
			"MCP_PORT":                   "9001",
			"MCP_BIND_ADDR":              "127.0.0.1",
			"MCP_OAUTH_ISSUER":           "https://mcp.example.test",
			"MCP_LOG":                    "debug",
		}))
		expectNoError(t, err)
		expectEqual(t, settings.Transport, "http")
		expectEqual(t, settings.LogLevelName, "debug")
		expectEqual(t, settings.Port, 9001)
		expectEqual(t, settings.BindAddr, "127.0.0.1")
		expectEqual(t, settings.OAuthIssuer, "https://mcp.example.test")
		expectEqual(t, settings.Client.BaseURL, "https://127.0.0.1:9999")
		expectEqual(t, settings.Client.UserID, "envuser")
		expectEqual(t, settings.Client.PasswordSha256, "abc")
		expectTrue(t, settings.Client.Insecure, "insecure not applied")
		expectEqual(t, int64(settings.Client.FetchTimeout), int64(5*time.Second))
		expectEqual(t, settings.MaxFileBytes, int64(1048576))
		expectEqual(t, int64(settings.FileLinkTTL), int64(time.Minute))
	})

	t.Run("flags override environment variables", func(t *testing.T) {
		settings, err := ResolveSettings(DefaultConfig(), "read", FlagOverrides{Transport: "http", LogLevel: "warn"}, envMap(map[string]string{
			"MCP_TRANSPORT": "stdio",
			"MCP_LOG":       "debug",
		}))
		expectNoError(t, err)
		expectEqual(t, settings.Transport, "http")
		expectEqual(t, settings.LogLevelName, "warn")
	})

	t.Run("the plain password is accepted only from the environment", func(t *testing.T) {
		settings, err := ResolveSettings(DefaultConfig(), "read", FlagOverrides{}, envMap(map[string]string{"GKILL_PASSWORD": "secret"}))
		expectNoError(t, err)
		expectEqual(t, settings.Client.Password, "secret")
		client := NewGkillClient(settings.Client)
		expectEqual(t, len(client.ResolvePasswordSha256()), 64)
	})

	t.Run("rejects an unknown kind, transport, log level and port", func(t *testing.T) {
		_, err := ResolveSettings(DefaultConfig(), "admin", FlagOverrides{}, envMap(nil))
		expectErrorContains(t, err, "--kind must be one of")
		_, err = ResolveSettings(DefaultConfig(), "read", FlagOverrides{Transport: "grpc"}, envMap(nil))
		expectErrorContains(t, err, "transport must be stdio or http")
		_, err = ResolveSettings(DefaultConfig(), "read", FlagOverrides{}, envMap(map[string]string{"MCP_LOG": "verbose"}))
		expectTrue(t, err != nil && strings.Contains(err.Error(), "verbose"), "unknown level accepted: %v", err)
		_, err = ResolveSettings(DefaultConfig(), "read", FlagOverrides{}, envMap(map[string]string{"MCP_PORT": "http"}))
		expectErrorContains(t, err, "MCP_PORT")
	})

	t.Run("the port of the requested kind comes from its own section", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Servers["write"] = ServerConfig{Port: 18809, OAuthIssuer: "https://write.example.test"}
		settings, err := ResolveSettings(cfg, "write", FlagOverrides{}, envMap(nil))
		expectNoError(t, err)
		expectEqual(t, settings.Port, 18809)
		expectEqual(t, settings.OAuthIssuer, "https://write.example.test")
	})
}

func TestApplySettings(t *testing.T) {
	originalMax := MaxIDFFileBytes
	originalTTL := FileLinkTTL
	t.Cleanup(func() {
		MaxIDFFileBytes = originalMax
		FileLinkTTL = originalTTL
	})
	ApplySettings(Settings{MaxFileBytes: 12345, FileLinkTTL: 7 * time.Second})
	expectEqual(t, MaxIDFFileBytes, int64(12345))
	expectEqual(t, int64(FileLinkTTL), int64(7*time.Second))
	expectEqual(t, int64(NewFileLinkStore(0).TTL()), int64(7*time.Second))
}

// 数値の環境変数が壊れていたら起動を止める（既定へ黙って落とすと「設定したつもり」で運用が続く）。
// GKILL_MCP_MAX_FILE_BYTES / GKILL_MCP_FILE_LINK_TTL_MS は単位付きの文字列を受けるので、
// 読めない値は既定へ落ちる（ParseByteLimit / ParseFileLinkTTL の契約）ことも同時に固定する。
func TestResolveSettingsRejectsBrokenNumericEnv(t *testing.T) {
	t.Run("GKILL_FETCH_TIMEOUT_MS must be a positive integer", func(t *testing.T) {
		for _, v := range []string{"abc", "0", "-5", "1.5"} {
			_, err := ResolveSettings(DefaultConfig(), "read", FlagOverrides{}, envMap(map[string]string{"GKILL_FETCH_TIMEOUT_MS": v}))
			expectErrorContains(t, err, "GKILL_FETCH_TIMEOUT_MS")
		}
		settings, err := ResolveSettings(DefaultConfig(), "read", FlagOverrides{}, envMap(map[string]string{"GKILL_FETCH_TIMEOUT_MS": " 250 "}))
		expectNoError(t, err)
		expectEqual(t, int64(settings.Client.FetchTimeout), int64(250*time.Millisecond))
	})

	t.Run("MCP_PORT must be within 1..65535", func(t *testing.T) {
		for _, v := range []string{"0", "65536", "-1", "8080x"} {
			_, err := ResolveSettings(DefaultConfig(), "read", FlagOverrides{}, envMap(map[string]string{"MCP_PORT": v}))
			expectErrorContains(t, err, "MCP_PORT")
		}
	})

	t.Run("unreadable byte limits and TTLs fall back to the defaults instead of zero", func(t *testing.T) {
		settings, err := ResolveSettings(DefaultConfig(), "read", FlagOverrides{}, envMap(map[string]string{
			"GKILL_MCP_MAX_FILE_BYTES":   "lots",
			"GKILL_MCP_FILE_LINK_TTL_MS": "soon",
		}))
		expectNoError(t, err)
		expectEqual(t, settings.MaxFileBytes, int64(8*1024*1024))
		expectEqual(t, int64(settings.FileLinkTTL), int64(time.Hour))
	})

	t.Run("GKILL_INSECURE accepts true / 1 and anything else means false", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Gkill.Insecure = true
		for v, want := range map[string]bool{"true": true, "1": true, "false": false, "0": false, "yes": false} {
			settings, err := ResolveSettings(cfg, "read", FlagOverrides{}, envMap(map[string]string{"GKILL_INSECURE": v}))
			expectNoError(t, err)
			expectEqual(t, settings.Client.Insecure, want)
		}
		// 空なら（未設定なら）ファイルの値
		settings, err := ResolveSettings(cfg, "read", FlagOverrides{}, envMap(nil))
		expectNoError(t, err)
		expectTrue(t, settings.Client.Insecure, "file value not used when env is unset")
	})
}
