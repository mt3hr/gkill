package reps

// fillFavicon / getFavicon の回帰テスト。
//
// safefetch 側の部品（2xx判定・LooksLikeSupportedImage・CheckImageDimensions）の
// 単体テストは api/safefetch/safefetch_test.go にある。ここで固定するのは
// **ur_log 側の配線**で、どれも外すと目の前ではエラーにならず、
// 画像でないものが favicon として保存される・無関係のアイコンが保存される、
// という静かな壊れ方をする:
//   - 取れたバイト列が画像であることを確かめてから保存する
//     （エラーページを200で返すサイトのHTMLがbase64で入っていた）
//   - 復号前に寸法を検査する（favicon経路だけ fillImage の検査を通らない）
//   - ホスト名が取れないURLではリクエスト自体を出さない
//     （Googleは domain= 空へ汎用アイコンを200で返すので、出したら弾けない）
//
// 本番の取得先は Google の favicon API 固定でテストから到達できないため、
// fetchFaviconBytes（ur_log.go のパッケージ変数）を httptest.Server を指す
// クロージャへ差し替える。safefetch.GetCapped 本体は通したままにする。

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/safefetch"
)

// makeFaviconTestPNG は1x1の実PNGバイト列を作る。
func makeFaviconTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

// makeHugeDimensionPNGHeader は寸法だけ巨大なPNG（IHDRのみ、画素データ無し）を作る。
// image.DecodeConfig はIHDRだけで寸法を返すので、画像爆弾の検査対象として十分。
func makeHugeDimensionPNGHeader(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	var ihdr bytes.Buffer
	ihdr.WriteString("IHDR")
	if err := binary.Write(&ihdr, binary.BigEndian, uint32(100000)); err != nil {
		t.Fatalf("binary.Write width: %v", err)
	}
	if err := binary.Write(&ihdr, binary.BigEndian, uint32(100000)); err != nil {
		t.Fatalf("binary.Write height: %v", err)
	}
	ihdr.Write([]byte{8, 6, 0, 0, 0}) // bit depth 8 / RGBA / 圧縮・フィルタ・インタレース既定
	if err := binary.Write(&buf, binary.BigEndian, uint32(13)); err != nil {
		t.Fatalf("binary.Write length: %v", err)
	}
	buf.Write(ihdr.Bytes())
	if err := binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(ihdr.Bytes())); err != nil {
		t.Fatalf("binary.Write crc: %v", err)
	}

	// フィクスチャの自己検査: ここが壊れているとテストが別の理由で通ってしまう
	cfg, _, err := image.DecodeConfig(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("フィクスチャのPNGヘッダが解釈できない: %v", err)
	}
	if cfg.Width != 100000 || cfg.Height != 100000 {
		t.Fatalf("フィクスチャの寸法 = %dx%d, want 100000x100000", cfg.Width, cfg.Height)
	}
	return buf.Bytes()
}

// overrideFaviconFetch は favicon の取得先を httptest.Server へ差し替える。
// httptest.Server はループバック待受なので allowPrivate=true で取る
// （2xx判定・サイズ上限など safefetch.GetCapped の実物は通したまま）。
func overrideFaviconFetch(t *testing.T, server *httptest.Server) {
	t.Helper()
	orig := fetchFaviconBytes
	fetchFaviconBytes = func(hostname string) ([]byte, error) {
		return safefetch.GetCapped(server.URL+"/s2/favicons?domain="+hostname, 5*time.Second, "", true, safefetch.DefaultMaxImageBytes)
	}
	t.Cleanup(func() { fetchFaviconBytes = orig })
}

// 正常系: 小さい実画像はbase64でFaviconImageへ保存される。
func TestURLogFillFavicon_SavesSmallImage(t *testing.T) {
	pngBytes := makeFaviconTestPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(pngBytes)
	}))
	defer server.Close()
	overrideFaviconFetch(t, server)

	urlog := &URLog{URL: "https://example.com/page"}
	if err := urlog.fillFavicon(); err != nil {
		t.Fatalf("fillFavicon failed: %v", err)
	}
	if want := base64.StdEncoding.EncodeToString(pngBytes); urlog.FaviconImage != want {
		t.Errorf("FaviconImage が取得した画像のbase64になっていない: got %d chars, want %d chars", len(urlog.FaviconImage), len(want))
	}
}

// HTTP 200 でも中身がHTML（エラーページを200で返すサイト想定）なら保存しない。
// 2xx判定を入れる前は404ページのHTMLがそのままbase64でFaviconImageに入っていた。
// 2xx判定を入れた今でも、エラーページを200で返すサイトは残るので中身で見る。
func TestURLogFillFavicon_RejectsHTMLBodyWith200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><title>Error 404 (Not Found)!!1</title><body>not an image</body></html>"))
	}))
	defer server.Close()
	overrideFaviconFetch(t, server)

	urlog := &URLog{URL: "https://example.com/page"}
	if err := urlog.fillFavicon(); err == nil {
		t.Error("HTML本文なのに fillFavicon が成功している（LooksLikeSupportedImage の配線が外れている）")
	}
	if urlog.FaviconImage != "" {
		t.Errorf("画像でない応答が FaviconImage へ保存されている: %d chars", len(urlog.FaviconImage))
	}
}

// マジックバイトは正当でも、寸法が巨大な画像（画像爆弾）は保存しない。
// favicon経路だけ fillImage の CheckImageDimensions を通らないので、ここで配線を固定する。
func TestURLogFillFavicon_RejectsHugeDimensionImage(t *testing.T) {
	hugePNG := makeHugeDimensionPNGHeader(t)
	if !safefetch.LooksLikeSupportedImage(hugePNG) {
		t.Fatal("フィクスチャがマジックバイト検査を通らない（寸法検査へ到達しない）")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(hugePNG)
	}))
	defer server.Close()
	overrideFaviconFetch(t, server)

	urlog := &URLog{URL: "https://example.com/page"}
	if err := urlog.fillFavicon(); err == nil {
		t.Error("寸法が巨大な画像なのに fillFavicon が成功している（CheckImageDimensions の配線が外れている）")
	}
	if urlog.FaviconImage != "" {
		t.Errorf("画像爆弾が FaviconImage へ保存されている: %d chars", len(urlog.FaviconImage))
	}
}

// ホスト名が取れないURL（スキーム無し等）ではリクエスト自体を出さずにスキップする。
// Googleは domain= 空に対して汎用アイコンを200で返すので、リクエストを出したら
// ステータスでも中身でも弾けず、無関係のアイコンが保存されてしまう。
func TestURLogGetFavicon_EmptyHostnameSkipsRequest(t *testing.T) {
	orig := fetchFaviconBytes
	fetchFaviconBytes = func(hostname string) ([]byte, error) {
		t.Errorf("ホスト名が取れないURLなのにfaviconリクエストが飛んだ: hostname=%q", hostname)
		return nil, fmt.Errorf("unexpected favicon request")
	}
	t.Cleanup(func() { fetchFaviconBytes = orig })

	if _, err := getFavicon("example.com/foo"); err == nil {
		t.Error("スキーム無しURL（ホスト名なし）で getFavicon が成功している")
	}

	// fillFavicon 経由でも同じ: エラーになり、FaviconImage は空のまま
	urlog := &URLog{URL: "example.com/foo"}
	if err := urlog.fillFavicon(); err == nil {
		t.Error("スキーム無しURLで fillFavicon が成功している")
	}
	if urlog.FaviconImage != "" {
		t.Errorf("スキップしたはずのURLで FaviconImage が埋まっている: %d chars", len(urlog.FaviconImage))
	}
}
