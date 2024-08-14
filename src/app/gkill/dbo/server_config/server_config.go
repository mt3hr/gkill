// ˅
package server_config

import "time"

// ˄

type ServerConfig struct {
	// ˅

	// ˄

	Device string

	IsLocalOnlyAccess bool

	Address string

	EnableTLS bool

	TLSCertFile string

	TLSKeyFile string

	OpenDirectoryCommand string

	OpenFileCommand string

	URLogTimeout time.Duration

	URLogUserAgent string

	UploadSizeLimitMonth int

	UserDataDirectory string

	// ˅

	// ˄
}

// ˅

// ˄
