package reps

// 外部ツール（ffmpeg / vips）に渡す引数列の回帰テスト。実行ファイルは要らない。
//
// 固定したいのは次の3つ。どれも破っても目の前ではエラーにならない。
//   - ffmpeg は -nostdin を付けて非対話で起こす。
//   - 静止画の ffmpeg は復号・フィルタ・符号化とも1スレッド。thumbSem が NumCPU 個の
//     プロセスを同時に走らせるので、中で NumCPU 本ずつ立てると NumCPU² に膨らむ。
//     動画は -ss の先読みで数十〜数百フレームを復号するので復号スレッドは既定のまま。
//   - vips の出力は絶対パス・.jpg 終端・[Q=<品質>,strip] 付き。相対パスだと入力の隣へ書き、
//     .tmp 終端だとセーバが選べず、strip が無いと回転済みの画素に Orientation が残る。

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// indexOfArgPair は "-flag value" の並びが args の中にある位置を返す（無ければ -1）。
func indexOfArgPair(args []string, flag, value string) int {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return i
		}
	}
	return -1
}

func countArgPair(args []string, flag, value string) int {
	n := 0
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			n++
		}
	}
	return n
}

func TestFFmpegThumbArgsStillImageIsSingleThreaded(t *testing.T) {
	args := ffmpegThumbArgs("in.heic", "out.tmp", 400, 400, 0, true)

	if !slices.Contains(args, "-nostdin") {
		t.Errorf("-nostdin が無い: %v", args)
	}
	if slices.Contains(args, "-ss") {
		t.Errorf("静止画に -ss が付いている: %v", args)
	}
	input := slices.Index(args, "-i")
	if input < 0 {
		t.Fatalf("-i が無い: %v", args)
	}
	// 復号側（-i より前）と符号化側（-i より後）の両方に -threads 1
	if n := countArgPair(args, "-threads", "1"); n != 2 {
		t.Errorf("-threads 1 が復号側と符号化側の2箇所に無い（%d箇所）: %v", n, args)
	}
	if idx := indexOfArgPair(args, "-threads", "1"); idx > input {
		t.Errorf("復号側の -threads 1 が -i より後ろにある: %v", args)
	}
	if indexOfArgPair(args, "-filter_threads", "1") < 0 {
		t.Errorf("-filter_threads 1 が無い: %v", args)
	}
	if indexOfArgPair(args, "-vf", "scale=400:400:force_original_aspect_ratio=increase,crop=400:400") < 0 {
		t.Errorf("縮小・切り抜きのフィルタが変わっている: %v", args)
	}
	if args[len(args)-1] != "out.tmp" {
		t.Errorf("出力が末尾でない: %v", args)
	}
}

func TestFFmpegThumbArgsVideoKeepsDecoderThreads(t *testing.T) {
	args := ffmpegThumbArgs("in.mp4", "out.tmp", 400, 400, 12.5, false)

	if !slices.Contains(args, "-nostdin") {
		t.Errorf("-nostdin が無い: %v", args)
	}
	if indexOfArgPair(args, "-ss", "12.500") < 0 {
		t.Errorf("-ss が無い: %v", args)
	}
	if slices.Contains(args, "-threads") {
		t.Errorf("動画の復号スレッドを絞っている（HTTP 経路の待ち時間が伸びる）: %v", args)
	}
	if indexOfArgPair(args, "-filter_threads", "1") < 0 {
		t.Errorf("-filter_threads 1 が無い: %v", args)
	}
}

func TestFFmpegDecodeFullArgsIsSingleThreaded(t *testing.T) {
	args := ffmpegDecodeFullArgs("tiled.heic", "full.tmp")

	if !slices.Contains(args, "-nostdin") {
		t.Errorf("-nostdin が無い: %v", args)
	}
	if n := countArgPair(args, "-threads", "1"); n != 2 {
		t.Errorf("-threads 1 が2箇所に無い（%d箇所）: %v", n, args)
	}
	if indexOfArgPair(args, "-filter_threads", "1") < 0 {
		t.Errorf("-filter_threads 1 が無い: %v", args)
	}
	if slices.Contains(args, "-vf") {
		t.Errorf("原寸の書き出しに縮小フィルタが付いている（タイル HEIF で exit 0 のまま何も書かなくなる）: %v", args)
	}
}

func TestVipsThumbArgs(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "abc_123_400x400.jpg")
	tmp := thumbTmpPathExt(dst, ".jpg")
	args := vipsThumbArgs("in.heic", tmp, 400, 300, 85)

	if args[0] != "thumbnail" {
		t.Errorf("操作が thumbnail でない: %v", args)
	}
	if !filepath.IsAbs(tmp) {
		t.Errorf("出力が絶対パスでない（vips は入力の隣へ書く）: %s", tmp)
	}
	if !strings.HasSuffix(tmp, ".jpg") {
		t.Errorf("出力が .jpg で終わっていない（vips はセーバを選べない）: %s", tmp)
	}
	if !strings.HasPrefix(tmp, dst+".") {
		t.Errorf("一時ファイルが最終パスの隣にない: %s", tmp)
	}
	if want := tmp + "[Q=85,strip]"; args[2] != want {
		t.Errorf("出力の指定が違う: got %q want %q", args[2], want)
	}
	if args[3] != "400" || indexOfArgPair(args, "--height", "300") < 0 {
		t.Errorf("幅・高さが違う: %v", args)
	}
	if indexOfArgPair(args, "--crop", "centre") < 0 {
		t.Errorf("--crop centre が無い（縮小だけで切り抜かれない）: %v", args)
	}
	if slices.Contains(args, "--no-rotate") {
		t.Errorf("--no-rotate が付いている（縦向きの写真が横倒しになる）: %v", args)
	}
}
