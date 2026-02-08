package reps

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"golang.org/x/sync/singleflight"
)

// VideoCacheGenerator is used by CLI batch generation.
// The queryURL is expected to be like "http://localhost:9999/<relpath>" (same style as thumbnails).
type VideoCacheGenerator interface {
	GenerateVideoCache(ctx context.Context, queryURL string) error
}

var (
	// 同一互換動画の同時生成を1回にまとめる
	videoSF singleflight.Group
	// 生成の同時実行数を制限（CPU/IO暴走防止）
	videoSem = make(chan struct{}, max(1, runtime.NumCPU()/2))
)

// NewVideoFileServer serves files under dir.
// If the target is a video and the codec is not browser-friendly (e.g. HEVC),
// it generates a cached MP4 (H.264 + AAC) capped at 720p and serves it.
// Other requests are delegated to base.
func NewVideoFileServer(dir string, base http.Handler) http.Handler {
	dir = filepath.Clean(os.ExpandEnv(dir))
	cacheDir := os.ExpandEnv(filepath.Join(gkill_options.CacheDir, "video_cache", filepath.Base(dir)))
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

	abs, ok := secureJoin(v.rootDir, rel)
	if !ok {
		return ensuredVideo{}, false, nil
	}
	st, err := os.Stat(abs)
	if err != nil || st.IsDir() {
		return ensuredVideo{}, false, nil
	}

	// If ffmpeg/ffprobe are missing, fallback to original.
	if !existFFMPEG || !existFFPROBE {
		return ensuredVideo{servePath: abs}, true, nil
	}

	needCompat, err := needsCompatVideoByProbe(ctx, abs)
	if err != nil {
		// probe失敗時は原本で運用継続（安全）
		return ensuredVideo{servePath: abs}, true, nil
	}
	if !needCompat {
		return ensuredVideo{servePath: abs}, true, nil
	}

	compatPath, err := v.compatPathFor(rel, st)
	if err != nil {
		return ensuredVideo{servePath: abs}, true, nil
	}
	if fileExists(compatPath) {
		return ensuredVideo{servePath: compatPath}, true, nil
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
		return nil, transcodeToCompatMP4(ctx, abs, compatPath, v.maxHeight, v.crf, v.preset)
	})
	if genErr != nil {
		// 生成に失敗したら原本へフォールバック
		return ensuredVideo{servePath: abs}, true, nil
	}
	return ensuredVideo{servePath: compatPath}, true, nil
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
	w.Header().Set("Content-Type", "video/mp4")
	http.ServeFile(w, r, res.servePath)
}

// needsCompatVideoByProbe returns true if we should transcode to a browser-friendly MP4.
// Policy: only H.264 is considered "safe"; anything else -> compat.
func needsCompatVideoByProbe(ctx context.Context, inputPath string) (bool, error) {
	return true, nil
	/*
		probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		codec, err := ffprobeVideoCodec(probeCtx, inputPath)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(codec) {
		case "h264", "avc1":
			return false, nil
		default:
			return true, nil
		}
	*/
}

// ffprobeVideoCodec extracts the first video codec_name.
func ffprobeVideoCodec(ctx context.Context, inputPath string) (string, error) {
	// ffprobe -v error -print_format json -show_streams <input>
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_streams",
		inputPath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffprobe failed: %w: %s", err, out.String())
	}
	// Minimal parse without new dependencies: look for \"codec_type\":\"video\" then \"codec_name\":\"...\".
	// This is robust enough for ffprobe output and avoids pulling in a struct here.
	// (thumb server already parses duration via json; here we keep it lean.)
	s := out.String()
	idx := strings.Index(s, "\"codec_type\":\"video\"")
	if idx < 0 {
		return "", fmt.Errorf("ffprobe: no video stream")
	}
	// find codec_name after that
	seg := s[idx:]
	cn := "\"codec_name\":\""
	j := strings.Index(seg, cn)
	if j < 0 {
		return "", fmt.Errorf("ffprobe: codec_name missing")
	}
	seg2 := seg[j+len(cn):]
	k := strings.Index(seg2, "\"")
	if k < 0 {
		return "", fmt.Errorf("ffprobe: codec_name parse failed")
	}
	return seg2[:k], nil
}

func (v *IDFVideoFileServer) compatPathFor(rel string, st os.FileInfo) (string, error) {
	hh := sha1.Sum([]byte(rel))
	key := hex.EncodeToString(hh[:])
	ver := fmt.Sprintf("%d_%d", st.ModTime().Unix(), st.Size())
	// include maxHeight in name to avoid collisions if policy changes
	name := fmt.Sprintf("%s_%s_compat_%dp.mp4", key, ver, v.maxHeight)
	return filepath.Join(v.cacheDir, name), nil
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

	// filters
	HasScale bool
}

var (
	ffCapsOnce      sync.Once
	ffCapsCached    ffmpegCaps
	ffCapsCachedErr error
)

func getFFMPEGCaps(ctx context.Context) (ffmpegCaps, error) {
	ffCapsOnce.Do(func() {
		// 検出が重いので短めタイムアウト（初回だけ）
		dctx, cancel := context.WithTimeout(ctx, 5*time.Second)
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

	fOut, err := runFFMPEGText(ctx, "ffmpeg", "-hide_banner", "-filters")
	if err != nil {
		// filters は必須じゃないので失敗しても続行（安全寄り）
		caps.HasScale = true
		return caps, nil
	}
	// " scale " のように空白区切りで出る想定（雑でも十分）
	caps.HasScale = strings.Contains(fOut, " scale ")

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
	encHealthMu sync.Mutex
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

// 優先エンコーダ選択（GPUを使えるなら使う／使えないなら確実なx264へ）
func chooseCandidateEncoders(c ffmpegCaps) []string {
	// OS別の「まず試す」順序。
	// Windows + RTX などは NVENC を最優先にするのが体感に合う。
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
	default: // linux etc
		out := []string{}
		if c.HasH264NVENC {
			out = append(out, "h264_nvenc")
		}
		if c.HasH264QSV {
			out = append(out, "h264_qsv")
		}
		if c.HasLibx264 {
			out = append(out, "libx264")
		}
		return out
	}
}

func chooseVerifiedH264Encoder(ctx context.Context, c ffmpegCaps) string {
	for _, enc := range chooseCandidateEncoders(c) {
		// libx264 は「動く」前提でよいが、一応判定関数を通して統一
		ok, _ := encoderWorks(ctx, enc)
		if ok {
			return enc
		}
	}
	// 最後の手段：エンコーダ名指定なし（ffmpegの既定に任せる）
	return ""
}

// 「最適」args（環境に合わせて組む）
func buildFFMPEGArgsPreferred(ctx context.Context, srcPath, tmpOut string, maxHeight, crf int, preset string, caps ffmpegCaps) []string {
	if maxHeight <= 0 {
		maxHeight = 720
	}
	if crf <= 0 {
		crf = 23
	}
	if preset == "" {
		preset = "veryfast"
	}

	filter := fmt.Sprintf("scale=-2:min(%d\\,ih)", maxHeight)
	enc := chooseVerifiedH264Encoder(ctx, caps)

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i", srcPath,
		"-map", "0:v:0",
		"-map", "0:a?",
	}

	// video
	if enc != "" {
		args = append(args, "-c:v", enc)
	}
	// 互換性（ブラウザ向け）
	args = append(args, "-pix_fmt", "yuv420p")

	// libx264 のときだけ crf/preset を使う（HW は効かない/挙動が違うことが多い）
	if enc == "libx264" || enc == "" {
		args = append(args,
			"-crf", strconv.Itoa(crf),
			"-preset", preset,
		)
	}

	// filters
	if caps.HasScale {
		args = append(args, "-vf", filter)
	} else {
		// scale が無いのは稀だが、無いなら無理に付けない（フォールバック側で再挑戦）
	}

	// audio（AACがなければフォールバックで落ちるので、ここは固定でもOK）
	args = append(args,
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-f", "mp4",
		tmpOut,
	)
	return args
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
	filter := fmt.Sprintf("scale=-2:min(%d\\,ih)", maxHeight)

	return []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i", srcPath,
		"-map", "0:v:0",
		"-map", "0:a?",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		"-crf", strconv.Itoa(crf),
		"-preset", preset,
		"-vf", filter,
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-f", "mp4",
		tmpOut,
	}
}

func transcodeToCompatMP4(ctx context.Context, srcPath, dstPath string, maxHeight, crf int, preset string) error {
	// NOTE: tmp は ".mp4" じゃないと -f mp4 指定してても失敗する環境があるので mp4 で統一
	tmp := dstPath + ".tmp.mp4"
	_ = os.Remove(tmp)

	ffCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	// ① capabilities（初回だけ）
	caps, _ := getFFMPEGCaps(ffCtx) // err は無視してもOK：検出失敗なら prefer が弱くなるだけ

	// ② args 構築（preferred）
	args := buildFFMPEGArgsPreferred(ffCtx, srcPath, tmp, maxHeight, crf, preset, caps)

	// ③ 実行 → 失敗なら fallback
	if err := runFFMPEG(ffCtx, args); err != nil {
		// fallback
		fb := buildFFMPEGArgsFallback(srcPath, tmp, maxHeight, crf, preset)
		if err2 := runFFMPEG(ffCtx, fb); err2 != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("ffmpeg failed (preferred): %v; (fallback): %w", err, err2)
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
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}

// Optional helper: run ffmpeg and stream stdout somewhere (e.g. progress), while still capturing stderr.
// Not used by default, but handy if you later want progress logging.
func runFFMPEGWithStdout(ctx context.Context, args []string, stdoutW io.Writer) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	if stdoutW != nil {
		cmd.Stdout = stdoutW
	} else {
		cmd.Stdout = io.Discard
	}
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}
