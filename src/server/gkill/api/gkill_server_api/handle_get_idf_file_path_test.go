package gkill_server_api

import (
	"net/http"
	"testing"
)

// ファイルの絶対パスは同一マシンのクライアントにしか意味がなく、
// 外部に渡すとユーザのディレクトリ構造の漏洩になる。
// その可否を決めるのがisLocalRequestなので、判定を固定する。
func TestIsLocalRequest(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       bool
	}{
		{
			name:       "IPv4ループバック",
			remoteAddr: "127.0.0.1:54321",
			want:       true,
		},
		{
			name:       "IPv6ループバック",
			remoteAddr: "[::1]:54321",
			want:       true,
		},
		{
			name:       "localhost",
			remoteAddr: "localhost:54321",
			want:       true,
		},
		{
			name:       "LAN内の別ホスト",
			remoteAddr: "192.168.1.20:54321",
			want:       false,
		},
		{
			name:       "外部ホスト",
			remoteAddr: "203.0.113.7:443",
			want:       false,
		},
		{
			name:       "IPv6の外部ホスト",
			remoteAddr: "[2001:db8::1]:443",
			want:       false,
		},
		{
			name:       "ループバックに見せかけたホスト名",
			remoteAddr: "127.0.0.1.example.com:80",
			want:       false,
		},
		{
			name:       "RemoteAddrが空",
			remoteAddr: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{RemoteAddr: tt.remoteAddr}
			if got := isLocalRequest(r); got != tt.want {
				t.Errorf("isLocalRequest(%q) = %v, want %v", tt.remoteAddr, got, tt.want)
			}
		})
	}
}
