package reps

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
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
	"sync"
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

	// existVIPS は libvips の CLI（vips）が PATH にあるか。
	// 静止画のサムネイルは、あればまず vips で作る（generateThumbJpeg。Go で読めて小さい画像は除く）。
	// 無いのは失敗ではない（Go → ffmpeg の経路で作る）ので、印にも積まない。
	existVIPS = false
)

// vipsBin は起動する libvips CLI の名前。
// テストが存在しない名前へ差し替えて「vips が失敗したら Go → ffmpeg へ落ちる」経路を通す。
var vipsBin = "vips"

// errFFToolsNotAvailable は ffmpeg / ffprobe が PATH に無くて動画サムネイルを作れないこと。
//
// これは「そのファイルが変換できない」ではなく「この環境では今どの動画も変換できない」なので、
// markThumbFailed の恒久マーカーからは外す（互換動画側が ffmpeg 不在でマーカーを残さず
// 原本へフォールバックするのと同じ判断）。焼いてしまうと ffmpeg が使えるようになっても
// 二度と生成されず、clear_cache thumb でキャッシュを丸ごと捨てるまで直らない。
//
// 2026-09-06 に実際にそうなった。本番サービスは LocalSystem 起動でシステムのPATHしか見えず、
// ffmpeg は利用者のPATHにしか入っていなかったので、ブラウザで一覧を開いた1回で
// 動画123件ぶんのマーカーが焼き付き、以後 generate_thumb_cache を何度回しても作られなくなった。
var errFFToolsNotAvailable = errors.New("ffmpeg/ffprobe not available")

func init() {
	_, existFFMPEG = findInPath("ffmpeg")
	_, existFFPROBE = findInPath("ffprobe")
	_, existVIPS = findInPath(vipsBin)
}

// logThumbBackendsOnce は、最初のサムネイル生成のときに使える外部ツールを Info で1行残す。
//
// init() の時点では gkill_log がまだ開いていないので、ここまで遅らせる。
// 本番サービスは LocalSystem 起動でシステムの PATH しか見えず、利用者の PATH に入れた
// ffmpeg / vips は見えない（2026-09-06 に ffmpeg で実際にそうなった）。
// 「見えていない」ことが失敗ではなく速度の差としてしか現れないので、ログで分かるようにする。
var logThumbBackendsOnce sync.Once

func logThumbBackends(ctx context.Context) {
	logThumbBackendsOnce.Do(func() {
		slog.Log(ctx, gkill_log.Info, "thumbnail backends detected", "vips", existVIPS, "ffmpeg", existFFMPEG, "ffprobe", existFFPROBE)
	})
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

	// CachedThumbNames は生成済みサムネイルのファイル名の集合と、生成に失敗した印の
	// ついているファイル名の集合を、ディレクトリ1回の列挙で返します。
	// failed の名前は印の接尾辞（.failed）を剥いだもので、generated と同じ形です。
	// キャッシュディレクトリがまだ無いときはどちらも空の集合を返します（エラーにしません）。
	//
	// 1件ずつ os.Stat すると数万件のリポジトリで数十秒かかるものが、
	// 1回の列挙なら数十ミリ秒で済みます。失敗の印も同じ列挙から取れるので、
	// 一括生成は印のある対象を goroutine へ投入せずに済みます。
	CachedThumbNames() (generated map[string]struct{}, failed map[string]struct{}, err error)
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
		logThumbBackends(ctx)
		if isVideo {
			if !existFFMPEG || !existFFPROBE {
				return nil, errFFToolsNotAvailable
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

// CachedThumbNames は生成済みサムネイルのファイル名の集合と、失敗の印のある名前の集合を返します。
// 契約は ThumbGenerator.CachedThumbNames を参照。
func (t *thumbFileServer) CachedThumbNames() (map[string]struct{}, map[string]struct{}, error) {
	generated := map[string]struct{}{}
	failed := map[string]struct{}{}
	entries, err := os.ReadDir(t.cacheDir)
	if err != nil {
		// まだ1件も生成していないだけ。呼び出し元は「全部未生成」として進めればよい
		if os.IsNotExist(err) {
			return generated, failed, nil
		}
		return nil, nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// 失敗の印は「生成済み」ではない。generated へ混ぜると、
		// 一括生成側が「サムネイルがある」と誤認して数を取り違える。
		// 別の集合で返し、一括生成側が印のある対象を投入前に外せるようにする。
		// HTTP 経路は GenerateThumbCacheFor / ServeHTTP の入口で印を見る。
		if name, ok := strings.CutSuffix(entry.Name(), thumbFailedMarkerSuffix); ok {
			failed[name] = struct{}{}
			continue
		}
		generated[entry.Name()] = struct{}{}
	}
	return generated, failed, nil
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
		logThumbBackends(r.Context())
		if isVideo {
			if !existFFMPEG || !existFFPROBE {
				return nil, errFFToolsNotAvailable
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
//
// ffmpeg / ffprobe がそもそも無いときも残しません。環境が変われば作れるようになるものを
// 焼くと、ffmpeg を入れ直しても二度と生成されなくなります（errFFToolsNotAvailable を参照）。
func markThumbFailed(ctx context.Context, thumbPath string, cause error) {
	if ctx.Err() != nil {
		return
	}
	if errors.Is(cause, errFFToolsNotAvailable) {
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

// thumbTmpPathExt は thumbTmpPath に拡張子を足した一時ファイルのパスを返します。
//
// vips は出力ファイルの拡張子でセーバを選ぶので、`.tmp` で終わる名前を渡すと
// 「unsupported」で何も書かずに失敗する。`.tmp.jpg` のように JPEG の拡張子で終える。
// 列挙（CachedThumbNames）にはキャッシュ名と一致しない名前として現れるだけで害は無い。
func thumbTmpPathExt(dstPath string, ext string) string {
	return thumbTmpPath(dstPath) + ext
}

// finalizeExternalThumbOutput は外部ツール（ffmpeg / vips）が書いたはずの tmp を dstPath へ atomic に移します。
//
// ffmpeg は exit 0 でも出力を書かないことがある。-ss が素材の末尾を越えたときがそれで、
// ffprobe が duration を返すのに、その1割の位置ではフレームが取れない古い .MOV が実在する。
// 確かめずに os.Rename まで進むと「指定されたファイルが見つかりません」という、
// ffmpeg が何もしなかったことを隠したエラーになる。
//
// 出力が無いときに ffmpeg 自身が非ゼロで終わるかどうかは、ビルドと素材で変わる。
// 終了コードを当てにせず、出力の有無だけで判定すること。vips も同じ判定に通す。
func finalizeExternalThumbOutput(tool, srcPath, tmp, dstPath, toolOutput string) error {
	if st, err := os.Stat(tmp); err != nil || st.Size() == 0 {
		_ = os.Remove(tmp)
		return fmt.Errorf("%s thumb wrote no output for %s: %s", tool, filepath.Base(srcPath), toolOutput)
	}

	_ = os.Remove(dstPath)
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// ffmpegBaseArgs は ffmpeg を非対話で起こすときに毎回付ける先頭の引数です。
//
// -nostdin は「標準入力から対話コマンドを読む」動きを止める。exec.Command の stdin は
// nil（/dev/null）なので実害は出ていないが、サービス起動や将来の配線の違いで
// 入力が繋がったときに ffmpeg が黙って待ちに入るのを避ける。
var ffmpegBaseArgs = []string{"-hide_banner", "-loglevel", "error", "-nostdin", "-y"}

// ffmpegThumbArgs は ffmpeg で1フレーム抜き出して縮小・中央切り抜きする引数列を組み立てます。
//
// singleThread のとき、復号（-i の前の -threads）・フィルタ（-filter_threads）・
// 符号化（出力側の -threads）をすべて1スレッドにする。静止画は1フレームなので
// フレーム並列は効かず、thumbSem が NumCPU 個のプロセスを同時に走らせる以上、
// 中で NumCPU 本ずつスレッドを立てると NumCPU² のコンテキストスイッチになるだけ。
// 動画のサムネイルは -ss の先読みで数十〜数百フレームを復号するので、
// HTTP 経路の待ち時間を優先して復号スレッドは ffmpeg の既定に任せる（フィルタは1で足りる）。
func ffmpegThumbArgs(srcPath, tmpPath string, w, h int, seekSec float64, singleThread bool) []string {
	args := append([]string{}, ffmpegBaseArgs...)
	if seekSec > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", seekSec))
	}
	if singleThread {
		args = append(args, "-threads", "1")
	}
	// scale while preserving aspect ratio, then crop center
	// force_original_aspect_ratio=increase ensures both dimensions cover the target, then crop.
	args = append(args,
		"-i", srcPath,
		"-frames:v", "1",
		"-filter_threads", "1",
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d", w, h, w, h),
	)
	if singleThread {
		args = append(args, "-threads", "1")
	}
	return append(args,
		"-q:v", "2",
		"-f", "image2",
		tmpPath,
	)
}

// ffmpegDecodeFullArgs は ffmpeg で1フレームを原寸のまま JPEG へ書き出す引数列を組み立てます。
// 常に1スレッド。タイル分割された HEIF は ffmpeg がタイルを貼り合わせる filtergraph を
// 内部で組むので、-filter_threads がここで効く。
func ffmpegDecodeFullArgs(srcPath, tmpPath string) []string {
	args := append([]string{}, ffmpegBaseArgs...)
	return append(args,
		"-threads", "1",
		"-i", srcPath,
		"-frames:v", "1",
		"-filter_threads", "1",
		"-threads", "1",
		"-q:v", "2",
		"-f", "image2",
		tmpPath,
	)
}

// runFFmpegThumb は ffmpeg で1フレーム抜き出し、dstPath へ atomic に書きます。
// seekSec が正のときだけ -ss を付けます（静止画には付けない）。
// singleThread の意味は ffmpegThumbArgs を参照。
func runFFmpegThumb(ctx context.Context, srcPath, dstPath string, w, h int, seekSec float64, singleThread bool) error {
	tmp := thumbTmpPath(dstPath)
	_ = os.Remove(tmp)

	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegThumbArgs(srcPath, tmp, w, h, seekSec, singleThread)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("ffmpeg thumb failed: %w: %s", err, out.String())
	}

	return finalizeExternalThumbOutput("ffmpeg", srcPath, tmp, dstPath, out.String())
}

// runFFmpegDecodeFull は ffmpeg で1フレームを原寸のまま JPEG へ書き出します。
// 縮小フィルタを付けられない入力（タイル分割された HEIF など）のための逃げ道です。
func runFFmpegDecodeFull(ctx context.Context, srcPath, dstPath string) error {
	tmp := thumbTmpPath(dstPath)
	_ = os.Remove(tmp)

	cmd := exec.CommandContext(ctx, "ffmpeg", ffmpegDecodeFullArgs(srcPath, tmp)...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("ffmpeg decode failed: %w: %s", err, out.String())
	}

	return finalizeExternalThumbOutput("ffmpeg", srcPath, tmp, dstPath, out.String())
}

// vipsThumbArgs は `vips thumbnail` の引数列を組み立てます。
//
//   - 操作は vipsthumbnail ではなく `vips thumbnail`（libvips 8.5 以降）。EXIF Orientation で
//     起こすのが既定で、--crop centre は「埋まるまで縮小してから中央を切り抜く」。
//     vipsthumbnail は版によって回転の既定（--rotate / --no-rotate）が変わった。
//   - 出力の `[Q=<quality>,strip]` は JPEG 品質と、メタデータを落とす指定。
//     回転済みの画素に Orientation タグが残るとブラウザがもう一度回してしまう。
//   - 出力は絶対パスで渡すこと。相対パスだと vips は入力ファイルの隣（= rep の中身）へ書く。
func vipsThumbArgs(srcPath, tmpPath string, w, h int, quality int) []string {
	return []string{
		"thumbnail",
		srcPath,
		fmt.Sprintf("%s[Q=%d,strip]", tmpPath, quality),
		strconv.Itoa(w),
		"--height", strconv.Itoa(h),
		"--crop", "centre",
	}
}

// runVipsThumb は libvips の CLI で静止画のサムネイルを作り、dstPath へ atomic に書きます。
//
// JPEG / HEIC は shrink-on-load（JPEG なら DCT の段階で 1/2・1/4・1/8 に縮めて復号）が
// 効くので、Go の image.Decode で原寸を起こしてから縮小するより数倍速い（12MP の JPEG で 3 倍）。
// PNG / WebP には無く、プロセス起動のぶん Go より遅い（generateThumbJpeg が振り分ける）。
// タイル分割された HEIF も libheif が直接読むので、ffmpeg 経由の原寸 JPEG 書き出しが要らない。
//
// 内部の並列度は VIPS_CONCURRENCY=1 で止める。thumbSem が NumCPU 個のプロセスを同時に
// 走らせるので、各プロセスが NumCPU 本のワーカーを立てると NumCPU² に膨らむ。
func runVipsThumb(ctx context.Context, srcPath, dstPath string, w, h int, quality int) error {
	tmp := thumbTmpPathExt(dstPath, ".jpg")
	_ = os.Remove(tmp)

	cmd := exec.CommandContext(ctx, vipsBin, vipsThumbArgs(srcPath, tmp, w, h, quality)...)
	cmd.Env = append(os.Environ(), "VIPS_CONCURRENCY=1")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("vips thumb failed: %w: %s", err, out.String())
	}

	return finalizeExternalThumbOutput("vips", srcPath, tmp, dstPath, out.String())
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

	err := runFFmpegThumb(ctx, srcPath, dstPath, w, h, sec, false)
	if err == nil || sec <= 0 {
		return err
	}

	// -ss が素材の末尾を越えていると1フレームも取れない。
	// duration を返さない古い .MOV と、動画の名前が付いた静止画がここへ来る。
	// 先頭から取り直せば通るので、1回だけやり直す。
	if retryErr := runFFmpegThumb(ctx, srcPath, dstPath, w, h, 0, false); retryErr != nil {
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

// thumbNativeMaxPixels / thumbNativeMaxPixelsJPEG は、Go で読める形式のうち
// 「プロセスを起こさず Go で作ったほうが速い」画素数の上限（形式別）。
//
// vips は起動だけで数十〜百数十ms かかる（DLL の読み込み。Windows の実測では 12MP の JPEG が
// 145ms で、そのうち復号は 50ms）。Go の復号 + 縮小は画素数に比例する（JPEG で約 40ms/MP、
// WebP で約 45ms/MP）。JPEG は vips が DCT 段階で 1/8 に縮めて復号する（shrink-on-load）ので
// vips 側がほぼ定数で、3MP 前後で並ぶ。PNG / WebP / GIF は vips も原寸復号なので
// プロセス起動ぶん vips が不利で、4MP のスクリーンショットでも Go のほうが 1〜2 割速い。
// gkill の主な入力は自動取得の画面のスクリーンショット（WebP、1〜4MP）とスマホの写真（12MP 級）。
// テストが差し替えられるよう変数にしている。
var (
	thumbNativeMaxPixels     = int64(6_000_000)
	thumbNativeMaxPixelsJPEG = int64(3_000_000)
)

func thumbNativeMaxPixelsFor(format string) int64 {
	if format == "jpeg" {
		return thumbNativeMaxPixelsJPEG
	}
	return thumbNativeMaxPixels
}

// preferNativeThumbDecode は、そのファイルを vips より先に Go で読むべきかを返します。
//
// ヘッダだけ読んで（image.DecodeConfig。画素は復号しない）、Go にデコーダがあり
// 画素数が形式別の上限以下なら true。Go で読めない形式（HEIC / BMP / TIFF …）や
// ヘッダが壊れているものは false で、vips → Go → ffmpeg の順になる。
func preferNativeThumbDecode(srcPath string) bool {
	f, err := os.Open(srcPath)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		return false
	}
	return int64(cfg.Width)*int64(cfg.Height) <= thumbNativeMaxPixelsFor(format)
}

// generateThumbJpeg は静止画のサムネイルを作ります。経路は3段で、前の段が失敗したら次へ落ちる。
//
//  1. vips が PATH にあれば `vips thumbnail`（JPEG / HEIC の shrink-on-load で速い。HEIC も直接読む）
//  2. Go の image.Decode → 中央切り抜き → 縮小 → EXIF 回転 → JPEG（jpeg/png/gif/webp）
//  3. ffmpeg（generateThumbViaFFmpeg）
//
// ただし Go で読めて小さいもの（preferNativeThumbDecode）は 1 を飛ばして 2 から始める。
// vips はプロセス起動が要るので、スクリーンショット級の画像では Go のほうが速い。
//
// vips が無いのは失敗ではない（logThumbBackends が1行残すだけ）。vips が読めなかった
// ファイルは Go → ffmpeg で拾い、3段とも失敗したときだけ呼び出し元が印を焼く。
// そのとき vips のエラー文も包んで返す（印の中身から原因を追えるように）。
func generateThumbJpeg(ctx context.Context, srcPath, dstPath string, w, h int, quality int) error {
	var vipsErr error
	if existVIPS && !preferNativeThumbDecode(srcPath) {
		vipsErr = runVipsThumb(ctx, srcPath, dstPath, w, h, quality)
		if vipsErr == nil {
			return nil
		}
		// ブラウザが待ちきれずに切った ctx で Go 側の復号まで進めない
		if ctx.Err() != nil {
			return vipsErr
		}
		slog.Log(ctx, gkill_log.Debug, "error at generate thumb by vips, falling back to native decode", "file", fmt.Sprintf("%q", filepath.Base(srcPath)), "error", fmt.Sprintf("%q", vipsErr))
	}

	err := generateThumbJpegNative(ctx, srcPath, dstPath, w, h, quality)
	if err != nil && vipsErr != nil {
		return fmt.Errorf("%w (vips: %v)", err, vipsErr)
	}
	return err
}

// generateThumbJpegNative: center-crop → resize → EXIF回転 → jpeg保存（atomic write）
//
// image.Decode で読めなかったものは ffmpeg へ落とす。isImage が通す37拡張子のうち
// Go にデコーダがあるのは jpeg/png/gif/webp の4形式だけで、
// heic/avif/bmp/ico/tiff は必ずそこで失敗する。ffmpeg は拡張子ではなく中身で
// 形式を決めるので、拡張子が実体と食い違っているファイルもここで拾える。
func generateThumbJpegNative(ctx context.Context, srcPath, dstPath string, w, h int, quality int) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close file opened for read", "error", fmt.Sprintf("%q", err))
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

	// 3) JPEG のときだけ Orientation を適用（縮小の後ろで。writeThumbFromImage を参照）
	if format != "jpeg" {
		orient = 1
	}

	return writeThumbFromImage(img, dstPath, w, h, quality, orient)
}

// generateThumbViaFFmpeg は image.Decode で読めなかった画像を ffmpeg で起こします。
// decodeErr は Go 側のデコード失敗で、ffmpeg でも作れなかったときに包んで返します。
func generateThumbViaFFmpeg(ctx context.Context, srcPath, dstPath string, w, h int, quality int, decodeErr error) error {
	// まずは縮小まで ffmpeg の中で済ませる。プロセス1回で終わるので速い
	fastErr := runFFmpegThumb(ctx, srcPath, dstPath, w, h, 0, true)
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
	return writeThumbFromImage(img, dstPath, w, h, quality, 1)
}

// scaleThumbImage は中央切り抜き → 縮小 → EXIF 回転で、w×h のサムネイル画像を作ります。
// orient は EXIF Orientation（1〜8。1 は回転なし）。
//
// 回転は縮小の**後ろ**で掛ける。原寸で回すと 12MP の写真1枚につき
// NRGBA への変換（汎用経路で画素ごとに RGBA64At / SetRGBA64）と回転コピーで 48MB を2回確保し、
// 縦向きのスマホ写真ではそれだけで縮小より時間がかかっていた。縮小後なら 400×400 で無視できる。
// 回転で軸が入れ替わる向き（5〜8）は、切り抜きのアスペクトを源座標で h/w にし、
// (h,w) に縮小してから回すと最終的に (w,h) になる。
//
// 縮小は BiLinear（tent）カーネル。x/image/draw の Kernel.Scale は縮小のとき
// カーネルの支持幅を縮小率ぶん広げる（面積平均に近くなる）ので、7倍縮小でも縞が出ない。
// ApproxBiLinear は近傍4画素しか見ない（支持幅を広げない）ので、大きな縮小では
// エイリアシングが出る。速いからといって置き換えないこと。CatmullRom は同じ品質を
// 保ったまま約2倍の計算量なので、一覧の 400×400 には過剰。
// 合成は Src。dst は新規確保で透明なので Over と結果は同じで、読み戻しの分だけ無駄。
func scaleThumbImage(img image.Image, w, h int, orient int) image.Image {
	swap := 5 <= orient && orient <= 8
	sw, sh := w, h
	if swap {
		sw, sh = h, w
	}

	cropped := cropCenterToAspect(img, float64(sw)/float64(sh))
	dst := image.NewRGBA(image.Rect(0, 0, sw, sh))
	xdraw.BiLinear.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), stdDraw.Src, nil)

	if orient != 1 {
		return applyExifOrientation(dst, orient)
	}
	return dst
}

// writeThumbFromImage は中央切り抜き → 縮小 → EXIF 回転 → JPEG保存（atomic write）を行います。
// orient の意味は scaleThumbImage を参照。
func writeThumbFromImage(img image.Image, dstPath string, w, h int, quality int, orient int) error {
	dst := scaleThumbImage(img, w, h, orient)

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
