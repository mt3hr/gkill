package api

// embed.go の init() が登録するMIME型を固定する。
// 登録行を落としても go build / go vet は通り、
// 「manifest が text/plain で配られる」形で静かに壊れるため、ここでしか気付けない。

import (
	"mime"
	"testing"
)

func TestInit_WebmanifestMIMEIsRegistered(t *testing.T) {
	// http.FileServer は mime.TypeByExtension が空を返すと中身を見て判定するので、
	// ここが空だと manifest.webmanifest が text/plain で配信される。
	got := mime.TypeByExtension(".webmanifest")
	want := "application/manifest+json"
	if got != want {
		t.Errorf("mime.TypeByExtension(\".webmanifest\") = %q, want %q", got, want)
	}
}
