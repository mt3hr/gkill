package mcp

// 設定ファイル $GKILL_HOME/configs/gkill_mcp.json とその解決（フラグ > 環境変数 > ファイル > 既定値）。
//
// 旧実装（Node）は環境変数だけで構成していた。一本化にあたり、同じ環境変数はそのまま効かせつつ、
// 何も渡さない起動でも読める設定ファイルを初回起動時に生成する。
// 壊れたファイルは黙って既定へ落とさず起動を止める（gkill_server 本体の設定と同じ考え方）。

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ConfigFileName は設定ファイルの既定の名前（$GKILL_HOME/configs/ 配下）。
const ConfigFileName = "gkill_mcp.json"

// GkillConfig は gkill 本体への接続。
type GkillConfig struct {
	BaseURL        string `json:"base_url"`
	Insecure       bool   `json:"insecure"`
	Locale         string `json:"locale"`
	FetchTimeoutMS int64  `json:"fetch_timeout_ms"`
	User           string `json:"user"`
	PasswordSha256 string `json:"password_sha256"`
	SessionID      string `json:"session_id"`
}

// ServerConfig は種別ごとの節。
type ServerConfig struct {
	Port        int    `json:"port"`
	OAuthIssuer string `json:"oauth_issuer"`
}

// Config は設定ファイルの形。
type Config struct {
	Gkill         GkillConfig             `json:"gkill"`
	LogLevel      string                  `json:"log_level"`
	MaxFileBytes  int64                   `json:"max_file_bytes"`
	FileLinkTTLMS int64                   `json:"file_link_ttl_ms"`
	Transport     string                  `json:"transport"`
	BindAddr      string                  `json:"bind_addr"`
	Servers       map[string]ServerConfig `json:"servers"`
}

// DefaultConfig は初回起動時に書き出す既定値（旧実装の環境変数の既定と同じ）。
func DefaultConfig() Config {
	return Config{
		Gkill: GkillConfig{
			BaseURL:        DefaultGkillBaseURL,
			Insecure:       false,
			Locale:         "ja",
			FetchTimeoutMS: DefaultFetchTimeout.Milliseconds(),
		},
		LogLevel:      DefaultMcpLogLevelName,
		MaxFileBytes:  defaultMaxIDFFileBytes,
		FileLinkTTLMS: defaultFileLinkTTL.Milliseconds(),
		Transport:     "stdio",
		BindAddr:      "0.0.0.0",
		Servers: map[string]ServerConfig{
			"read":      {Port: ReadStartSpec.DefaultPort},
			"write":     {Port: WriteStartSpec.DefaultPort},
			"readwrite": {Port: ReadWriteStartSpec.DefaultPort},
		},
	}
}

// DefaultConfigPath は既定の設定ファイルの場所。
func DefaultConfigPath(configDir string) string {
	return filepath.Join(configDir, ConfigFileName)
}

// LoadOrCreateConfig は設定ファイルを読む。無ければ既定値で生成して（0600）返し、created を true にする。
// 既存のファイルは書き換えない。壊れていればエラー。
func LoadOrCreateConfig(path string) (Config, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Config{}, false, fmt.Errorf("error at read mcp config %s: %w", path, err)
		}
		cfg := DefaultConfig()
		if err := writeConfig(path, cfg); err != nil {
			return Config{}, false, err
		}
		return cfg, true, nil
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, false, fmt.Errorf("error at parse mcp config %s: %w", path, err)
	}
	if cfg.Servers == nil {
		cfg.Servers = DefaultConfig().Servers
	}
	return cfg, false, nil
}

func writeConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("error at mkdir for mcp config %s: %w", path, err)
	}
	encoded, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error at encode mcp config: %w", err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		return fmt.Errorf("error at write mcp config %s: %w", path, err)
	}
	return nil
}

// FlagOverrides はコマンドラインで明示された値（明示されたものだけが最優先になる）。
type FlagOverrides struct {
	Transport string
	LogLevel  string
}

// Settings は解決済みの起動設定。
type Settings struct {
	Kind         string
	Transport    string
	Client       ClientConfig
	LogLevelName string
	MaxFileBytes int64
	FileLinkTTL  time.Duration
	BindAddr     string
	Port         int
	OAuthIssuer  string
}

// Getenv は環境変数の読み口（テストで差し替える）。
type Getenv func(key string) string

// ResolveSettings はフラグ > 環境変数 > ファイル > 既定値の順で起動設定を決める。
func ResolveSettings(cfg Config, kind string, flags FlagOverrides, getenv Getenv) (Settings, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	if kind != "read" && kind != "write" && kind != "readwrite" {
		return Settings{}, fmt.Errorf("--kind must be one of read, write, readwrite (got %q)", kind)
	}
	pick := func(envKey, fileValue string) string {
		if v := getenv(envKey); v != "" {
			return v
		}
		return fileValue
	}
	settings := Settings{Kind: kind}

	// トランスポート
	transport := strings.ToLower(pick("MCP_TRANSPORT", cfg.Transport))
	if flags.Transport != "" {
		transport = strings.ToLower(flags.Transport)
	}
	if transport == "" {
		transport = "stdio"
	}
	if transport != "stdio" && transport != "http" {
		return Settings{}, fmt.Errorf("transport must be stdio or http (got %q)", transport)
	}
	settings.Transport = transport

	// ログレベル（未知の値は起動を止める。旧実装は黙って info へ落としていた）
	levelName := pick("MCP_LOG", cfg.LogLevel)
	if flags.LogLevel != "" {
		levelName = flags.LogLevel
	}
	if _, err := ParseMcpLogLevel(levelName); err != nil {
		return Settings{}, err
	}
	levelName = strings.ToLower(strings.TrimSpace(levelName))
	if levelName == "" {
		levelName = DefaultMcpLogLevelName
	}
	settings.LogLevelName = levelName

	// gkill 接続
	client := ClientConfig{
		BaseURL:        pick("GKILL_BASE_URL", cfg.Gkill.BaseURL),
		UserID:         pick("GKILL_USER", cfg.Gkill.User),
		PasswordSha256: pick("GKILL_PASSWORD_SHA256", cfg.Gkill.PasswordSha256),
		// 平文パスワードは環境変数からだけ受ける（ファイルには置かない）
		Password:  getenv("GKILL_PASSWORD"),
		Locale:    pick("GKILL_LOCALE", cfg.Gkill.Locale),
		SessionID: pick("GKILL_SESSION_ID", cfg.Gkill.SessionID),
		Insecure:  cfg.Gkill.Insecure,
	}
	if v := getenv("GKILL_INSECURE"); v != "" {
		client.Insecure = v == "true" || v == "1"
	}
	fetchTimeoutMS := cfg.Gkill.FetchTimeoutMS
	if v := getenv("GKILL_FETCH_TIMEOUT_MS"); v != "" {
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil || parsed <= 0 {
			return Settings{}, fmt.Errorf("GKILL_FETCH_TIMEOUT_MS must be a positive integer (got %q)", v)
		}
		fetchTimeoutMS = parsed
	}
	if fetchTimeoutMS <= 0 {
		fetchTimeoutMS = DefaultFetchTimeout.Milliseconds()
	}
	client.FetchTimeout = time.Duration(fetchTimeoutMS) * time.Millisecond
	settings.Client = client

	// 上限と寿命
	settings.MaxFileBytes = cfg.MaxFileBytes
	if v := getenv("GKILL_MCP_MAX_FILE_BYTES"); v != "" {
		settings.MaxFileBytes = ParseByteLimit(v, defaultMaxIDFFileBytes, 1)
	}
	if settings.MaxFileBytes <= 0 {
		settings.MaxFileBytes = defaultMaxIDFFileBytes
	}
	fileLinkTTLMS := cfg.FileLinkTTLMS
	if v := getenv("GKILL_MCP_FILE_LINK_TTL_MS"); v != "" {
		settings.FileLinkTTL = ParseFileLinkTTL(v)
	} else {
		settings.FileLinkTTL = ParseFileLinkTTL(strconv.FormatInt(fileLinkTTLMS, 10))
	}

	// HTTP
	settings.BindAddr = pick("MCP_BIND_ADDR", cfg.BindAddr)
	if settings.BindAddr == "" {
		settings.BindAddr = "0.0.0.0"
	}
	serverCfg := cfg.Servers[kind]
	port := serverCfg.Port
	if v := getenv("MCP_PORT"); v != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || parsed <= 0 || parsed > 65535 {
			return Settings{}, fmt.Errorf("MCP_PORT must be a port number (got %q)", v)
		}
		port = parsed
	}
	if port == 0 {
		port = StartSpecFor(kind).DefaultPort
	}
	settings.Port = port
	settings.OAuthIssuer = pick("MCP_OAUTH_ISSUER", serverCfg.OAuthIssuer)
	return settings, nil
}

// ApplySettings は package 変数で持つ上限（環境変数から初期化される）を解決済みの値で上書きする。
func ApplySettings(settings Settings) {
	if settings.MaxFileBytes > 0 {
		MaxIDFFileBytes = settings.MaxFileBytes
	}
	if settings.FileLinkTTL > 0 {
		FileLinkTTL = settings.FileLinkTTL
	}
}
