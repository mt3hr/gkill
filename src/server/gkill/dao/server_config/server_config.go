// Package server_config はサーバ設定のDAO。
package server_config

import (
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/account"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// 待受の既定値。初回起動ブロック（gkill_server_api.go）と DAO の既定マップ
// （server_config_dao_sqlite3_impl.go の serverConfigDefaultValue）が両方ここを引く。
//
// 既定はループバック限定にする。人生記録を保存するアプリなので、何も知らない利用者が
// 起動しただけで LAN の第三者から届く状態にしない。他の端末から使うときは
// サーバ設定画面でアドレスと「ローカルアクセスのみ許可」を明示的に開く。
// 2026-09-14 まで初回起動ブロックは ":9999"（全インターフェース）+ IsLocalOnlyAccess=false を
// リテラルで書いており、DAO 側の既定（IsLocalOnlyAccess=true）と食い違っていた。
// 既定を2箇所に書くとこうなるので、ここ以外に既定値を書かないこと。
const (
	// DefaultListenAddress は待受アドレスの既定。ホスト部を空（":9999"）にすると全インターフェースで待つ。
	DefaultListenAddress = "127.0.0.1:9999"
	// DefaultIsLocalOnlyAccess は「ローカルアクセスのみ許可」の既定。
	// 待受をループバックに限定していても、--address で bind を広げたときの2枚目の防御線になる。
	DefaultIsLocalOnlyAccess = true
)

type ServerConfig struct {
	EnableThisDevice bool `json:"enable_this_device"`

	Device string `json:"device"`

	IsLocalOnlyAccess bool `json:"is_local_only_access"`

	Address string `json:"address"`

	EnableTLS bool `json:"enable_tls"`

	TLSCertFile string `json:"tls_cert_file"`

	TLSKeyFile string `json:"tls_key_file"`

	OpenDirectoryCommand string `json:"open_directory_command"`

	OpenFileCommand string `json:"open_file_command"`

	URLogTimeout time.Duration `json:"urlog_timeout"`

	URLogUserAgent string `json:"urlog_useragent"`

	UploadSizeLimitMonth int `json:"upload_size_limit_month"`

	UserDataDirectory string `json:"user_data_directory"`

	GkillNotificationPublicKey string `json:"gkill_notification_public_key"`

	GkillNotificationPrivateKey string `json:"gkill_notification_private_key"`

	UseGkillNotification bool `json:"use_gkill_notification"`

	GoogleMapAPIKey string `json:"google_map_api_key"`

	LanHostname string `json:"lan_hostname"`

	GlobalHostname string `json:"global_hostname"`

	Repositories []*user_config.Repository `json:"repositories"`

	Accounts []*account.Account `json:"accounts"`
}
