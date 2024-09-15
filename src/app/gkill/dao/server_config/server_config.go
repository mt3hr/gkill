package server_config

import "time"

type ServerConfig struct {
	Device string `json:"device"`

	IsLocalOnlyAccess bool `json:"is_local_only_access"`

	Address string `json:"address"`

	EnableTLS bool `json:"enable_tls"`

	TLSCertFile string `json:"tls_cert_file"`

	TLSKeyFile string `json:"tls_key_file"`

	OpenDirectoryCommand string `json:"open_directory_command"`

	OpenFileCommand string `json:"open_file_command"`

	URLogTimeout time.Duration `json:"ur_log_timeout"`

	URLogUserAgent string `json:"ur_log_user_agent"`

	UploadSizeLimitMonth int `json:"upload_size_limit_month"`

	UserDataDirectory string `json:"user_data_directory"`
}
