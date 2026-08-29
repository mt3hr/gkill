package reps

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"golang.org/x/sync/singleflight"
)

// VideoCacheGenerator はブラウザ互換動画キャッシュの生成と、生成済みかどうかの判定を提供します。
type VideoCacheGenerator interface {
	// GenerateVideoCache は配信URLと同じ形の文字列から互換動画キャッシュを生成します。
	// queryURL は "http://localhost:9999/<相対パス>"（サムネイルと同じ形）を想定します。
	GenerateVideoCache(ctx context.Context, queryURL string) error

	// GenerateVideoCacheFor はURLを介さずに互換動画キャッシュを生成します。
	// rel はリポジトリ内の相対パス、st はその実ファイルの情報です。一括生成はこちらを使ってください。
	GenerateVideoCacheFor(ctx context.Context, rel string, st os.FileInfo) error

	// CompatCacheName はリポジトリ内の相対パスとファイルサイズから、キャッシュのファイル名を返します。
	// CachedCompatNames が返す集合と突き合わせるために使います。
	CompatCacheName(rel string, size int64) string

	// CachedCompatNames は生成済み互換動画のファイル名の集合を、ディレクトリ1回の列挙で返します。
	// キャッシュディレクトリがまだ無いときは空の集合を返します（エラーにしません）。
	CachedCompatNames() (map[string]struct{}, error)
}

var (
	// 同一互換動画の同時生成を1回にまとめる
	videoSF singleflight.Group
	// 生成の同時実行数を制限（CPU/IO暴走防止）
	videoSem = make(chan struct{}, max(1, runtime.NumCPU()/2))
)

// NewVideoFileServer serves files under dir.
// If the target is a video that the browser cannot play as-is, it generates a
// cached MP4 (H.264 + AAC) capped at 720p and serves that instead.
// Whether the original is playable is decided by videoNeedsCompat from an ffprobe
// result; anything not provably playable is transcoded.
// Requires both ffmpeg and ffprobe on PATH; other requests are delegated to base.
func NewVideoFileServer(userID string, dir string, base http.Handler) http.Handler {
	dir = filepath.Clean(os.ExpandEnv(dir))
	cacheDir := derivedCacheDirForUser("video_cache", userID, dir)
	return &IDFVideoFileServer{
		rootDir:   dir,
		cacheDir:  cacheDir,
		base:      base,
		maxHeight: 720,
		crf:       23,
		preset:    "veryfast",
	}
}

// IDFVideoFileServer implements http.Handler and VideoCacheGenerator.
// It is designed to be placed *outside* the thumbnail server, so that thumb requests still work.
type IDFVideoFileServer struct {
	rootDir   string
	cacheDir  string
	base      http.Handler
	maxHeight int
	crf       int
	preset    string
}

type ensuredVideo struct {
	servePath string // original or compat cached path
}

// ensureServePathForURL decides whether we need a compat video, generates it if needed,
// and returns the file path to serve.
// ok=false means "not a video request" and the caller should delegate to base.
func (v *IDFVideoFileServer) ensureServePathForURL(ctx context.Context, u *url.URL) (ensuredVideo, bool, error) {
	// URL Path（StripPrefix後）を安全に相対化
	rel, ok := cleanRelURLPath(u.Path)
	if !ok || rel == "" {
		return ensuredVideo{}, false, nil
	}

	// Only handle video extensions; let base handle everything else.
	if !isVideo(rel) {
		return ensuredVideo{}, false, nil
	}

	abs, ok := SecureJoin(v.rootDir, rel)
	if !ok {
		return ensuredVideo{}, false, nil
	}
	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		return ensuredVideo{}, false, nil
	}

	return v.ensureServePath(ctx, rel, abs, st)
}

// ensureServePath は相対パス・実パス・ファイル情報が揃った状態から配信するパスを決める。
// URL から取り出す部分と分けてあるのは、一括生成が URL を組み立てずに呼べるようにするため。
func (v *IDFVideoFileServer) ensureServePath(ctx context.Context, rel string, abs string, st os.FileInfo) (ensuredVideo, bool, error) {
	// 既存の互換キャッシュがあれば ffmpeg の有無に関わらず配信する。
	// compatPathFor は rel とファイルサイズのみに依存し ffmpeg を使わないため、
	// ffmpeg存在判定より前にキャッシュ有無を確認できる。
	compatPath, cpErr := v.compatPathFor(rel, st)
	if cpErr == nil && fileExists(compatPath) {
		return ensuredVideo{servePath: compatPath}, true, nil
	}

	// 変換に失敗した印があれば、二度と挑まずに原本へ落とす。
	// 印が無いと、失敗する動画へアクセスするたびにハードウェアデコードの疎通確認
	// （最大4種×15秒）と全ffmpegバリアントとフォールバックをやり直すことになる。
	// 消したいときは clear_cache video <利用者ID>（キャッシュディレクトリごと消える）。
	if cpErr == nil && fileExists(compatPath+compatFailedMarkerSuffix) {
		return ensuredVideo{servePath: abs}, true, nil
	}

	// 「変換しなくてよい」と判定済みなら probe をやり直さない。
	//
	// これが無いと、原本のまま配信する動画は**リクエストのたびに ffprobe を起動する**。
	// 動画の再生は Range リクエストで何度もここを通るので、
	// シークするたびに ffprobe のプロセス生成を払うことになる。
	if cpErr == nil && isKnownNoCompatVideo(compatPath) {
		return ensuredVideo{servePath: abs}, true, nil
	}

	// キャッシュが無くffmpeg/ffprobeも無い場合は生成できないので原本へフォールバック。
	if !existFFMPEG || !existFFPROBE {
		return ensuredVideo{servePath: abs}, true, nil
	}

	probe, probeErr := probeVideoStreams(ctx, abs)
	needCompat, videoCodec := true, ""
	if probeErr != nil {
		// probeできなくても変換は試す。変換にも失敗したら原本へ落ちる。
		// ここで原本を返してしまうと、再生できない動画が無音の黒枠になる
		slog.Log(ctx, gkill_log.Debug, "error at probe video", "error", fmt.Sprintf("%q", probeErr))
	} else {
		needCompat = videoNeedsCompat(probe, filepath.Ext(abs))
		videoCodec = probe.primaryVideoCodec()
	}
	if !needCompat {
		if cpErr == nil {
			markKnownNoCompatVideo(compatPath)
		}
		return ensuredVideo{servePath: abs}, true, nil
	}

	if cpErr != nil {
		return ensuredVideo{servePath: abs}, true, nil
	}

	_, genErr, _ := videoSF.Do(compatPath, func() (any, error) {
		videoSem <- struct{}{}
		defer func() { <-videoSem }()

		if fileExists(compatPath) {
			return nil, nil
		}
		if err := os.MkdirAll(filepath.Dir(compatPath), 0o755); err != nil {
			return nil, err
		}
		return nil, transcodeToCompatMP4(ctx, abs, compatPath, v.maxHeight, v.crf, v.preset, videoCodec)
	})
	if genErr != nil {
		// 生成に失敗したら原本へフォールバック。次回以降やり直さないよう印を残す
		markCompatFailed(ctx, compatPath, genErr)
		return ensuredVideo{servePath: abs}, true, nil
	}
	return ensuredVideo{servePath: compatPath}, true, nil
}

// compatFailedMarkerSuffix は互換動画の生成に失敗した印のファイル名につく接尾辞。
const compatFailedMarkerSuffix = ".failed"

var (
	// noCompatVideos は「原本のまま配信してよい」と判定済みの動画の集合。
	// 鍵は互換キャッシュのパスで、相対パス・ファイルサイズ・変換後の高さを含む。
	//
	// ディスクへ印を残さないのは、判定のやり直しが ffprobe 1回で済むため。
	// 起動直後の1リクエストだけ払えばよい。
	noCompatVideosMu sync.RWMutex
	noCompatVideos   = map[string]struct{}{}
)

func isKnownNoCompatVideo(compatPath string) bool {
	noCompatVideosMu.RLock()
	defer noCompatVideosMu.RUnlock()
	_, ok := noCompatVideos[compatPath]
	return ok
}

func markKnownNoCompatVideo(compatPath string) {
	noCompatVideosMu.Lock()
	defer noCompatVideosMu.Unlock()
	noCompatVideos[compatPath] = struct{}{}
}

// markCompatFailed は互換動画の生成に失敗した印を残します。
//
// ctx が切れているときは印を残しません。HTTP経由ではブラウザが待ちきれずに
// 接続を切ると ffmpeg も一緒に落ちるので、それを恒久的な失敗として焼いてはいけません。
func markCompatFailed(ctx context.Context, compatPath string, cause error) {
	if ctx.Err() != nil {
		return
	}
	markerPath := compatPath + compatFailedMarkerSuffix
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		slog.Log(ctx, gkill_log.Debug, "error at make video compat failed marker directory", "error", fmt.Sprintf("%q", err))
		return
	}
	content := fmt.Sprintf("%s\n%v\n", time.Now().Format(time.RFC3339), cause)
	if err := os.WriteFile(markerPath, []byte(content), 0o644); err != nil {
		slog.Log(ctx, gkill_log.Debug, "error at write video compat failed marker", "error", fmt.Sprintf("%q", err))
	}
}

func (v *IDFVideoFileServer) GenerateVideoCache(ctx context.Context, queryURL string) error {
	u, err := url.Parse(queryURL)
	if err != nil {
		return fmt.Errorf("error at parse url %s: %w", queryURL, err)
	}
	res, ok, err := v.ensureServePathForURL(ctx, u)
	if !ok {
		return nil
	}
	// If servePath is original, it means no compat needed or tooling missing.
	_ = res
	return err
}

// GenerateVideoCacheFor はURLを介さずに互換動画キャッシュを生成します。
// 契約は VideoCacheGenerator.GenerateVideoCacheFor を参照。
func (v *IDFVideoFileServer) GenerateVideoCacheFor(ctx context.Context, rel string, st os.FileInfo) error {
	if st == nil || st.IsDir() {
		return nil
	}
	if !isVideo(rel) {
		return nil
	}
	abs, ok := SecureJoin(v.rootDir, rel)
	if !ok {
		return fmt.Errorf("bad path %s", rel)
	}
	_, _, err := v.ensureServePath(ctx, rel, abs, st)
	return err
}

// CachedCompatNames は生成済み互換動画のファイル名の集合を返します。
// 契約は VideoCacheGenerator.CachedCompatNames を参照。
func (v *IDFVideoFileServer) CachedCompatNames() (map[string]struct{}, error) {
	names := map[string]struct{}{}
	entries, err := os.ReadDir(v.cacheDir)
	if err != nil {
		// まだ1件も生成していないだけ
		if os.IsNotExist(err) {
			return names, nil
		}
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names[entry.Name()] = struct{}{}
	}
	return names, nil
}

func (v *IDFVideoFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res, ok, err := v.ensureServePathForURL(r.Context(), r.URL)
	if !ok {
		v.base.ServeHTTP(w, r)
		return
	}
	if err != nil {
		// 何かあれば原本へ
		v.base.ServeHTTP(w, r)
		return
	}

	// Serve compat (or original decided by ensure) with correct content-type.
	// We keep Range support via http.ServeFile.
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(res.servePath))); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	http.ServeFile(w, r, res.servePath)
}

// ffprobeStream は ffprobe -show_streams の1ストリームぶんのうち、互換判定に使う欄。
type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Profile   string `json:"profile"`
	PixFmt    string `json:"pix_fmt"`
}

// ffprobeResult は ffprobe の出力のうち、互換判定に使う部分。
type ffprobeResult struct {
	Streams []ffprobeStream `json:"streams"`
}

// primaryVideoCodec は先頭の映像ストリームのコーデック名を返す。無ければ空文字。
func (r ffprobeResult) primaryVideoCodec() string {
	for _, stream := range r.Streams {
		if strings.EqualFold(stream.CodecType, "video") {
			return strings.ToLower(stream.CodecName)
		}
	}
	return ""
}

// browserPlayableVideoCodecs は「原本のまま配信してよい」コンテナと映像コーデックの対応。
//
// VP9 in MP4 を入れていないのは意図的。Chrome / Firefox は再生できるが
// Safari / iOS は再生できず、クライアントには再生失敗の受け皿が無い。
var browserPlayableVideoCodecs = map[string][]string{
	".mp4":  {"h264"},
	".m4v":  {"h264"},
	".mov":  {"h264"},
	".webm": {"vp8", "vp9"},
}

// browserPlayableAudioCodecs はコンテナごとに原本のまま配信してよい音声コーデック。
var browserPlayableAudioCodecs = map[string][]string{
	".mp4":  {"aac", "mp3"},
	".m4v":  {"aac", "mp3"},
	".mov":  {"aac", "mp3"},
	".webm": {"opus", "vorbis"},
}

// browserPlayablePixFmts は原本のまま配信してよい画素形式。10bit・4:2:2・4:4:4 は弾く。
var browserPlayablePixFmts = []string{"yuv420p", "yuvj420p"}

// browserPlayableH264Profiles は原本のまま配信してよい H.264 プロファイル。
var browserPlayableH264Profiles = []string{"baseline", "constrained baseline", "main", "high"}

// videoNeedsCompat は probe の結果から、互換MP4へ変換する必要があるかを返します。
//
// 「原本のまま配信して確実に再生できる」と言い切れるときだけ false を返します。
// 分からないものは全部 true（変換する）に倒します。クライアントには再生失敗の
// 受け皿が無く、再生できなければエラーも出ずに無音の黒枠になるためです。
//
// 2026-02の初版から一度も有効になったことがない旧実装は、映像コーデック名だけを見て
// H.264なら変換しない、という判定でした。それだけでは
// コンテナが非対応（.mkv 等）・10bit や 4:2:2・音声が AC3/DTS のものが素通りします。
func videoNeedsCompat(probe ffprobeResult, ext string) bool {
	ext = strings.ToLower(ext)
	videoCodecs, playableContainer := browserPlayableVideoCodecs[ext]
	if !playableContainer {
		return true
	}

	videoStreams := []ffprobeStream{}
	audioStreams := []ffprobeStream{}
	for _, stream := range probe.Streams {
		switch strings.ToLower(stream.CodecType) {
		case "video":
			videoStreams = append(videoStreams, stream)
		case "audio":
			audioStreams = append(audioStreams, stream)
		}
	}

	// 映像が1本でないものは <video> がどれを選ぶか定まらないので変換する
	if len(videoStreams) != 1 {
		return true
	}
	video := videoStreams[0]
	if !slices.Contains(videoCodecs, strings.ToLower(video.CodecName)) {
		return true
	}
	if !slices.Contains(browserPlayablePixFmts, strings.ToLower(video.PixFmt)) {
		return true
	}
	if strings.EqualFold(video.CodecName, "h264") && !slices.Contains(browserPlayableH264Profiles, strings.ToLower(video.Profile)) {
		return true
	}

	// 音声は無いか、コンテナに対して素直な形のものだけ許す
	if len(audioStreams) > 1 {
		return true
	}
	if len(audioStreams) == 1 && !slices.Contains(browserPlayableAudioCodecs[ext], strings.ToLower(audioStreams[0].CodecName)) {
		return true
	}
	return false
}

// videoProbeTimeout は1本ぶんの ffprobe に許す時間。
const videoProbeTimeout = 10 * time.Second

// probeVideoStreams は ffprobe で対象の映像・音声ストリームを読み取ります。
//
// 出力は必ず encoding/json で読むこと。ffprobe の JSON はストリームごとに
// codec_name が codec_type より**前**に出るので、`"codec_type":"video"` を探してから
// 後方の codec_name を拾う文字列検索だと、映像を読み飛ばして音声のコーデックを返します。
func probeVideoStreams(ctx context.Context, srcPath string) (ffprobeResult, error) {
	pctx, cancel := context.WithTimeout(ctx, videoProbeTimeout)
	defer cancel()

	cmd := exec.CommandContext(pctx, "ffprobe",
		"-hide_banner",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		srcPath,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return ffprobeResult{}, fmt.Errorf("error at ffprobe: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	result := ffprobeResult{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return ffprobeResult{}, fmt.Errorf("error at parse ffprobe output: %w", err)
	}
	return result, nil
}

// CompatCacheName はキャッシュのファイル名を返します。
// 契約は VideoCacheGenerator.CompatCacheName を参照。
func (v *IDFVideoFileServer) CompatCacheName(rel string, size int64) string {
	hh := sha1.Sum([]byte(rel))
	key := hex.EncodeToString(hh[:])
	// include maxHeight in name to avoid collisions if policy changes
	return fmt.Sprintf("%s_%d_compat_%dp.mp4", key, size, v.maxHeight)
}

func (v *IDFVideoFileServer) compatPathFor(rel string, st os.FileInfo) (string, error) {
	return filepath.Join(v.cacheDir, v.CompatCacheName(rel, st.Size())), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type ffmpegCaps struct {
	// encoders
	HasLibx264          bool
	HasH264NVENC        bool
	HasH264QSV          bool
	HasH264AMF          bool
	HasH264VideoToolbox bool
	HasH264VAAPI        bool
	HasH264V4L2M2M      bool
	HasH264MediaCodec   bool
	HasH264OMX          bool

	// hwaccels (availability only; actual usability is checked per-file)
	HasHWAccelAuto         bool
	HasHWAccelCUDA         bool
	HasHWAccelQSV          bool
	HasHWAccelD3D11VA      bool
	HasHWAccelDXVA2        bool
	HasHWAccelVAAPI        bool
	HasHWAccelVideoToolbox bool
	HasHWAccelMediaCodec   bool

	// filters
	HasScale      bool // CPU scale
	HasScaleCUDA  bool // scale_cuda
	HasScaleVAAPI bool // scale_vaapi
	HasVPPQSV     bool // vpp_qsv (QSV resize)
}

var (
	ffCapsOnce      sync.Once
	ffCapsCached    ffmpegCaps
	ffCapsCachedErr error
)

func getFFMPEGCaps(ctx context.Context) (ffmpegCaps, error) {
	ffCapsOnce.Do(func() {
		// 検出が重いので短めタイムアウト（初回だけ）。
		//
		// 呼び出し元のキャンセルからは切り離す。検出はプロセス生涯で1回しか走らないので、
		// たまたま最初に呼んだリクエストが打ち切られると、以降ずっと検出結果が
		// ゼロ値のままになり、ハードウェアエンコーダが1つも使われなくなる。
		dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		ffCapsCached, ffCapsCachedErr = detectFFMPEGCaps(dctx)
	})
	return ffCapsCached, ffCapsCachedErr
}

func detectFFMPEGCaps(ctx context.Context) (ffmpegCaps, error) {
	var caps ffmpegCaps

	encOut, err := runFFMPEGText(ctx, "ffmpeg", "-hide_banner", "-encoders")
	if err != nil {
		return caps, err
	}
	caps.HasLibx264 = strings.Contains(encOut, "libx264")
	caps.HasH264NVENC = strings.Contains(encOut, "h264_nvenc")
	caps.HasH264QSV = strings.Contains(encOut, "h264_qsv")
	caps.HasH264AMF = strings.Contains(encOut, "h264_amf")
	caps.HasH264VideoToolbox = strings.Contains(encOut, "h264_videotoolbox")
	caps.HasH264VAAPI = strings.Contains(encOut, "h264_vaapi")
	caps.HasH264V4L2M2M = strings.Contains(encOut, "h264_v4l2m2m")
	caps.HasH264MediaCodec = strings.Contains(encOut, "h264_mediacodec")
	caps.HasH264OMX = strings.Contains(encOut, "h264_omx")

	// hwaccels are optional; if this fails we still can transcode (CPU).
	hwOut, err := runFFMPEGText(ctx, "ffmpeg", "-hide_banner", "-hwaccels")
	if err == nil {
		h := strings.ToLower(hwOut)
		// "auto" is not listed, but ffmpeg accepts it. If ffmpeg runs, we treat it as available.
		caps.HasHWAccelAuto = true
		caps.HasHWAccelCUDA = strings.Contains(h, "\ncuda") || strings.Contains(h, " cuda")
		caps.HasHWAccelQSV = strings.Contains(h, "\nqsv") || strings.Contains(h, " qsv")
		caps.HasHWAccelD3D11VA = strings.Contains(h, "\nd3d11va") || strings.Contains(h, " d3d11va")
		caps.HasHWAccelDXVA2 = strings.Contains(h, "\ndxva2") || strings.Contains(h, " dxva2")
		caps.HasHWAccelVAAPI = strings.Contains(h, "\nvaapi") || strings.Contains(h, " vaapi")
		caps.HasHWAccelVideoToolbox = strings.Contains(h, "\nvideotoolbox") || strings.Contains(h, " videotoolbox")
		caps.HasHWAccelMediaCodec = strings.Contains(h, "\nmediacodec") || strings.Contains(h, " mediacodec")
	} else {
		caps.HasHWAccelAuto = true
	}

	fOut, err := runFFMPEGText(ctx, "ffmpeg", "-hide_banner", "-filters")
	if err != nil {
		// filters は必須じゃないので失敗しても続行（安全寄り）
		caps.HasScale = true
		caps.HasScaleCUDA = false
		return caps, nil
	}
	// " scale " のように空白区切りで出る想定（雑でも十分）
	caps.HasScale = strings.Contains(fOut, " scale ")
	caps.HasScaleCUDA = strings.Contains(fOut, " scale_cuda ")
	caps.HasScaleVAAPI = strings.Contains(fOut, " scale_vaapi ")
	caps.HasVPPQSV = strings.Contains(fOut, " vpp_qsv ")

	return caps, nil
}

func runFFMPEGText(ctx context.Context, bin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// ffmpegはstderrに重要情報が出がち
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("%s failed: %w: %s", bin, err, msg)
	}
	// 念のためstderrも含めて返す（環境によりstdout/stderrが揺れるため）
	return stdout.String() + stderr.String(), nil
}

// --- encoder health cache ---
// 「ffmpeg -encoders に出る」だけでは動かないケースがある（特にNVENCはドライバ/API差分で落ちる）ため、
// 小さなテストエンコードで "本当に動く" を判定してキャッシュする。
var (
	encHealthMu sync.RWMutex
	encHealth   = map[string]struct {
		ok  bool
		err error
	}{}
)

func encoderWorks(ctx context.Context, encoder string) (bool, error) {
	encHealthMu.Lock()
	if v, ok := encHealth[encoder]; ok {
		encHealthMu.Unlock()
		return v.ok, v.err
	}
	encHealthMu.Unlock()

	// 1秒のダミー動画を "null" 出力へ。成功すれば採用してよい。
	// 失敗したらそのエンコーダは使わずフォールバックする。
	tctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-f", "lavfi",
		"-i", "testsrc2=size=1280x720:rate=30",
		"-vf", "format=yuv420p",
		"-t", "1",
		"-c:v", encoder,
		"-f", "null",
		"-",
	}
	err := runFFMPEG(tctx, args)
	ok := (err == nil)

	encHealthMu.Lock()
	encHealth[encoder] = struct {
		ok  bool
		err error
	}{ok: ok, err: err}
	encHealthMu.Unlock()

	return ok, err
}

func extractArgValue(args []string, key string) string {
	for i := range len(args) - 1 {
		if args[i] == key {
			return args[i+1]
		}
	}
	return ""
}

// 優先エンコーダ選択（GPUを使えるなら使う／使えないなら確実なx264へ）
func chooseCandidateEncoders(c ffmpegCaps) []string {
	// OS別の「まず試す」順序。
	switch runtime.GOOS {
	case "darwin":
		out := []string{}
		if c.HasH264VideoToolbox {
			out = append(out, "h264_videotoolbox")
		}
		if c.HasLibx264 {
			out = append(out, "libx264")
		}
		return out
	case "windows":
		out := []string{}
		if c.HasH264NVENC {
			out = append(out, "h264_nvenc")
		}
		if c.HasH264QSV {
			out = append(out, "h264_qsv")
		}
		if c.HasH264AMF {
			out = append(out, "h264_amf")
		}
		if c.HasLibx264 {
			out = append(out, "libx264")
		}
		return out
	case "android":
		out := []string{}
		if c.HasH264MediaCodec {
			out = append(out, "h264_mediacodec")
		}
		if c.HasLibx264 {
			out = append(out, "libx264")
		}
		return out
	default: // linux, bsd, etc
		out := []string{}
		if c.HasH264NVENC {
			out = append(out, "h264_nvenc")
		}
		if c.HasH264QSV {
			out = append(out, "h264_qsv")
		}
		if c.HasH264VAAPI {
			out = append(out, "h264_vaapi")
		}
		if c.HasH264V4L2M2M {
			out = append(out, "h264_v4l2m2m")
		}
		if c.HasH264OMX {
			out = append(out, "h264_omx")
		}
		if c.HasLibx264 {
			out = append(out, "libx264")
		}
		return out
	}
}

// listHWDecodeChoicesForFile returns multiple working HW decode choices in priority order.
// This is used for runtime fallback when a chosen HW decode works for "one frame" but fails
// with the full filter/encode pipeline.
func listHWDecodeChoicesForFile(ctx context.Context, c ffmpegCaps, srcPath string, videoCodec string) []hwDecodeChoice {
	choices := make([]hwDecodeChoice, 0, 4)
	addIfWorks := func(hw string, extra []string) {
		if hw == "" {
			return
		}
		if ok, _ := hwaccelWorksForFile(ctx, hw, extra, srcPath, videoCodec); ok {
			choices = append(choices, hwDecodeChoice{hw: hw, extraArgs: extra})
		}
	}

	switch runtime.GOOS {
	case "windows":
		if c.HasHWAccelCUDA {
			addIfWorks("cuda", []string{"-hwaccel_output_format", "cuda"})
		}
		if c.HasHWAccelD3D11VA {
			addIfWorks("d3d11va", nil)
		}
		if c.HasHWAccelDXVA2 {
			addIfWorks("dxva2", nil)
		}
		if c.HasHWAccelQSV {
			addIfWorks("qsv", []string{"-hwaccel_output_format", "qsv"})
		}
	case "android":
		if c.HasHWAccelMediaCodec {
			addIfWorks("mediacodec", nil)
		}
	case "darwin":
		if c.HasHWAccelVideoToolbox {
			addIfWorks("videotoolbox", nil)
		}
	default: // linux, bsd, etc
		if c.HasHWAccelCUDA {
			addIfWorks("cuda", []string{"-hwaccel_output_format", "cuda"})
		}
		if c.HasHWAccelQSV {
			addIfWorks("qsv", []string{"-hwaccel_output_format", "qsv"})
		}
		if c.HasHWAccelVAAPI {
			if dev, okDev := detectVAAPIDevice(); okDev {
				addIfWorks("vaapi", []string{"-vaapi_device", dev})
			}
		}
	}

	// "auto" is accepted by ffmpeg and can still use HW paths internally, but it's harder to reason about.
	// We keep it as a last resort choice so we can try it when explicit hwaccels fail.
	if c.HasHWAccelAuto {
		choices = append(choices, hwDecodeChoice{hw: "auto"})
	}
	return choices
}

// --- hw decode selection ---
// デコードも「GPU優先、ダメなら自動/CPUへフォールバック」したいので、
// 入力ファイルに対して使える hwaccel を短いデコード試験で選ぶ。

// --- hw decode health cache ---
// 「-hwaccel が使える」だけでは、入力ファイルや環境（ドライバ/デバイス）によって失敗することがあるため、
// 実ファイルで "1フレームだけデコード" を試して使えるものをキャッシュする。
var (
	hwHealthMu sync.RWMutex
	hwHealth   = map[string]struct {
		ok  bool
		err error
	}{}
)

// key is "hw|extra|srcPath" so we can cache per file + variant.
// hwaccelWorksForFile は指定のハードウェアデコードで対象を1フレームだけデコードしてみます。
//
// 結果のキャッシュは映像コーデック単位にする。判定の実体は
// 「このハードウェアデコーダがこの種類のストリームを扱えるか」であってファイル固有ではないので、
// パスでキーにすると、動画1本ごとに最大4種×15秒の疎通確認が走るうえ、
// 一括生成では表が動画の本数ぶん際限なく伸びる。
// コーデックが分からないとき（probe失敗）だけは、従来どおりパスでキーにする。
func hwaccelWorksForFile(ctx context.Context, hw string, extraArgs []string, srcPath string, videoCodec string) (bool, error) {
	cacheTarget := videoCodec
	if cacheTarget == "" {
		cacheTarget = srcPath
	}
	key := hw + "|" + strings.Join(extraArgs, ",") + "|" + cacheTarget
	hwHealthMu.Lock()
	if v, ok := hwHealth[key]; ok {
		hwHealthMu.Unlock()
		return v.ok, v.err
	}
	hwHealthMu.Unlock()

	tctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-hwaccel", hw,
	}
	args = append(args, extraArgs...)
	args = append(args,
		"-i", srcPath,
		"-map", "0:v:0",
		"-an",
		"-frames:v", "1",
		"-f", "null",
		"-",
	)

	err := runFFMPEG(tctx, args)
	ok := (err == nil)

	hwHealthMu.Lock()
	hwHealth[key] = struct {
		ok  bool
		err error
	}{ok: ok, err: err}
	hwHealthMu.Unlock()

	return ok, err
}

// linux VAAPI needs a device node in most setups
func detectVAAPIDevice() (string, bool) {
	candidates := []string{"/dev/dri/renderD128", "/dev/dri/card0"}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}

type hwDecodeChoice struct {
	hw        string
	extraArgs []string
}

func isGPUEncoder(enc string) bool {
	switch enc {
	case "h264_nvenc", "h264_qsv", "h264_amf", "h264_videotoolbox", "h264_vaapi", "h264_v4l2m2m", "h264_mediacodec", "h264_omx":
		return true
	default:
		return false
	}
}

func cloneArgs(a []string) []string {
	out := make([]string, len(a))
	copy(out, a)
	return out
}

// listVerifiedH264Encoders returns encoders in priority order that pass encoderWorks().
// This improves robustness: some encoders may pass a synthetic test but still fail on
// certain real inputs, and the reverse also happens. We therefore try multiple candidates.
func listVerifiedH264Encoders(ctx context.Context, c ffmpegCaps) []string {
	out := make([]string, 0, 4)
	for _, enc := range chooseCandidateEncoders(c) {
		ok, _ := encoderWorks(ctx, enc)
		if ok {
			out = append(out, enc)
		}
	}
	// Always ensure libx264 is present as the last resort when available.
	if c.HasLibx264 {
		found := false
		for _, e := range out {
			if e == "libx264" {
				found = true
				break
			}
		}
		if !found {
			out = append(out, "libx264")
		}
	}
	return out
}

// chooseOutputPixFmt decides whether to force an output pixel format.
//   - libx264: force yuv420p for maximum browser compatibility.
//   - GPU encoders: prefer nv12 when frames are in system memory, but avoid forcing when
//     staying on GPU frames (forcing can trigger implicit downloads).
func chooseOutputPixFmt(enc string, usingGPUFrames bool) (pixFmt string, ok bool) {
	if usingGPUFrames {
		return "", false
	}
	if enc == "libx264" || enc == "" {
		return "yuv420p", true
	}
	if isGPUEncoder(enc) {
		// nv12 is broadly supported and usually the preferred input for hardware encoders.
		return "nv12", true
	}
	return "", false
}

func defaultBitrateForHeight(maxHeight int) (b, max, buf string) {
	// Conservative-ish defaults; tuned for 720p targets.
	// We keep them strings to avoid unnecessary strconv calls at call sites.
	if maxHeight <= 480 {
		return "2M", "3M", "4M"
	}
	if maxHeight <= 720 {
		return "5M", "7M", "10M"
	}
	if maxHeight <= 1080 {
		return "8M", "12M", "16M"
	}
	return "12M", "18M", "24M"
}

// buildFFMPEGArgsPreferredVariants returns multiple candidate argument sets.
//
// We try them in order until one succeeds, to achieve **independent fallback**:
// - Prefer GPU encode when available.
// - Prefer GPU decode when available.
// - If GPU decode doesn't work with the chosen filter chain, fall back to CPU decode *without* downgrading encode.
// - If GPU encode doesn't work, fall back to CPU encode.
//
// NOTE:
// We keep a conservative "all-GPU" path (GPU frames throughout) when it is known to be compatible.
// Otherwise, we still may try HW decode but download to system memory before CPU filters/encode.
func buildFFMPEGArgsPreferredVariants(ctx context.Context, srcPath, tmpOut string, maxHeight, crf int, preset string, caps ffmpegCaps, videoCodec string) [][]string {
	if maxHeight <= 0 {
		maxHeight = 720
	}
	if crf <= 0 {
		crf = 23
	}
	if preset == "" {
		preset = "veryfast"
	}

	// Encode / Decode are decided independently, but we keep multiple fallbacks.
	encoders := listVerifiedH264Encoders(ctx, caps)
	if len(encoders) == 0 {
		encoders = []string{""}
	}
	decs := listHWDecodeChoicesForFile(ctx, caps, srcPath, videoCodec)
	if len(decs) == 0 {
		decs = []hwDecodeChoice{{}}
	}

	// Default: CPU scale (safe everywhere)
	cpuScale := fmt.Sprintf("scale=-2:min(%d\\,ih)", maxHeight)

	// Build base args (shared prefix)
	base := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
		"-vsync", "vfr",
	}

	buildOne := func(encName string, enableHW bool, hw hwDecodeChoice, vf string, usingGPUFrames bool) []string {
		args := cloneArgs(base)
		if enableHW && hw.hw != "" {
			args = append(args, "-hwaccel", hw.hw)
			args = append(args, hw.extraArgs...)
		}
		args = append(args,
			"-i", srcPath,
			"-map", "0:v:0",
			"-map", "0:a?",
		)
		if vf != "" {
			args = append(args, "-vf", vf)
		}
		if encName != "" {
			args = append(args, "-c:v", encName)
		}
		if pix, ok := chooseOutputPixFmt(encName, usingGPUFrames); ok {
			args = append(args, "-pix_fmt", pix)
		}
		// Some HW encoders benefit from explicit bitrate control (not CRF-based).
		if encName == "h264_videotoolbox" || encName == "h264_mediacodec" {
			b, mx, buf := defaultBitrateForHeight(maxHeight)
			args = append(args, "-b:v", b, "-maxrate", mx, "-bufsize", buf)
		}
		if encName == "libx264" || encName == "" {
			args = append(args, "-crf", strconv.Itoa(crf), "-preset", preset)
		}
		args = append(args,
			"-c:a", "aac",
			"-b:a", "128k",
			"-movflags", "+faststart",
			"-f", "mp4",
			tmpOut,
		)
		return args
	}

	// Prepare CPU scale filter (used by several variants)
	cpuVF := ""
	if caps.HasScale {
		cpuVF = cpuScale
	}

	// Variant list in priority order. We build OS-tuned sequences that emphasize
	// "GPU encode if possible" and make HW decode optional and independently fall back.
	variants := make([][]string, 0, 12)

	// Helper to append a set of decode variants for a given encoder.
	appendVariantsForEnc := func(enc string) {
		// OS-specific preference: on android/darwin, GPU decode is often less stable; favor encode-only first.
		preferEncodeOnlyFirst := (runtime.GOOS == "android" || runtime.GOOS == "darwin")

		// 1) CPU decode + (possibly GPU) encode (most robust while still benefiting from GPU encode)
		if preferEncodeOnlyFirst {
			variants = append(variants, buildOne(enc, false, hwDecodeChoice{}, cpuVF, false))
		}

		// 2) All-GPU pipeline where we know the full chain is compatible.
		for _, dec := range decs {
			useGPUFrames := false
			gpuVF := ""
			switch runtime.GOOS {
			case "windows":
				// CUDA pipeline: NVDEC(CUDA) -> scale_cuda -> NVENC
				if dec.hw == "cuda" && enc == "h264_nvenc" && caps.HasScaleCUDA {
					useGPUFrames = true
					gpuVF = fmt.Sprintf("scale_cuda=-2:%d:format=nv12", maxHeight)
				}
				// QSV pipeline: QSV decode -> vpp_qsv -> QSV encode
				if !useGPUFrames && dec.hw == "qsv" && enc == "h264_qsv" && caps.HasVPPQSV {
					useGPUFrames = true
					gpuVF = fmt.Sprintf("vpp_qsv=w=-2:h=%d", maxHeight)
				}
			case "darwin":
				// Keep all-GPU conservative; VideoToolbox decode+filter chains vary widely.
			case "android":
				// Keep all-GPU conservative; mediacodec decode+filters vary by device.
			default: // linux/bsd/etc
				// VAAPI all-GPU: vaapi decode -> scale_vaapi -> vaapi encode
				if dec.hw == "vaapi" && enc == "h264_vaapi" && caps.HasScaleVAAPI {
					useGPUFrames = true
					gpuVF = fmt.Sprintf("scale_vaapi=w=-2:h=%d", maxHeight)
				}
				// QSV all-GPU
				if !useGPUFrames && dec.hw == "qsv" && enc == "h264_qsv" && caps.HasVPPQSV {
					useGPUFrames = true
					gpuVF = fmt.Sprintf("vpp_qsv=w=-2:h=%d", maxHeight)
				}
				// CUDA all-GPU
				if !useGPUFrames && dec.hw == "cuda" && enc == "h264_nvenc" && caps.HasScaleCUDA {
					useGPUFrames = true
					gpuVF = fmt.Sprintf("scale_cuda=-2:%d:format=nv12", maxHeight)
				}
			}

			if useGPUFrames && dec.hw != "" && dec.hw != "auto" {
				variants = append(variants, buildOne(enc, true, dec, gpuVF, true))
			}
		}

		// 3) HW decode only (download to system memory) + keep encoder.
		//    We may try both "hwdownload" and "no hwdownload" flavors for unstable platforms.
		for _, dec := range decs {
			if dec.hw == "" {
				continue
			}
			// Skip explicit "auto" here; it is less predictable and will be tried last.
			if dec.hw == "auto" {
				continue
			}
			vf := cpuVF
			switch dec.hw {
			case "cuda", "qsv", "vaapi":
				// HW frames -> download to system mem before CPU scale.
				if vf != "" {
					vf = "hwdownload,format=nv12," + vf
				} else {
					vf = "hwdownload,format=nv12"
				}
			case "videotoolbox", "mediacodec":
				// Try without hwdownload first (some builds auto-convert), then with hwdownload.
				variants = append(variants, buildOne(enc, true, dec, cpuVF, false))
				if vf != "" {
					vf = "hwdownload,format=nv12," + vf
				} else {
					vf = "hwdownload,format=nv12"
				}
			}
			variants = append(variants, buildOne(enc, true, dec, vf, false))
		}

		// 4) CPU decode + keep encoder (if we didn't already prefer it first)
		if !preferEncodeOnlyFirst {
			variants = append(variants, buildOne(enc, false, hwDecodeChoice{}, cpuVF, false))
		}

		// 5) Encoder-specific CPU decode + GPU upload filter chains for VAAPI/QSV encoders.
		//    These can be faster than CPU encode even without HW decode.
		if cpuVF != "" {
			switch enc {
			case "h264_vaapi":
				if caps.HasScaleVAAPI {
					vf := fmt.Sprintf("format=nv12,hwupload,scale_vaapi=w=-2:h=%d", maxHeight)
					variants = append(variants, buildOne(enc, false, hwDecodeChoice{}, vf, true))
				}
			case "h264_qsv":
				if caps.HasVPPQSV {
					vf := fmt.Sprintf("format=nv12,hwupload=extra_hw_frames=16,vpp_qsv=w=-2:h=%d", maxHeight)
					variants = append(variants, buildOne(enc, false, hwDecodeChoice{}, vf, true))
				}
			}
		}

		// 6) As a last resort, try explicit hwaccel auto with the same encoder.
		for _, dec := range decs {
			if dec.hw == "auto" {
				variants = append(variants, buildOne(enc, true, dec, cpuVF, false))
				break
			}
		}
	}

	for _, enc := range encoders {
		appendVariantsForEnc(enc)
	}

	// Ensure we always have a deterministic libx264 CPU variant at the very end.
	if caps.HasLibx264 {
		variants = append(variants, buildOne("libx264", false, hwDecodeChoice{}, cpuVF, false))
	}
	return variants
}

// 「堅牢デフォルト」args（失敗時フォールバック用：libx264固定）
func buildFFMPEGArgsFallback(srcPath, tmpOut string, maxHeight, crf int, preset string) []string {
	if maxHeight <= 0 {
		maxHeight = 720
	}
	if crf <= 0 {
		crf = 23
	}
	if preset == "" {
		preset = "veryfast"
	}
	cpuFilter := fmt.Sprintf("scale=-2:min(%d\\,ih)", maxHeight)

	return []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-fflags", "+genpts",
		"-avoid_negative_ts", "make_zero",
		"-vsync", "vfr",
		"-i", srcPath,
		"-map", "0:v:0",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-crf", strconv.Itoa(crf),
		"-preset", preset,
		"-vf", cpuFilter,
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-f", "mp4",
		tmpOut,
	}
}

func transcodeToCompatMP4(ctx context.Context, srcPath, dstPath string, maxHeight, crf int, preset string, videoCodec string) error {
	// NOTE: tmp は ".mp4" じゃないと -f mp4 指定してても失敗する環境があるので mp4 で統一
	tmp := dstPath + ".tmp.mp4"
	_ = os.Remove(tmp)

	ffCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	// ① capabilities（初回だけ）
	caps, _ := getFFMPEGCaps(ffCtx) // err は無視してもOK：検出失敗なら prefer が弱くなるだけ

	// ② args 構築（preferred variants）
	variants := buildFFMPEGArgsPreferredVariants(ffCtx, srcPath, tmp, maxHeight, crf, preset, caps, videoCodec)

	// ③ 実行：上から順に試す。失敗しても次へ（独立フォールバック）。
	//    Also apply lightweight error-based skipping so we don't repeatedly try impossible paths.
	skipEnc := map[string]bool{}
	skipHW := map[string]bool{}
	var lastErr error
	for i, a := range variants {
		enc := extractArgValue(a, "-c:v")
		if enc != "" && skipEnc[enc] {
			continue
		}
		hw := extractArgValue(a, "-hwaccel")
		if hw != "" && skipHW[hw] {
			continue
		}

		err := runFFMPEG(ffCtx, a)
		if err == nil {
			lastErr = nil
			break
		}
		lastErr = fmt.Errorf("variant[%d] failed: %w", i, err)

		// Error-based skip heuristics
		emsg := strings.ToLower(err.Error())
		if strings.Contains(emsg, "unknown encoder") || strings.Contains(emsg, "encoder '"+strings.ToLower(enc)+"' not found") || strings.Contains(emsg, "invalid encoder") {
			if enc != "" {
				skipEnc[enc] = true
			}
		}
		if strings.Contains(emsg, "unknown hwaccel") || strings.Contains(emsg, "hardware accelerator not found") || strings.Contains(emsg, "no device") || strings.Contains(emsg, "device creation failed") || strings.Contains(emsg, "failed to open") || strings.Contains(emsg, "vaapi") && strings.Contains(emsg, "device") {
			if hw != "" {
				skipHW[hw] = true
			}
		}
		// If the error is clearly filter-chain related, we only skip HW for this variant and let CPU decode proceed.
		if strings.Contains(emsg, "hwdownload") || strings.Contains(emsg, "hwupload") || strings.Contains(emsg, "scale_cuda") || strings.Contains(emsg, "scale_vaapi") || strings.Contains(emsg, "vpp_qsv") {
			if hw != "" {
				skipHW[hw] = true
			}
		}
	}
	if lastErr != nil {
		// fallback（堅牢デフォルト）
		fb := buildFFMPEGArgsFallback(srcPath, tmp, maxHeight, crf, preset)
		if err2 := runFFMPEG(ffCtx, fb); err2 != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("ffmpeg failed (preferred variants): %v; (fallback): %w", lastErr, err2)
		}
	}

	_ = os.Remove(dstPath)
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// runFFMPEG executes ffmpeg and captures stdout/stderr separately.
// FFmpeg typically logs important diagnostics to stderr.
func runFFMPEG(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return fmt.Errorf("%w: %s\nargs: %s", err, msg, strings.Join(args, " "))
	}
	return nil
}
