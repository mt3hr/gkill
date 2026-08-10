package gkill_server_api

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// /api/ かつ Accept-Encoding: gzip のとき、応答が圧縮されて正しく復元できること
func TestGzipMiddlewareCompressesAPIResponse(t *testing.T) {
	body := strings.Repeat(`{"id":"01234567-89ab-cdef-0123-456789abcdef"}`, 1000)
	handler := gzipMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/get_kyous", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", got)
	}
	if recorder.Body.Len() >= len(body) {
		t.Fatalf("compressed size %d >= original %d", recorder.Body.Len(), len(body))
	}

	gzipReader, err := gzip.NewReader(recorder.Body)
	if err != nil {
		t.Fatalf("error at new gzip reader: %v", err)
	}
	defer gzipReader.Close()
	decoded, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("error at read gzip body: %v", err)
	}
	if string(decoded) != body {
		t.Fatalf("decoded body mismatch: len=%d want=%d", len(decoded), len(body))
	}
}

// Accept-Encoding に gzip が無ければ素通しすること
func TestGzipMiddlewareSkipsWhenNotAccepted(t *testing.T) {
	handler := gzipMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"messages":null}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/get_kyous", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if got := recorder.Header().Get("Content-Encoding"); got != "" {
		t.Fatalf("Content-Encoding = %q, want empty", got)
	}
	if recorder.Body.String() != `{"messages":null}` {
		t.Fatalf("body should be passed through: %q", recorder.Body.String())
	}
}

// /api/ 以外(ファイル配信・ZIP展開物)は圧縮しないこと。
// 動画のRange配信や既圧縮メディアを壊さないための不変条件
func TestGzipMiddlewareSkipsNonAPIPaths(t *testing.T) {
	for _, path := range []string{"/files/rep/movie.mp4", "/zip_cache/rep/x/1.png", "/rykv"} {
		handler := gzipMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("binary"))
		}))

		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Accept-Encoding", "gzip")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		if got := recorder.Header().Get("Content-Encoding"); got != "" {
			t.Fatalf("path %s: Content-Encoding = %q, want empty", path, got)
		}
		if recorder.Body.String() != "binary" {
			t.Fatalf("path %s: body should be passed through", path)
		}
	}
}
