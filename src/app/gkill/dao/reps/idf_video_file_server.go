package reps

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"mime"
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
	if ct := mime.TypeByExtension(strings.ToLower(filepath.Ext(res.servePath))); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	http.ServeFile(w, r, res.servePath)
}

// needsCompatVideoByProbe returns true if we should transcode to a browser-friendly MP4.
// Policy: only H.264 is considered "safe"; anything else -> compat.
// func needsCompatVideoByProbe(ctx context.Context, inputPath string) (bool, error) {
func needsCompatVideoByProbe(_ context.Context, _ string) (bool, error) {
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

/*
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
*/

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

/*
func insertAfter(args []string, needle []string, add []string) []string {
	if len(needle) == 0 || len(add) == 0 {
		return args
	}
	for i := 0; i+len(needle) <= len(args); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if args[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			out := make([]string, 0, len(args)+len(add))
			out = append(out, args[:i+len(needle)]...)
			out = append(out, add...)
			out = append(out, args[i+len(needle):]...)
			return out
		}
	}
	return args
}
*/

func extractArgValue(args []string, key string) string {
	for i := 0; i < len(args)-1; i++ {
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

/*
func chooseVerifiedH264Encoder(ctx context.Context, c ffmpegCaps) string {
	list := listVerifiedH264Encoders(ctx, c)
	if len(list) > 0 {
		return list[0]
	}
	// 最後の手段：エンコーダ名指定なし（ffmpegの既定に任せる）
	return ""
}
*/

// listHWDecodeChoicesForFile returns multiple working HW decode choices in priority order.
// This is used for runtime fallback when a chosen HW decode works for "one frame" but fails
// with the full filter/encode pipeline.
func listHWDecodeChoicesForFile(ctx context.Context, c ffmpegCaps, srcPath string) []hwDecodeChoice {
	choices := make([]hwDecodeChoice, 0, 4)
	addIfWorks := func(hw string, extra []string) {
		if hw == "" {
			return
		}
		if ok, _ := hwaccelWorksForFile(ctx, hw, extra, srcPath); ok {
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
func hwaccelWorksForFile(ctx context.Context, hw string, extraArgs []string, srcPath string) (bool, error) {
	key := hw + "|" + strings.Join(extraArgs, ",") + "|" + srcPath
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

/*
// Choose a hardware decode method for this OS/file.
// Important: this is ONLY about decode. Encode selection is handled separately.
func chooseHWDecodeForFile(ctx context.Context, c ffmpegCaps, srcPath string) hwDecodeChoice {
	switch runtime.GOOS {
	case "windows":
		// NVIDIA: prefer CUDA. If it doesn't work, fall back to "auto" (or no hw).
		if c.HasHWAccelCUDA {
			if ok, _ := hwaccelWorksForFile(ctx, "cuda", []string{"-hwaccel_output_format", "cuda"}, srcPath); ok {
				return hwDecodeChoice{hw: "cuda", extraArgs: []string{"-hwaccel_output_format", "cuda"}}
			}
		}
		if c.HasHWAccelD3D11VA {
			if ok, _ := hwaccelWorksForFile(ctx, "d3d11va", nil, srcPath); ok {
				return hwDecodeChoice{hw: "d3d11va"}
			}
		}
		if c.HasHWAccelDXVA2 {
			if ok, _ := hwaccelWorksForFile(ctx, "dxva2", nil, srcPath); ok {
				return hwDecodeChoice{hw: "dxva2"}
			}
		}
		if c.HasHWAccelQSV {
			if ok, _ := hwaccelWorksForFile(ctx, "qsv", []string{"-hwaccel_output_format", "qsv"}, srcPath); ok {
				return hwDecodeChoice{hw: "qsv", extraArgs: []string{"-hwaccel_output_format", "qsv"}}
			}
		}
		if c.HasHWAccelAuto {
			return hwDecodeChoice{hw: "auto"}
		}
		return hwDecodeChoice{}
	case "android":
		if c.HasHWAccelMediaCodec {
			if ok, _ := hwaccelWorksForFile(ctx, "mediacodec", nil, srcPath); ok {
				return hwDecodeChoice{hw: "mediacodec"}
			}
		}
		if c.HasHWAccelAuto {
			return hwDecodeChoice{hw: "auto"}
		}
		return hwDecodeChoice{}
	case "darwin":
		if c.HasHWAccelVideoToolbox {
			if ok, _ := hwaccelWorksForFile(ctx, "videotoolbox", nil, srcPath); ok {
				return hwDecodeChoice{hw: "videotoolbox"}
			}
		}
		if c.HasHWAccelAuto {
			return hwDecodeChoice{hw: "auto"}
		}
		return hwDecodeChoice{}
	default: // linux, bsd, etc
		if c.HasHWAccelCUDA {
			if ok, _ := hwaccelWorksForFile(ctx, "cuda", []string{"-hwaccel_output_format", "cuda"}, srcPath); ok {
				return hwDecodeChoice{hw: "cuda", extraArgs: []string{"-hwaccel_output_format", "cuda"}}
			}
		}
		if c.HasHWAccelQSV {
			if ok, _ := hwaccelWorksForFile(ctx, "qsv", []string{"-hwaccel_output_format", "qsv"}, srcPath); ok {
				return hwDecodeChoice{hw: "qsv", extraArgs: []string{"-hwaccel_output_format", "qsv"}}
			}
		}
		if c.HasHWAccelVAAPI {
			if dev, okDev := detectVAAPIDevice(); okDev {
				extra := []string{"-vaapi_device", dev}
				if ok, _ := hwaccelWorksForFile(ctx, "vaapi", extra, srcPath); ok {
					return hwDecodeChoice{hw: "vaapi", extraArgs: extra}
				}
			}
		}
		if c.HasHWAccelAuto {
			return hwDecodeChoice{hw: "auto"}
		}
		return hwDecodeChoice{}
	}
}
*/

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
func buildFFMPEGArgsPreferredVariants(ctx context.Context, srcPath, tmpOut string, maxHeight, crf int, preset string, caps ffmpegCaps) [][]string {
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
	decs := listHWDecodeChoicesForFile(ctx, caps, srcPath)
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

func transcodeToCompatMP4(ctx context.Context, srcPath, dstPath string, maxHeight, crf int, preset string) error {
	// NOTE: tmp は ".mp4" じゃないと -f mp4 指定してても失敗する環境があるので mp4 で統一
	tmp := dstPath + ".tmp.mp4"
	_ = os.Remove(tmp)

	ffCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	// ① capabilities（初回だけ）
	caps, _ := getFFMPEGCaps(ffCtx) // err は無視してもOK：検出失敗なら prefer が弱くなるだけ

	// ② args 構築（preferred variants）
	variants := buildFFMPEGArgsPreferredVariants(ffCtx, srcPath, tmp, maxHeight, crf, preset, caps)

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
