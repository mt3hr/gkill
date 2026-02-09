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
	HasH264VAAPI        bool
	HasH264V4L2M2M      bool
	HasH264MediaCodec   bool
	HasH264OMX          bool

	// hwaccels (availability only; actual usability is checked per-file via a quick decode test)
	HasHWAccelAuto         bool
	HasHWAccelCUDA         bool
	HasHWAccelQSV          bool
	HasHWAccelD3D11VA      bool
	HasHWAccelDXVA2        bool
	HasHWAccelVideoToolbox bool

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
	caps.HasH264VAAPI = strings.Contains(encOut, "h264_vaapi")
	caps.HasH264V4L2M2M = strings.Contains(encOut, "h264_v4l2m2m")
	caps.HasH264MediaCodec = strings.Contains(encOut, "h264_mediacodec")
	caps.HasH264OMX = strings.Contains(encOut, "h264_omx")

	hwOut, err := runFFMPEGText(ctx, "ffmpeg", "-hide_banner", "-hwaccels")
	if err == nil {
		caps.HasHWAccelAuto = true // "auto" is an ffmpeg option; treat as present if ffmpeg runs
		h := strings.ToLower(hwOut)
		caps.HasHWAccelCUDA = strings.Contains(h, "\\ncuda") || strings.Contains(h, " cuda")
		caps.HasHWAccelQSV = strings.Contains(h, "\\nqsv") || strings.Contains(h, " qsv")
		caps.HasHWAccelD3D11VA = strings.Contains(h, "\\nd3d11va") || strings.Contains(h, " d3d11va")
		caps.HasHWAccelDXVA2 = strings.Contains(h, "\\ndxva2") || strings.Contains(h, " dxva2")
		caps.HasHWAccelVideoToolbox = strings.Contains(h, "\\nvideotoolbox") || strings.Contains(h, " videotoolbox")
	} else {
		// hwaccels は必須じゃないので失敗しても続行（安全寄り）
		caps.HasHWAccelAuto = true
	}

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

func chooseVAAPIDeviceCandidates() []string {
	// 一般的なVAAPIデバイス（render node優先）
	return []string{"/dev/dri/renderD128", "/dev/dri/card0"}
}

func chooseExistingVAAPIDevice() string {
	for _, dev := range chooseVAAPIDeviceCandidates() {
		if _, err := os.Stat(dev); err == nil {
			return dev
		}
	}
	return ""
}

func encoderWorks(ctx context.Context, encoder string) (bool, error) {
	// VAAPIはデバイス依存なのでキーに含める
	key := encoder
	vaapiDev := ""
	if encoder == "h264_vaapi" {
		vaapiDev = chooseExistingVAAPIDevice()
		if vaapiDev == "" {
			// 使えるデバイスが無いなら即NG（後でlibx264へ）
			encHealthMu.Lock()
			encHealth[encoder+"|<no_vaapi_dev>"] = struct {
				ok  bool
				err error
			}{ok: false, err: fmt.Errorf("vaapi device not found")}
			encHealthMu.Unlock()
			return false, fmt.Errorf("vaapi device not found")
		}
		key = encoder + "|" + vaapiDev
	}

	encHealthMu.Lock()
	if v, ok := encHealth[key]; ok {
		encHealthMu.Unlock()
		return v.ok, v.err
	}
	encHealthMu.Unlock()

	// 1秒のダミー動画を "null" 出力へ。成功すれば採用してよい。
	tctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	var args []string
	switch encoder {
	case "h264_vaapi":
		// VAAPI: hwupload が必要。scaleはテストでは不要（アップロードだけ確認）
		args = []string{
			"-hide_banner",
			"-loglevel", "error",
			"-vaapi_device", vaapiDev,
			"-f", "lavfi",
			"-i", "testsrc2=size=1280x720:rate=30",
			"-t", "1",
			"-vf", "format=nv12,hwupload",
			"-c:v", encoder,
			"-f", "null",
			"-",
		}
	default:
		// それ以外は従来通り
		args = []string{
			"-hide_banner",
			"-loglevel", "error",
			"-f", "lavfi",
			"-i", "testsrc2=size=1280x720:rate=30",
			"-t", "1",
			"-c:v", encoder,
			"-f", "null",
			"-",
		}
	}

	err := runFFMPEG(tctx, args)
	ok := (err == nil)

	encHealthMu.Lock()
	encHealth[key] = struct {
		ok  bool
		err error
	}{ok: ok, err: err}
	encHealthMu.Unlock()

	return ok, err
}

// 優先エンコーダ選択（GPUを使えるなら使う／使えないなら確実なx264へ）
func chooseCandidateEncoders(c ffmpegCaps) []string {
	// OS別の「まず試す」順序。
	// 目的は「可能ならHWエンコード」「ダメなら確実なlibx264」。
	switch runtime.GOOS {
	case "android":
		out := []string{}
		// Android: 端末依存だが、ffmpegビルドによっては h264_mediacodec が使える
		if c.HasH264MediaCodec {
			out = append(out, "h264_mediacodec")
		}
		// 一部環境では v4l2m2m/omx が入っていることもあるが、Androidでは稀なので後ろへ
		if c.HasLibx264 {
			out = append(out, "libx264")
		}
		return out

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

	default: // linux / unix
		out := []string{}
		// Linux: NVENC/QSV/VAAPI が有力候補。環境により存在しない/動かないので実動作で判定する。
		if c.HasH264NVENC {
			out = append(out, "h264_nvenc")
		}
		if c.HasH264QSV {
			out = append(out, "h264_qsv")
		}
		if c.HasH264VAAPI {
			out = append(out, "h264_vaapi")
		}
		// ARM系や一部SoCでは v4l2m2m / omx が有効なことがある（入っていれば試す）
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

// --- hwaccel health cache ---
// ハードウェアデコードは「存在する」だけでは動かないケースがあるため、
// 入力ファイルを使って小さなデコード試験を行い、使えるものをキャッシュする。
var (
	hwHealthMu sync.Mutex
	hwHealth   = map[string]struct {
		ok  bool
		err error
	}{}
)

func hwaccelWorksForFile(ctx context.Context, hwaccel string, srcPath string) (bool, error) {
	// キーは hw + srcPath にする（ファイルによってはHWデコード不可があり得る）
	key := hwaccel + "|" + srcPath
	hwHealthMu.Lock()
	if v, ok := hwHealth[key]; ok {
		hwHealthMu.Unlock()
		return v.ok, v.err
	}
	hwHealthMu.Unlock()

	// 1フレームだけデコードして null へ。
	// 成功すればその hwaccel は少なくともこのファイルに対しては使える。
	tctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-hwaccel", hwaccel,
		"-i", srcPath,
		"-map", "0:v:0",
		"-an",
		"-frames:v", "1",
		"-f", "null",
		"-",
	}

	err := runFFMPEG(tctx, args)
	if err != nil {
		switch hwaccel {
		case "cuda":
			args2 := append([]string{}, args...)
			args2 = insertAfter(args2, []string{"-hwaccel", hwaccel}, []string{"-hwaccel_output_format", "cuda"})
			err = runFFMPEG(tctx, args2)
		case "qsv":
			args2 := append([]string{}, args...)
			args2 = insertAfter(args2, []string{"-hwaccel", hwaccel}, []string{"-hwaccel_output_format", "qsv"})
			err = runFFMPEG(tctx, args2)
		}
	}
	ok := (err == nil)

	hwHealthMu.Lock()
	hwHealth[key] = struct {
		ok  bool
		err error
	}{ok: ok, err: err}
	hwHealthMu.Unlock()

	return ok, err
}

// insertAfter inserts add immediately after the first occurrence of needle (as a contiguous subsequence).
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

// --- hw decode selection ---
// デコードも「GPU優先、ダメなら自動/CPUへフォールバック」したいので、
// 入力ファイルに対して使える hwaccel を短いデコード試験で選ぶ。
func chooseCandidateHWAccels(c ffmpegCaps) []string {
	switch runtime.GOOS {
	case "darwin":
		out := []string{}
		if c.HasHWAccelVideoToolbox {
			out = append(out, "videotoolbox")
		}
		if c.HasHWAccelAuto {
			out = append(out, "auto")
		}
		return out
	case "windows":
		out := []string{}
		if c.HasHWAccelCUDA {
			out = append(out, "cuda")
		}
		if c.HasHWAccelD3D11VA {
			out = append(out, "d3d11va")
		}
		if c.HasHWAccelDXVA2 {
			out = append(out, "dxva2")
		}
		if c.HasHWAccelQSV {
			out = append(out, "qsv")
		}
		if c.HasHWAccelAuto {
			out = append(out, "auto")
		}
		return out
	default:
		out := []string{}
		if c.HasHWAccelCUDA {
			out = append(out, "cuda")
		}
		if c.HasHWAccelQSV {
			out = append(out, "qsv")
		}
		if c.HasHWAccelAuto {
			out = append(out, "auto")
		}
		return out
	}
}

func chooseVerifiedHWAccelForFile(ctx context.Context, c ffmpegCaps, srcPath string) string {
	for _, hw := range chooseCandidateHWAccels(c) {
		if hw == "auto" {
			return hw
		}
		ok, _ := hwaccelWorksForFile(ctx, hw, srcPath)
		if ok {
			return hw
		}
	}
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

	cpuFilter := fmt.Sprintf("scale=-2:min(%d\\,ih)", maxHeight)
	vaapiFilter := fmt.Sprintf("format=nv12,hwupload,scale_vaapi=w=-2:h=%d:force_original_aspect_ratio=decrease", maxHeight)
	enc := chooseVerifiedH264Encoder(ctx, caps)
	hw := chooseVerifiedHWAccelForFile(ctx, caps, srcPath)

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
	}

	// Prefer HW decode when available; if it fails later, transcodeToCompatMP4 will fall back.
	if hw != "" {
		args = append(args, "-hwaccel", hw)
		// Some environments are more stable with an explicit output format.
		switch hw {
		case "cuda":
			args = append(args, "-hwaccel_output_format", "cuda")
		case "qsv":
			args = append(args, "-hwaccel_output_format", "qsv")
		}
	}

	// VAAPI はデバイス指定が必要なことが多いので、必要なら追加する（decode/encode共通）
	if hw == "vaapi" || enc == "h264_vaapi" {
		if dev := chooseExistingVAAPIDevice(); dev != "" {
			args = append(args, "-vaapi_device", dev)
		}
	}

	args = append(args,
		"-i", srcPath,
		"-map", "0:v:0",
		"-map", "0:a?",
	)

	// video
	if enc != "" {
		args = append(args, "-c:v", enc)
	}
	// 互換性（ブラウザ向け）
	// HWエンコードでは -pix_fmt 指定がCPU変換を引き起こすことがあるため、基本はlibx264のときだけ指定する。
	if enc == "libx264" || enc == "" {
		args = append(args, "-pix_fmt", "yuv420p")
	}

	// libx264 のときだけ crf/preset を使う（HW は効かない/挙動が違うことが多い）
	if enc == "libx264" || enc == "" {
		args = append(args,
			"-crf", strconv.Itoa(crf),
			"-preset", preset,
		)
	}

	// filters
	if caps.HasScale {
		if enc == "h264_vaapi" {
			args = append(args, "-vf", vaapiFilter)
		} else {
			args = append(args, "-vf", cpuFilter)
		}
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
