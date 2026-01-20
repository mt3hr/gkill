package reps

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"image"
	stdDraw "image/draw"
	"image/jpeg"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "image/gif"
	_ "image/png"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // webp decode
	"golang.org/x/sync/singleflight"
)

var (
	thumbParamRe = regexp.MustCompile(`^(\d{1,4})x(\d{1,4})$`)

	// 同一サムネの同時生成を1回にまとめる
	thumbSF singleflight.Group

	// 生成の同時実行数を制限（CPU/IO暴走防止）
	thumbSem = make(chan struct{}, 3)
)

// NewThumbFileServer は dir 配下をサーブしつつ、?thumb=200x200 のときだけサムネを返す。
// - それ以外は base.ServeHTTP に委譲（既存挙動維持）
func NewThumbFileServer(dir string, base http.Handler) http.Handler {
	dir = filepath.Clean(os.ExpandEnv(dir))
	cacheDir := os.ExpandEnv(filepath.Join(gkill_options.CacheDir, "thumb_cache", filepath.Base(dir)))

	return &thumbFileServer{
		rootDir:  dir,
		cacheDir: cacheDir,
		base:     base,
		maxSize:  1024,
		jpegQ:    85,
	}
}

type thumbFileServer struct {
	rootDir  string
	cacheDir string
	base     http.Handler

	maxSize int
	jpegQ   int
}

func (t *thumbFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	thumb := r.URL.Query().Get("thumb")
	if thumb == "" {
		t.base.ServeHTTP(w, r)
		return
	}

	// サイズ解析
	tw, th, ok := parseThumb(thumb)
	if !ok || tw <= 0 || th <= 0 || tw > t.maxSize || th > t.maxSize {
		// 不正ならフォールバック（400でもOK）
		t.base.ServeHTTP(w, r)
		return
	}

	// URL Path（StripPrefix後）を安全に相対化
	rel, ok := cleanRelURLPath(r.URL.Path)
	if !ok || rel == "" {
		t.base.ServeHTTP(w, r)
		return
	}

	// デコード対応の拡張子だけサムネ処理（無駄なdecode回避）
	if !looksLikeDecodableImage(rel) {
		t.base.ServeHTTP(w, r)
		return
	}

	abs, ok := secureJoin(t.rootDir, rel)
	if !ok {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}

	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		t.base.ServeHTTP(w, r)
		return
	}

	thumbPath, etag, err := t.thumbPathFor(rel, st, tw, th)
	if err != nil {
		t.base.ServeHTTP(w, r)
		return
	}

	// キャッシュがあれば返す（ETagで304も返す）
	if fileExists(thumbPath) {
		serveThumbFile(w, r, thumbPath, etag)
		return
	}

	// 生成（同時生成まとめ + 同時実行制限）
	_, genErr, _ := thumbSF.Do(thumbPath, func() (any, error) {
		thumbSem <- struct{}{}
		defer func() { <-thumbSem }()

		if fileExists(thumbPath) {
			return nil, nil
		}
		if err := os.MkdirAll(filepath.Dir(thumbPath), 0o755); err != nil {
			return nil, err
		}
		return nil, generateThumbJpeg(abs, thumbPath, tw, th, t.jpegQ)
	})

	if genErr != nil {
		// 生成失敗時は元画像にフォールバック（運用安全）
		t.base.ServeHTTP(w, r)
		return
	}

	serveThumbFile(w, r, thumbPath, etag)
}

func parseThumb(s string) (int, int, bool) {
	m := thumbParamRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	w, _ := strconv.Atoi(m[1])
	h, _ := strconv.Atoi(m[2])
	return w, h, true
}

func looksLikeDecodableImage(rel string) bool {
	ext := strings.ToLower(filepath.Ext(rel))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

// stripPrefix後のURL pathを安全に正規化して、相対パスを返す
func cleanRelURLPath(p string) (string, bool) {
	// URLは "/" 区切りなので path.Clean を使う
	cp := path.Clean("/" + p)
	cp = strings.TrimPrefix(cp, "/")
	if cp == "" || strings.Contains(cp, "\x00") {
		return "", false
	}
	return cp, true
}

// rootDir から外へ出ないように join
func secureJoin(rootDir, rel string) (string, bool) {
	root := filepath.Clean(rootDir)
	full := filepath.Join(root, filepath.FromSlash(rel))
	full = filepath.Clean(full)

	// root配下のみ許可
	if full == root {
		return full, true
	}
	prefix := root + string(os.PathSeparator)
	if strings.HasPrefix(full, prefix) {
		return full, true
	}
	return "", false
}

func (t *thumbFileServer) thumbPathFor(rel string, st os.FileInfo, w, h int) (thumbPath string, etag string, err error) {
	// rel + mtime + size + w/h でキー化（元更新で別サムネになる）
	hh := sha1.Sum([]byte(rel))
	key := hex.EncodeToString(hh[:])

	ver := fmt.Sprintf("%d_%d", st.ModTime().Unix(), st.Size())
	name := fmt.Sprintf("%s_%s_%dx%d.jpg", key, ver, w, h)

	// 同じverなら同じETag（thumb=... のURLは変わらないので、ETagで整合を取る）
	etag = fmt.Sprintf(`W/"%s"`, name)

	return filepath.Join(t.cacheDir, name), etag, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func serveThumbFile(w http.ResponseWriter, r *http.Request, thumbPath string, etag string) {
	// ETagで更新検知（URLが変わらない設計なので immutable は避ける）
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, no-cache")
	w.Header().Set("Content-Type", "image/jpeg")

	if inm := r.Header.Get("If-None-Match"); inm != "" && inm == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	http.ServeFile(w, r, thumbPath)
}

// generateThumbJpeg: center-crop → resize → jpeg保存（atomic write）
func generateThumbJpeg(srcPath, dstPath string, w, h int, quality int) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}

	cropped := cropCenterToAspect(img, float64(w)/float64(h))
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	// 高品質スケール（初回だけ重い）
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), stdDraw.Over, nil)

	// atomic write
	tmp := dstPath + ".tmp"
	tf, err := os.Create(tmp)
	if err != nil {
		return err
	}

	encErr := jpeg.Encode(tf, dst, &jpeg.Options{Quality: quality})
	closeErr := tf.Close()

	if encErr != nil {
		_ = os.Remove(tmp)
		return encErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}

	// Windows対策：既存があれば消してからrename
	_ = os.Remove(dstPath)
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return nil
}

// aspect比に合わせてcenter-crop
func cropCenterToAspect(src image.Image, targetAspect float64) image.Image {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw <= 0 || sh <= 0 {
		return src
	}

	srcAspect := float64(sw) / float64(sh)

	var crop image.Rectangle
	if srcAspect > targetAspect {
		// 幅が広い：左右を削る
		newW := int(float64(sh) * targetAspect)
		if newW <= 0 {
			return src
		}
		x0 := (sw - newW) / 2
		crop = image.Rect(b.Min.X+x0, b.Min.Y, b.Min.X+x0+newW, b.Min.Y+sh)
	} else {
		// 高さが高い：上下を削る
		newH := int(float64(sw) / targetAspect)
		if newH <= 0 {
			return src
		}
		y0 := (sh - newH) / 2
		crop = image.Rect(b.Min.X, b.Min.Y+y0, b.Min.X+sw, b.Min.Y+y0+newH)
	}

	// SubImageできる型なら最速
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	if si, ok := src.(subImager); ok {
		return si.SubImage(crop)
	}

	// できない型はコピー
	out := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	stdDraw.Draw(out, out.Bounds(), src, crop.Min, stdDraw.Src)
	return out
}
