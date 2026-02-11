package reps

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"image"
	stdDraw "image/draw"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "image/gif"
	_ "image/png"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/rwcarlsen/goexif/exif"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // webp decode
	"golang.org/x/sync/singleflight"
)

var (
	existFFMPEG  = false
	existFFPROBE = false
)

func init() {
	var err error
	_, err = exec.LookPath("ffmpeg")
	existFFMPEG = err == nil
	_, err = exec.LookPath("ffprobe")
	existFFPROBE = err == nil
}

var (
	thumbParamRe = regexp.MustCompile(`^(\d{1,4})x(\d{1,4})$`)

	// 同一サムネの同時生成を1回にまとめる
	thumbSF singleflight.Group

	// 生成の同時実行数を制限（CPU/IO暴走防止）
	thumbSem = make(chan struct{}, runtime.NumCPU())
)

type ThumbGenerator interface {
	GenerateThumbCache(ctx context.Context, url string) error
}

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

func (t *thumbFileServer) GenerateThumbCache(ctx context.Context, queryURL string) error {
	queryURLObj, err := url.Parse(queryURL)
	if err != nil {
		err = fmt.Errorf("error at parse url %s: %w", queryURL, err)
		return err
	}

	thumb := queryURLObj.Query().Get("thumb")
	if thumb == "" {
		err = fmt.Errorf("error at get parse thumb size %s: %w", queryURL, err)
		return err
	}

	// サイズ解析
	tw, th, ok := parseThumb(thumb)
	if !ok || tw <= 0 || th <= 0 || tw > t.maxSize || th > t.maxSize {
		err = fmt.Errorf("error at get parse thumb size %s: %w", queryURL, err)
		return err
	}

	// URL Path（StripPrefix後）を安全に相対化
	rel, ok := cleanRelURLPath(queryURLObj.Path)
	if !ok || rel == "" {
		err := fmt.Errorf("illegal url path %s", queryURLObj.Path)
		return err
	}

	// 動画サムネのリクエストの場合
	isVideo := queryURLObj.Query().Get("is_video") == "true"

	// 対象拡張子だけサムネ処理（無駄な処理回避）
	if !looksLikeThumbTarget(rel, isVideo) {
		return nil
	}

	abs, ok := secureJoin(t.rootDir, rel)
	if !ok {
		err := fmt.Errorf("bad path %s", queryURL)
		return err
	}

	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		return nil
	}

	thumbPath, _, err := t.thumbPathFor(rel, st, tw, th)
	if err != nil {
		return nil
	}

	// キャッシュがあれば返す（ETagで304も返す）
	if fileExists(thumbPath) {
		return nil
	}

	// 生成
	if fileExists(thumbPath) {
		return nil
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
		if isVideo {
			if !existFFMPEG || !existFFPROBE {
				return nil, fmt.Errorf("ffmpeg/ffprobe not available")
			}
			return nil, generateVideoThumbJpeg(ctx, abs, thumbPath, tw, th)
		}
		return nil, generateThumbJpeg(abs, thumbPath, tw, th, t.jpegQ)
	})

	if genErr != nil {
		return genErr
	}
	return nil
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

	// 動画サムネのリクエストの場合
	isVideo := r.URL.Query().Get("is_video") == "true"

	// 対象拡張子だけサムネ処理（無駄な処理回避）
	if !looksLikeThumbTarget(rel, isVideo) {
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
		if isVideo {
			if !existFFMPEG || !existFFPROBE {
				return nil, fmt.Errorf("ffmpeg/ffprobe not available")
			}
			return nil, generateVideoThumbJpeg(r.Context(), abs, thumbPath, tw, th)
		}
		return nil, generateThumbJpeg(abs, thumbPath, tw, th, t.jpegQ)
	})

	if genErr != nil {
		// thumb要求（特に動画poster）は、動画本体へフォールバックすると Content-Type が壊れる。
		// 画像は既存挙動維持でフォールバックする。
		if isVideo {
			http.Error(w, genErr.Error(), http.StatusInternalServerError)
			return
		}
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

// looksLikeThumbTarget returns true if the request should be handled as a thumbnail generation.
// - isVideo=true : allow common video extensions
// - otherwise    : allow decodable images only
func looksLikeThumbTarget(rel string, isVideo_ bool) bool {
	if isVideo_ {
		return isVideo(rel)
	}
	return isImage(rel)
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

// generateVideoThumbJpeg creates a JPEG thumbnail for a video using ffmpeg.
// Policy: take a frame at ~10% of the duration (fallback: 1s), then scale+center-crop to WxH.
// It writes atomically to dstPath.
func generateVideoThumbJpeg(ctx context.Context, srcPath, dstPath string, w, h int) error {
	// Decide timestamp seconds: duration * 0.1
	sec := 1.0
	if existFFPROBE {
		if d, err := ffprobeDurationSeconds(ctx, srcPath); err == nil && d > 0 {
			sec = d * 0.10
			if sec < 0 {
				sec = 0
			}
		}
	}

	// scale while preserving aspect ratio, then crop center
	// force_original_aspect_ratio=increase ensures both dimensions cover the target, then crop.
	vf := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", w, h, w, h)

	tmp := dstPath + ".tmp"
	_ = os.Remove(tmp)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-ss", fmt.Sprintf("%.3f", sec),
		"-i", srcPath,
		"-frames:v", "1",
		"-vf", vf,
		"-q:v", "2",
		"-f", "image2",
		tmp,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("ffmpeg thumb failed: %w: %s", err, out.String())
	}

	_ = os.Remove(dstPath)
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func ffprobeDurationSeconds(ctx context.Context, srcPath string) (float64, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(probeCtx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		srcPath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w: %s", err, out.String())
	}
	s := strings.TrimSpace(out.String())
	if s == "" {
		return 0, fmt.Errorf("duration empty")
	}
	d, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return d, nil
}

// generateThumbJpeg: center-crop → resize → jpeg保存（atomic write）
func generateThumbJpeg(srcPath, dstPath string, w, h int, quality int) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	// 1) EXIF Orientation を読む（失敗したら 1 扱い）
	orient := 1
	if _, err := f.Seek(0, io.SeekStart); err == nil {
		if x, err := exif.Decode(f); err == nil {
			if tag, err := x.Get(exif.Orientation); err == nil {
				if v, err := tag.Int(0); err == nil && 1 <= v && v <= 8 {
					orient = v
				}
			}
		}
	}

	// 2) Decode に備えて先頭へ戻す
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	img, format, err := image.Decode(f)
	if err != nil {
		return err
	}

	// 3) JPEG のときだけ Orientation を適用
	if format == "jpeg" && orient != 1 {
		img = applyExifOrientation(img, orient)
	}

	// 4) いつもの crop → resize
	cropped := cropCenterToAspect(img, float64(w)/float64(h))
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), stdDraw.Over, nil)

	// 5) atomic write
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

	_ = os.Remove(dstPath)
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func applyExifOrientation(src image.Image, orient int) image.Image {
	s := toNRGBA(src)
	sw, sh := s.Bounds().Dx(), s.Bounds().Dy()

	// dst のサイズ（90/270系は入れ替わる）
	dw, dh := sw, sh
	if orient >= 5 && orient <= 8 {
		dw, dh = sh, sw
	}
	d := image.NewNRGBA(image.Rect(0, 0, dw, dh))

	// src を走査して dst へ転送（1回だけなのでシンプル優先）
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			si := y*s.Stride + x*4

			var dx, dy int
			switch orient {
			case 1: // normal
				dx, dy = x, y
			case 2: // mirror horizontal
				dx, dy = sw-1-x, y
			case 3: // rotate 180
				dx, dy = sw-1-x, sh-1-y
			case 4: // mirror vertical
				dx, dy = x, sh-1-y
			case 5: // transpose
				dx, dy = y, x
			case 6: // rotate 90 CW
				dx, dy = sh-1-y, x
			case 7: // transverse
				dx, dy = sh-1-y, sw-1-x
			case 8: // rotate 270 CW
				dx, dy = y, sw-1-x
			default:
				dx, dy = x, y
			}

			di := dy*d.Stride + dx*4
			copy(d.Pix[di:di+4], s.Pix[si:si+4])
		}
	}
	return d
}

func toNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok && n.Rect.Min.X == 0 && n.Rect.Min.Y == 0 {
		return n
	}
	b := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	stdDraw.Draw(dst, dst.Bounds(), img, b.Min, stdDraw.Src)
	return dst
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
