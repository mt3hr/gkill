// Package gkill_options はCLIフラグとディレクトリ構成のグローバル設定。
package gkill_options

import (
	"net"
	"runtime"
	"time"
)

var (
	GkillHomeDir         = "$HOME/gkill"
	LibDir               = "$HOME/gkill/lib/base_directory"
	CacheDir             = "$HOME/gkill/caches"
	LogDir               = "$HOME/gkill/logs"
	ConfigDir            = "$HOME/gkill/configs"
	TLSCertFileDefault   = "$HOME/gkill/tls/cert.cer"
	TLSKeyFileDefault    = "$HOME/gkill/tls/key.pem"
	DataDirectoryDefault = "$HOME/gkill/datas"
	// SkillsDir は利用者が AI 向けに書くスキル（SKILL.md とその付属ファイル）の置き場。
	// skills/<user_id>/<skill-name>/ の形で、gkill_server だけが読み書きする（ADR-0634）。
	SkillsDir = "$HOME/gkill/skills"

	PreLoadUserNames = []string{}

	// ServerAddress はリッスンアドレスのCLI上書き。空文字なら設定DBのADDRESSを使う
	ServerAddress = ""

	IsCacheInMemory = true
	IsOutputLog     = false

	// LogRotateMaxBytes は1つのログファイルの上限。超えると gkill_xxx.log.1 .. .N へ世代を回す。
	// 0以下にすると回転しない（上限なく育つ）。
	LogRotateMaxBytes int64 = 32 * 1024 * 1024
	// LogRotateKeep は残す世代数。0以下なら退避せずに捨てる。
	LogRotateKeep = 5

	DisableTLSForce = false

	Optimize = false

	LoadIDFRepOnly = false

	CacheRepsLocalStorage = false

	IDFIgnore = []string{
		".gkill",
		"gkill_id.db",
		"gkill_id.db-journal",
		"gkill_id.db-shm",
		"gkill_id.db-wal",
		".nomedia",
		"desktop.ini",
		"thumbnails",
		".thumbnails",
		"Thumbs.db",
		"steam_autocloud.vdf",
		".DS_Store",
		".localized",
		".kyou",
		"id.db",
		"id.db-journal",
		"id.db-shm",
		"id.db-wal",
	}

	GoroutinePool = runtime.NumCPU()

	CacheClearCountLimit int64 = 3000 // int64(9007199254740991) // javascriptのNumber上限値
	CacheUpdateDuration        = 1 * time.Minute

	CacheKmemoReps        = &IsCacheInMemory
	CacheKCReps           = &IsCacheInMemory
	CacheURLogReps        = &IsCacheInMemory
	CacheNlogReps         = &IsCacheInMemory
	CacheTimeIsReps       = &IsCacheInMemory
	CacheMiReps           = &IsCacheInMemory
	CacheLantanaReps      = &IsCacheInMemory
	CacheIDFKyouReps      = &IsCacheInMemory
	CacheTagReps          = &IsCacheInMemory
	CacheTextReps         = &IsCacheInMemory
	CacheNotificationReps = &IsCacheInMemory
	CacheReKyouReps       = &IsCacheInMemory
	CacheMiReKyouReps     = &IsCacheInMemory
	CacheGitCommitLogReps = &IsCacheInMemory
)

// ResolveServerAddress は--addressが指定されていればそれを、なければ設定DBの値を返す
func ResolveServerAddress(configured string) string {
	if ServerAddress != "" {
		return ServerAddress
	}
	return configured
}

// ServerAddressPortSuffix は "http://localhost%s" のような組み立てに使う ":ポート" 部分を返す。
// --addressにホストが含まれていても (127.0.0.1:9999 等) ポートだけを取り出す
func ServerAddressPortSuffix(configured string) string {
	address := ResolveServerAddress(configured)
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	return ":" + port
}
