package safefetch

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsDisallowedFetchIP(t *testing.T) {
	// allowPrivate=false: 内部系はすべて拒否
	disallowed := []string{
		"127.0.0.1", "10.0.0.1", "172.16.0.1", "192.168.1.1",
		"169.254.169.254", "0.0.0.0", "::1", "fe80::1", "fc00::1",
	}
	for _, s := range disallowed {
		if !IsDisallowedFetchIP(net.ParseIP(s), false) {
			t.Errorf("IsDisallowedFetchIP(%s, false) = false, want true", s)
		}
	}
	allowed := []string{"93.184.216.34", "8.8.8.8", "2001:4860:4860::8888"}
	for _, s := range allowed {
		if IsDisallowedFetchIP(net.ParseIP(s), false) {
			t.Errorf("IsDisallowedFetchIP(%s, false) = true, want false", s)
		}
	}
	if !IsDisallowedFetchIP(nil, false) {
		t.Error("IsDisallowedFetchIP(nil, false) = false, want true")
	}

	// allowPrivate=true: loopback/private は許可、リンクローカル(metadata)/マルチキャスト/未指定は拒否
	if IsDisallowedFetchIP(net.ParseIP("127.0.0.1"), true) {
		t.Error("allowPrivate=true should allow loopback")
	}
	if IsDisallowedFetchIP(net.ParseIP("192.168.1.1"), true) {
		t.Error("allowPrivate=true should allow private")
	}
	stillBlocked := []string{"169.254.169.254", "fe80::1", "224.0.0.1", "0.0.0.0"}
	for _, s := range stillBlocked {
		if !IsDisallowedFetchIP(net.ParseIP(s), true) {
			t.Errorf("IsDisallowedFetchIP(%s, true) = false, want true (link-local/multicast/unspecified always blocked)", s)
		}
	}
}

func TestGetCapped_BlocksLoopbackByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request to loopback server should have been blocked")
	}))
	defer server.Close()

	_, err := GetCapped(server.URL, 0, "", false, DefaultMaxBodyBytes)
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Errorf("GetCapped to loopback with allowPrivate=false should fail with blocked, got: %v", err)
	}
}

func TestGetCapped_AllowsLoopbackWhenAllowPrivate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))
	}))
	defer server.Close()

	b, err := GetCapped(server.URL, 0, "", true, DefaultMaxBodyBytes)
	if err != nil {
		t.Fatalf("GetCapped with allowPrivate=true should succeed against loopback, got: %v", err)
	}
	if string(b) != "hello" {
		t.Errorf("got %q, want hello", string(b))
	}
}

func TestGetCapped_RejectsScheme(t *testing.T) {
	for _, u := range []string{"file:///etc/passwd", "ftp://example.com/a", "gopher://example.com"} {
		_, err := GetCapped(u, 0, "", false, DefaultMaxBodyBytes)
		if err == nil || !strings.Contains(err.Error(), "unsupported url scheme") {
			t.Errorf("GetCapped(%s) should fail with scheme error, got: %v", u, err)
		}
	}
}

func TestGetCapped_EnforcesSizeCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 1000))
	}))
	defer server.Close()

	_, err := GetCapped(server.URL, 0, "", true, 10)
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Errorf("GetCapped should reject over-cap body, got: %v", err)
	}
}

func TestCheckImageDimensions(t *testing.T) {
	// 2x2 の実PNGを作る
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()

	if err := CheckImageDimensions(data, DefaultMaxImagePixels); err != nil {
		t.Errorf("2x2 image should pass default cap, got: %v", err)
	}
	// 上限を1ピクセルに絞ると 2x2=4 は拒否される（画像爆弾の判定ロジック）。
	if err := CheckImageDimensions(data, 1); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Errorf("2x2 image should exceed a 1-pixel cap, got: %v", err)
	}
	// 画像でないデータは DecodeConfig で弾く。
	if err := CheckImageDimensions([]byte("not an image"), DefaultMaxImagePixels); err == nil {
		t.Error("non-image data should fail CheckImageDimensions")
	}
}

// GetCapped は2xx以外の本文を返してはいけない。
//
// ここを見ていなかったせいで、404ページのHTMLがそのまま favicon として
// base64 で保存されていた(指摘でGoogleの "Error 404 (Not Found)!!1" を実測)。
// 404ページの <title> がブックマークのタイトルとして保存される経路も同じ原因。
func TestGetCapped_RejectsNon2xx(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte("<html><title>Error 404 (Not Found)!!1</title></html>"))
		}))

		b, err := GetCapped(server.URL, 0, "", true, DefaultMaxBodyBytes)
		server.Close()

		if err == nil {
			t.Errorf("GetCapped should reject status %d, got body %q", status, string(b))
			continue
		}
		if !strings.Contains(err.Error(), "status =") {
			t.Errorf("status %d のエラー文にステータスが入っていない: %v", status, err)
		}
		if b != nil {
			t.Errorf("status %d で本文を返している: %q", status, string(b))
		}
	}
}

// 2xx なら今までどおり本文を返す(退行していないこと)。
func TestGetCapped_Accepts2xx(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, 299} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte("ok body"))
		}))

		b, err := GetCapped(server.URL, 0, "", true, DefaultMaxBodyBytes)
		server.Close()

		if err != nil {
			t.Errorf("status %d は通るべき: %v", status, err)
			continue
		}
		if string(b) != "ok body" {
			t.Errorf("status %d: got %q, want %q", status, string(b), "ok body")
		}
	}
}

// LooksLikeSupportedImage はクライアントが表示できる4形式だけを通す。
//
// 判定形式は src/client/classes/use-ur-log-view.ts の base64_to_data_uri と揃えてある。
// あちらが増えたらこちらも足すこと(逆も同じ)。
func TestLooksLikeSupportedImage(t *testing.T) {
	png2x2 := makeTestPNG(t)

	supported := map[string][]byte{
		"png":    png2x2,
		"gif87a": []byte("GIF87a....."),
		"gif89a": []byte("GIF89a....."),
		"jpeg":   {0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
		"webp":   append([]byte("RIFF\x00\x00\x00\x00WEBP"), 0x00),
	}
	for name, data := range supported {
		if !LooksLikeSupportedImage(data) {
			t.Errorf("%s が画像として認識されない", name)
		}
	}

	rejected := map[string][]byte{
		"404のHTML":   []byte("<html><title>Error 404 (Not Found)!!1</title></html>"),
		"空":          {},
		"短すぎるRIFF":   []byte("RIFF"),
		"RIFFだがwave": []byte("RIFF\x00\x00\x00\x00WAVE"),
		"svg":        []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
		"ico":        {0x00, 0x00, 0x01, 0x00},
	}
	for name, data := range rejected {
		if LooksLikeSupportedImage(data) {
			t.Errorf("%s が画像として通ってしまう", name)
		}
	}
}

// makeTestPNG は2x2の実PNGを作る。
func makeTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}
