package gkill_server_api

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// gzipMiddleware は /api/ 配下の応答をgzip圧縮するミドルウェアを返します。
//
// 検索応答(get_kyousなど)は数十万件で100MB級になり、WAN越しでは転送時間が支配的になるため。
// 対象を /api/ に限定しているのは、/files/ と /zip_cache/ が動画のRange配信・
// 既圧縮メディアを扱うためです(圧縮するとRangeが壊れ、メディアは縮まない)。
// 静的アセット(embed配信)はブラウザキャッシュが効くので対象にしていません。
// 圧縮レベルはBestSpeed。ストリーミング圧縮なので転送と重なり、
// サーバ側の追加レイテンシは実測で転送短縮に対して十分小さいです。
func gzipMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/") ||
				!strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// 圧縮後のサイズは書き込み時点で不明なのでContent-Lengthは付けられない
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Add("Vary", "Accept-Encoding")

			gzipWriter, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
			if err != nil {
				// BestSpeedは有効値なのでここへは来ないが、来たら素通しする
				w.Header().Del("Content-Encoding")
				next.ServeHTTP(w, r)
				return
			}
			defer gzipWriter.Close()

			next.ServeHTTP(&gzipResponseWriter{ResponseWriter: w, gzipWriter: gzipWriter}, r)
		})
	}
}

// gzipResponseWriter はWriteをgzip圧縮に通すResponseWriterラッパです。
type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter *gzip.Writer
}

// Write は応答ボディをgzipライタへ書きます。
func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	return g.gzipWriter.Write(data)
}

// Flush はgzipの内部バッファを掃き出してから下位のFlusherへ委譲します。
// ストリーミング応答(json.NewEncoderの逐次書き込み)を途中で送出できるようにするためのものです。
func (g *gzipResponseWriter) Flush() {
	g.gzipWriter.Flush()
	if flusher, ok := g.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// 静的検査用: FlusherとWriterを満たしていること
var (
	_ http.ResponseWriter = (*gzipResponseWriter)(nil)
	_ http.Flusher        = (*gzipResponseWriter)(nil)
	_ io.Writer           = (*gzipResponseWriter)(nil)
)
