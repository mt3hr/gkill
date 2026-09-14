package gkill_server_api

import (
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
)

// isLoopbackOnlyBindAddress は「TLS 無しで外から届くアドレスで待っている」警告の可否を決める。
// ホスト部が空（":9999"）は全インターフェース待受なので、これをループバック扱いにすると
// 警告が一度も出なくなる。判定を固定する。
func TestIsLoopbackOnlyBindAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{name: "ホスト部が空は全インターフェース", address: ":9999", want: false},
		{name: "0.0.0.0 は全インターフェース", address: "0.0.0.0:9999", want: false},
		{name: "[::] は全インターフェース", address: "[::]:9999", want: false},
		{name: "LAN のアドレス", address: "192.168.1.2:9999", want: false},
		{name: "IPv4 ループバック", address: "127.0.0.1:9999", want: true},
		{name: "127.0.0.0/8 の別アドレス", address: "127.0.0.2:9999", want: true},
		{name: "IPv6 ループバック", address: "[::1]:9999", want: true},
		{name: "localhost", address: "localhost:9999", want: true},
		{name: "ポート無しの localhost", address: "localhost", want: true},
		{name: "解釈できない文字列", address: "invalid", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLoopbackOnlyBindAddress(tt.address); got != tt.want {
				t.Errorf("isLoopbackOnlyBindAddress(%q) = %v, want %v", tt.address, got, tt.want)
			}
		})
	}
}

// 既定の待受アドレスはループバック限定であること。
// 既定を ":9999" に戻すと、初回起動しただけで LAN の第三者から届く状態になる（ADR-0708）。
func TestDefaultListenAddressIsLoopbackOnly(t *testing.T) {
	if !isLoopbackOnlyBindAddress(server_config.DefaultListenAddress) {
		t.Errorf("server_config.DefaultListenAddress = %q はループバック限定ではない", server_config.DefaultListenAddress)
	}
	if !server_config.DefaultIsLocalOnlyAccess {
		t.Error("server_config.DefaultIsLocalOnlyAccess = false。既定は「ローカルアクセスのみ許可」であること")
	}
}
