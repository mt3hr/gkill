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
	"sync/atomic"
	"time"

	_ "image/gif"
	_ "image/png"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
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
	_, existFFMPEG = findInPath("ffmpeg")
	_, existFFPROBE = findInPath("ffprobe")
}

func findInPath(name string) (string, bool) {
	exts := []string{""}
	if runtime.GOOS == "windows" {
		exts = []string{".exe", ".cmd", ".bat", ""}
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		for _, ext := range exts {
			full := filepath.Join(dir, name+ext)
			if info, err := os.Stat(full); err == nil && !info.IsDir() {
				return full, true
			}
		}
	}
	return "", false
}

var (
	thumbParamRe = regexp.MustCompile(`^(\d{1,4})x(\d{1,4})$`)

	// 同一サムネの同時生成を1回にまとめる
	thumbSF singleflight.Group

	// 生成の同時実行数を制限（CPU/IO暴走防止）
	thumbSem = make(chan struct{}, runtime.NumCPU())
)

// ThumbGenerator はサムネイルキャッシュの生成と、生成済みかどうかの判定を提供します。
type ThumbGenerator interface {
	// GenerateThumbCache は配信URLと同じ形の文字列からサムネイルキャッシュを生成します。
	// 対象がサムネイル化できない種別だったり、すでに生成済みだったときは何もせずnilを返します。
	GenerateThumbCache(ctx context.Context, url string) error

	// GenerateThumbCacheFor はURLを介さずにサムネイルキャッシュを生成します。
	// rel はリポジトリ内の相対パス、st はその実ファイルの情報です。
	//
	// 一括生成はこちらを使ってください。GenerateThumbCache は1件ごとに
	// URL文字列を組み立てて即座に解析し直すので、数十万件規模では無視できません。
	GenerateThumbCacheFor(ctx context.Context, rel string, st os.FileInfo, isVideo bool, w int, h int) error

	// ThumbCacheName はリポジトリ内の相対パスとファイルサイズから、キャッシュのファイル名を返します。
	// CachedThumbNames が返す集合と突き合わせるために使います。
	ThumbCacheName(rel string, size int64, w int, h int) string

	// CachedThumbNames は生成済みサムネイルのファイル名の集合を、ディレクトリ1回の列挙で返します。
	// キャッシュディレクトリがまだ無いときは空の集合を返します（エラーにしません）。
	//
	// 1件ずつ os.Stat すると数万件のリポジトリで数十秒かかるものが、
	// 1回の列挙なら数十ミリ秒で済みます。
	CachedThumbNames() (map[string]struct{}, error)
}

// NewThumbFileServer は dir 配下をサーブしつつ、?thumb=200x200 のときだけサムネを返す。
// - それ以外は base.ServeHTTP に委譲（既存挙動維持）
func NewThumbFileServer(userID string, dir string, base http.Handler) http.Handler {
	dir = filepath.Clean(os.ExpandEnv(dir))
	cacheDir := derivedCacheDirForUser("thumb_cache", userID, dir)

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
		err = fmt.Errorf("error at get parse thumb size %s: thumb parameter is empty", queryURL)
		return err
	}

	// サイズ解析
	tw, th, ok := parseThumb(thumb)
	if !ok || tw <= 0 || th <= 0 || tw > t.maxSize || th > t.maxSize {
		err = fmt.Errorf("error at get parse thumb size %s: invalid thumb dimensions", queryURL)
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

	abs, ok := SecureJoin(t.rootDir, rel)
	if !ok {
		err := fmt.Errorf("bad path %s", queryURL)
		return err
	}

	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		return nil
	}

	return t.GenerateThumbCacheFor(ctx, rel, st, isVideo, tw, th)
}

// GenerateThumbCacheFor はURLを介さずにサムネイルキャッシュを生成します。
// 契約は ThumbGenerator.GenerateThumbCacheFor を参照。
func (t *thumbFileServer) GenerateThumbCacheFor(ctx context.Context, rel string, st os.FileInfo, isVideo bool, tw int, th int) error {
	if tw <= 0 || th <= 0 || tw > t.maxSize || th > t.maxSize {
		return fmt.Errorf("invalid thumb dimensions %dx%d for %s", tw, th, rel)
	}
	if st == nil || st.IsDir() {
		return nil
	}
	if !looksLikeThumbTarget(rel, isVideo) {
		return nil
	}

	abs, ok := SecureJoin(t.rootDir, rel)
	if !ok {
		return fmt.Errorf("bad path %s", rel)
	}

	thumbPath := filepath.Join(t.cacheDir, t.ThumbCacheName(rel, st.Size(), tw, th))
	if fileExists(thumbPath) {
		return nil
	}

	// 生成に失敗した印があれば、二度と挑まない。
	// 印が無いと、デコードできないファイルは一括生成のたびに全件やり直しになる。
	// 消したいときは clear_cache thumb <利用者ID>（キャッシュディレクトリごと消える）。
	if fileExists(thumbPath + thumbFailedMarkerSuffix) {
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
		return nil, generateThumbJpeg(ctx, abs, thumbPath, tw, th, t.jpegQ)
	})

	if genErr != nil {
		markThumbFailed(ctx, thumbPath, genErr)
		return genErr
	}
	return nil
}

// CachedThumbNames は生成済みサムネイルのファイル名の集合を返します。
// 契約は ThumbGenerator.CachedThumbNames を参照。
func (t *thumbFileServer) CachedThumbNames() (map[string]struct{}, error) {
	names := map[string]struct{}{}
	entries, err := os.ReadDir(t.cacheDir)
	if err != nil {
		// まだ1件も生成していないだけ。呼び出し元は「全部未生成」として進めればよい
		if os.IsNotExist(err) {
			return names, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// 失敗の印は「生成済み」ではない。ここへ混ぜると、
		// 一括生成側が「サムネイルがある」と誤認して数を取り違える。
		// 印そのものは GenerateThumbCacheFor が入口で見て早く戻る。
		if strings.HasSuffix(entry.Name(), thumbFailedMarkerSuffix) {
			continue
		}
		names[entry.Name()] = struct{}{}
	}
	return names, nil
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

	abs, ok := SecureJoin(t.rootDir, rel)
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

	// 生成に失敗した印があれば、二度と挑まない。
	// 印が無いと、デコードできないファイルは表示のたびに
	// EXIFパース + Decode + ffmpeg をやり直すことになる。
	if fileExists(thumbPath + thumbFailedMarkerSuffix) {
		if isVideo {
			// 動画posterは原本へフォールバックすると Content-Type が壊れる（下と同じ理由）。
			http.Error(w, "thumb generation failed", http.StatusInternalServerError)
			return
		}
		t.base.ServeHTTP(w, r)
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
		return nil, generateThumbJpeg(r.Context(), abs, thumbPath, tw, th, t.jpegQ)
	})

	if genErr != nil {
		markThumbFailed(r.Context(), thumbPath, genErr)
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

// browserRenderableWithoutThumbExts は、ブラウザが img でそのまま描けるのに
// Go の image.Decode でも ffmpeg でもラスタ化できない拡張子。
//
// サムネイルを作ろうとすると必ず失敗するので、生成を試みずに原本を配信させる。
// SVG はベクタで元から小さく、縮小して配る意味も薄い。
var browserRenderableWithoutThumbExts = map[string]struct{}{
	".svg": {},
}

func isBrowserRenderableWithoutThumb(filename string) bool {
	_, ok := browserRenderableWithoutThumbExts[strings.ToLower(filepath.Ext(filename))]
	return ok
}

// looksLikeThumbTarget returns true if the request should be handled as a thumbnail generation.
// - isVideo=true : allow common video extensions
// - otherwise    : allow images, except the ones the browser draws better than we can
func looksLikeThumbTarget(rel string, isVideo_ bool) bool {
	if isVideo_ {
		return isVideo(rel)
	}
	if isBrowserRenderableWithoutThumb(rel) {
		return false
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

// SecureJoin は rootDir から外へ出ないように join する。
// 結果が rootDir の**真下**でなければ ok=false を返す。
//
// rootDir 自身（rel が "." や "" や "foo/.." のとき）も ok=false にする。
// 呼び出し元は6箇所ともファイル1件のパスを求めており、ディレクトリ自身を指すパスを
// 受け取っても書き込みにも配信にも使えない（アップロードなら os.Rename が
// ディレクトリを潰そうとして失敗し、配信なら中身の無いパスを返す）。
// ZIP展開のディレクトリエントリだけは以前 ok=true で MkdirAll に渡っていたが、
// 対象は展開先そのもので既に存在するため、弾いても実質no-opになる。
//
// この「rootDir 自身は許可しない」は CodeQL 対策も兼ねる。
// go/path-injection は strings.HasPrefix によるガードはバリアとして認識するが、
// full == root の分岐は認識せず、実際には安全なアップロード経路が
// 3件のアラートになっていた（2026-08-22）。
func SecureJoin(rootDir, rel string) (string, bool) {
	root := filepath.Clean(rootDir)
	full := filepath.Join(root, filepath.FromSlash(rel))
	full = filepath.Clean(full)

	// root の真下のみ許可する
	prefix := root + string(os.PathSeparator)
	if strings.HasPrefix(full, prefix) {
		return full, true
	}
	return "", false
}

// thumbFailedMarkerSuffix はサムネイル生成に失敗した印のファイル名につく接尾辞。
const thumbFailedMarkerSuffix = ".failed"

// markThumbFailed はサムネイル生成に失敗した印を残します。
//
// ctx が切れているときは印を残しません。HTTP経由ではブラウザが待ちきれずに
// 接続を切ると ffmpeg も一緒に落ちるので、それを恒久的な失敗として焼いてはいけません。
// 互換動画側の markCompatFailed と同じ判断です。
func markThumbFailed(ctx context.Context, thumbPath string, cause error) {
	if ctx.Err() != nil {
		return
	}
	markerPath := thumbPath + thumbFailedMarkerSuffix
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		slog.Log(ctx, gkill_log.Debug, "error at make thumb failed marker directory", "error", fmt.Sprintf("%q", err))
		return
	}
	content := fmt.Sprintf("%s\n%v\n", time.Now().Format(time.RFC3339), cause)
	if err := os.WriteFile(markerPath, []byte(content), 0o644); err != nil {
		slog.Log(ctx, gkill_log.Debug, "error at write thumb failed marker", "error", fmt.Sprintf("%q", err))
	}
}

// ThumbCacheName はキャッシュのファイル名を返します。
// 契約は ThumbGenerator.ThumbCacheName を参照。
func (t *thumbFileServer) ThumbCacheName(rel string, size int64, w int, h int) string {
	// rel + size + w/h でキー化
	hh := sha1.Sum([]byte(rel))
	key := hex.EncodeToString(hh[:])

	return fmt.Sprintf("%s_%d_%dx%d.jpg", key, size, w, h)
}

func (t *thumbFileServer) thumbPathFor(rel string, st os.FileInfo, w, h int) (thumbPath string, etag string, err error) {
	name := t.ThumbCacheName(rel, st.Size(), w, h)
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

// thumbTmpSeq は同一プロセス内の一時ファイル名を分けるための連番。
var thumbTmpSeq atomic.Uint64

// thumbTmpPath は書きかけのサムネイルを置く一時ファイルのパスを返します。
//
// プロセスIDと連番を挟むのは、**同じサムネイルを複数のプロセスが同時に作りうる**ため。
// サーバ本体・MCPサーバ・generate_thumb_cache は別プロセスで同じキャッシュ置き場を共有していて、
// singleflight はプロセスの中しかまとめられない。固定名の `.tmp` を共有すると、
// 片方の os.Rename がもう片方の消した tmp を掴んで失敗し、
// 実際には作れるサムネイルに失敗の印が焼かれる。
func thumbTmpPath(dstPath string) string {
	return fmt.Sprintf("%s.%d.%d.tmp", dstPath, os.Getpid(), thumbTmpSeq.Add(1))
}

// finalizeFFmpegThumbOutput は ffmpeg が書いたはずの tmp を dstPath へ atomic に移します。
//
// ffmpeg は exit 0 でも出力を書かないことがある。-ss が素材の末尾を越えたときがそれで、
// ffprobe が duration を返すのに、その1割の位置ではフレームが取れない古い .MOV が実在する。
// 確かめずに os.Rename まで進むと「指定されたファイルが見つかりません」という、
// ffmpeg が何もしなかったことを隠したエラーになる。
//
// 出力が無いときに ffmpeg 自身が非ゼロで終わるかどうかは、ビルドと素材で変わる。
// 終了コードを当てにせず、出力の有無だけで判定すること。
func finalizeFFmpegThumbOutput(srcPath, tmp, dstPath, ffmpegOutput string) error {
	if st, err := os.Stat(tmp); err != nil || st.Size() == 0 {
		_ = os.Remove(tmp)
		return fmt.Errorf("ffmpeg thumb wrote no output for %s: %s", filepath.Base(srcPath), ffmpegOutput)
	}

	_ = os.Remove(dstPath)
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// runFFmpegThumb は ffmpeg で1フレーム抜き出し、dstPath へ atomic に書きます。
// seekSec が正のときだけ -ss を付けます（静止画には付けない）。
func runFFmpegThumb(ctx context.Context, srcPath, dstPath string, w, h int, seekSec float64) error {
	tmp := thumbTmpPath(dstPath)
	_ = os.Remove(tmp)

	args := []string{"-hide_banner", "-loglevel", "error", "-y"}
	if seekSec > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", seekSec))
	}
	// scale while preserving aspect ratio, then crop center
	// force_original_aspect_ratio=increase ensures both dimensions cover the target, then crop.
	args = append(args,
		"-i", srcPath,
		"-frames:v", "1",
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", w, h, w, h),
		"-q:v", "2",
		"-f", "image2",
		tmp,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("ffmpeg thumb failed: %w: %s", err, out.String())
	}

	return finalizeFFmpegThumbOutput(srcPath, tmp, dstPath, out.String())
}

// runFFmpegDecodeFull は ffmpeg で1フレームを原寸のまま JPEG へ書き出します。
// 縮小フィルタを付けられない入力（タイル分割された HEIF など）のための逃げ道です。
func runFFmpegDecodeFull(ctx context.Context, srcPath, dstPath string) error {
	tmp := thumbTmpPath(dstPath)
	_ = os.Remove(tmp)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i", srcPath,
		"-frames:v", "1",
		"-q:v", "2",
		"-f", "image2",
		tmp,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("ffmpeg decode failed: %w: %s", err, out.String())
	}

	return finalizeFFmpegThumbOutput(srcPath, tmp, dstPath, out.String())
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

	err := runFFmpegThumb(ctx, srcPath, dstPath, w, h, sec)
	if err == nil || sec <= 0 {
		return err
	}

	// -ss が素材の末尾を越えていると1フレームも取れない。
	// duration を返さない古い .MOV と、動画の名前が付いた静止画がここへ来る。
	// 先頭から取り直せば通るので、1回だけやり直す。
	if retryErr := runFFmpegThumb(ctx, srcPath, dstPath, w, h, 0); retryErr != nil {
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
//
// image.Decode で読めなかったものは ffmpeg へ落とす。isImage が通す37拡張子のうち
// Go にデコーダがあるのは jpeg/png/gif/webp の4形式だけで、
// heic/avif/bmp/ico/tiff は必ずそこで失敗する。ffmpeg は拡張子ではなく中身で
// 形式を決めるので、拡張子が実体と食い違っているファイルもここで拾える。
func generateThumbJpeg(ctx context.Context, srcPath, dstPath string, w, h int, quality int) error {
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
		if !existFFMPEG {
			return err
		}
		return generateThumbViaFFmpeg(ctx, srcPath, dstPath, w, h, quality, err)
	}

	// 3) JPEG のときだけ Orientation を適用
	if format == "jpeg" && orient != 1 {
		img = applyExifOrientation(img, orient)
	}

	return writeThumbFromImage(img, dstPath, w, h, quality)
}

// generateThumbViaFFmpeg は image.Decode で読めなかった画像を ffmpeg で起こします。
// decodeErr は Go 側のデコード失敗で、ffmpeg でも作れなかったときに包んで返します。
func generateThumbViaFFmpeg(ctx context.Context, srcPath, dstPath string, w, h int, quality int, decodeErr error) error {
	// まずは縮小まで ffmpeg の中で済ませる。プロセス1回で終わるので速い
	fastErr := runFFmpegThumb(ctx, srcPath, dstPath, w, h, 0)
	if fastErr == nil {
		return nil
	}

	// タイルに分割された HEIF は、ffmpeg が内部で complex filtergraph を組んで
	// タイルを貼り合わせてから1枚の画像にする。そこへこちらの簡易フィルタ（-vf）を足すと
	// 「Simple and complex filtering cannot be used together」で拒否され、
	// **何も書かないまま exit 0** で終わる。
	// -vf を外せば読めるので、原寸で書き出してから Go 側で縮小する。
	full := thumbTmpPath(dstPath)
	defer func() { _ = os.Remove(full) }()
	if err := runFFmpegDecodeFull(ctx, srcPath, full); err != nil {
		return fmt.Errorf("%w (ffmpeg fallback: %v)", decodeErr, fastErr)
	}

	ff, err := os.Open(full)
	if err != nil {
		return fmt.Errorf("%w (ffmpeg fallback: %v)", decodeErr, err)
	}
	img, _, err := image.Decode(ff)
	if closeErr := ff.Close(); closeErr != nil {
		slog.Log(ctx, gkill_log.Debug, "error at close ffmpeg output", "error", fmt.Sprintf("%q", closeErr))
	}
	if err != nil {
		return fmt.Errorf("%w (ffmpeg fallback: %v)", decodeErr, err)
	}
	return writeThumbFromImage(img, dstPath, w, h, quality)
}

// writeThumbFromImage は中央切り抜き → 縮小 → JPEG保存（atomic write）を行います。
func writeThumbFromImage(img image.Image, dstPath string, w, h int, quality int) error {
	cropped := cropCenterToAspect(img, float64(w)/float64(h))
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), stdDraw.Over, nil)

	tmp := thumbTmpPath(dstPath)
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
	for y := range sh {
		for x := range sw {
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
