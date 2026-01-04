package gkill_options

import "time"

var (
	falseValue = false
	trueValue  = true
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

	PreLoadUserNames = []string{}

	IsCacheInMemory = true
	IsOutputLog     = false
	DisableTLSForce = false

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

	GoroutinePool = 1000

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
	CacheNotificationReps = &falseValue // 未検証
	CacheReKyouReps       = &falseValue // 未検証
)
