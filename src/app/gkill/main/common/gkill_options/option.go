package gkill_options

import (
	"math"
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

	IsCacheInMemory = false
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

	GoroutinePool = 100

	CacheClearCountLimit = math.MaxInt

	CacheKmemoReps        = false
	CacheKCReps           = false
	CacheURLogReps        = false
	CacheNlogReps         = false
	CacheTimeIsReps       = false
	CacheMiReps           = false
	CacheLantanaReps      = false
	CacheIDFKyouReps      = false
	CacheTagReps          = true
	CacheTextReps         = false
	CacheNotificationReps = false
	CacheReKyouReps       = false
)
