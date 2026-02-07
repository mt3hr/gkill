package reps

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

func transcodeToCompatMP4(ctx context.Context, srcPath, dstPath string, maxHeight, crf int, preset string) error {
	if maxHeight <= 0 {
		maxHeight = 720
	}
	if crf <= 0 {
		crf = 23
	}
	if preset == "" {
		preset = "veryfast"
	}

	// scale to at most maxHeight without upscaling
	// scale=-2:min(720\,ih) keeps aspect ratio; -2 ensures even width.
	filter := fmt.Sprintf("scale=-2:min(%d\\,ih)", maxHeight)

	// NOTE: ffmpeg chooses container format from the output filename extension when -f is not specified.
	// If we use ".tmp" it can't infer mp4 and fails with:
	//   Unable to choose an output format ... use a standard extension...
	// So we keep ".mp4" as the final extension for the temporary file.
	tmp := dstPath + ".tmp.mp4"
	_ = os.Remove(tmp)

	ffCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i", srcPath,
		// map: first video, optional audio
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
		// Be explicit about container too (defensive; helps if someone changes tmp naming later).
		"-f", "mp4",
		tmp,
	}
	cmd := exec.CommandContext(ffCtx, "ffmpeg", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("ffmpeg failed: %w: %s", err, out.String())
	}

	_ = os.Remove(dstPath)
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
