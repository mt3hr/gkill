package reps

// サムネイルの Go 経路（vips が無い環境と、vips が読めなかったファイル）の縮小と回転のベンチマーク。
//
// 12MP（4000×3000）の JPEG を想定して、image/jpeg が返す *image.YCbCr（4:2:0）を合成し、
// 400×400 のサムネイルを作る各手順を比べる。絶対値は環境で大きく変わるので比率だけ見ること。
//
//	go test -run '^$' -bench Thumb -benchmem ./gkill/dao/reps/
//
// 比べるもの:
//   - BenchmarkThumbScale: 縮小カーネルと合成モード（旧: CatmullRom + Over、新: BiLinear + Src）。
//     ApproxBiLinear は最速だが近傍4画素しか見ないので 7倍縮小で縞が出る（採らない理由の物差し）。
//   - BenchmarkThumbOrientation6: EXIF 回転の位置（旧: 原寸で回してから縮小、新: 縮小してから回す）。
//   - BenchmarkThumbBackends: vips / Go の端から端まで（12MP の JPEG と 3.7MP の PNG）。

import (
	"context"
	"image"
	stdDraw "image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	xdraw "golang.org/x/image/draw"
)

// newBenchYCbCr は image/jpeg の復号結果と同じ型・同じ画素形式の大きな画像を合成する。
// 縞や勾配を入れて、縮小が画素を実際に読む形にしておく。
func newBenchYCbCr(w, h int) *image.YCbCr {
	img := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio420)
	for y := range h {
		for x := range w {
			img.Y[img.YOffset(x, y)] = uint8((x + y) % 256)
		}
	}
	for y := 0; y < h; y += 2 {
		for x := 0; x < w; x += 2 {
			ci := img.COffset(x, y)
			img.Cb[ci] = uint8(x % 256)
			img.Cr[ci] = uint8(y % 256)
		}
	}
	return img
}

const (
	benchSrcW = 4000
	benchSrcH = 3000
	benchDstW = 400
	benchDstH = 400
)

func BenchmarkThumbScale(b *testing.B) {
	src := newBenchYCbCr(benchSrcW, benchSrcH)
	cropped := cropCenterToAspect(src, float64(benchDstW)/float64(benchDstH))

	cases := []struct {
		name   string
		scaler xdraw.Scaler
		op     stdDraw.Op
	}{
		{"catmullrom_over_old", xdraw.CatmullRom, stdDraw.Over},
		{"catmullrom_src", xdraw.CatmullRom, stdDraw.Src},
		{"bilinear_src_new", xdraw.BiLinear, stdDraw.Src},
		{"approxbilinear_src_rejected", xdraw.ApproxBiLinear, stdDraw.Src},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				dst := image.NewRGBA(image.Rect(0, 0, benchDstW, benchDstH))
				c.scaler.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), c.op, nil)
			}
		})
	}
}

func BenchmarkThumbOrientation6(b *testing.B) {
	src := newBenchYCbCr(benchSrcW, benchSrcH)

	// 旧: 原寸を NRGBA に変換して回し、それから切り抜き・縮小（2026-09-22 まで）
	b.Run("rotate_full_then_scale_old", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			rotated := applyExifOrientation(src, 6)
			cropped := cropCenterToAspect(rotated, float64(benchDstW)/float64(benchDstH))
			dst := image.NewRGBA(image.Rect(0, 0, benchDstW, benchDstH))
			xdraw.CatmullRom.Scale(dst, dst.Bounds(), cropped, cropped.Bounds(), stdDraw.Over, nil)
		}
	})

	// 新: 切り抜き・縮小してから 400×400 を回す
	b.Run("scale_then_rotate_new", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = scaleThumbImage(src, benchDstW, benchDstH, 6)
		}
	})
}

// writeBenchJPEG は w×h の JPEG を書く（4000×3000 なら 12MP のスマホ写真相当）。
func writeBenchJPEG(b *testing.B, path string, w, h int) {
	b.Helper()
	f, err := os.Create(path)
	if err != nil {
		b.Fatalf("Create failed: %v", err)
	}
	if err := jpeg.Encode(f, newBenchYCbCr(w, h), &jpeg.Options{Quality: 90}); err != nil {
		b.Fatalf("jpeg.Encode failed: %v", err)
	}
	if err := f.Close(); err != nil {
		b.Fatalf("Close failed: %v", err)
	}
}

// writeBenchPNG は w×h の PNG を書く（2560×1440 なら画面のスクリーンショット相当）。
func writeBenchPNG(b *testing.B, path string, w, h int) {
	b.Helper()
	f, err := os.Create(path)
	if err != nil {
		b.Fatalf("Create failed: %v", err)
	}
	if err := png.Encode(f, newBenchYCbCr(w, h)); err != nil {
		b.Fatalf("png.Encode failed: %v", err)
	}
	if err := f.Close(); err != nil {
		b.Fatalf("Close failed: %v", err)
	}
}

// BenchmarkThumbBackends は同じ画像から 400×400 を作る端から端までの時間をバックエンドごとに比べる。
// プロセス起動と復号を含む、実際の1件あたりの時間に近い。
//
//   - jpeg12mp: 12MP の写真。vips は shrink-on-load（DCT 段階で 1/8）が効き、Go は原寸復号になる
//   - png3mp: 2560×1440 のスクリーンショット。どちらも原寸復号で、vips はプロセス起動ぶん不利
//   - routed: generateThumbJpeg の入口（preferNativeThumbDecode の振り分け込み）
//
// vips が無い環境では vips の計測を飛ばす。初回の vips 起動は OS の実行ファイル走査で
// 1秒を超えることがあるので、計測前に1回空回しする。
func BenchmarkThumbBackends(b *testing.B) {
	base := b.TempDir()
	dst := filepath.Join(base, "thumb.jpg")
	jpeg12mp := filepath.Join(base, "photo.jpg")
	writeBenchJPEG(b, jpeg12mp, benchSrcW, benchSrcH)
	png3mp := filepath.Join(base, "screen.png")
	writeBenchPNG(b, png3mp, 2560, 1440)

	if existVIPS {
		_ = runVipsThumb(context.Background(), png3mp, dst, benchDstW, benchDstH, 85)
	}

	for _, src := range []struct {
		name string
		path string
	}{{"jpeg12mp", jpeg12mp}, {"png3mp", png3mp}} {
		b.Run(src.name+"/vips", func(b *testing.B) {
			if !existVIPS {
				b.Skip("vips が無い")
			}
			b.ReportAllocs()
			for b.Loop() {
				if err := runVipsThumb(context.Background(), src.path, dst, benchDstW, benchDstH, 85); err != nil {
					b.Fatalf("生成に失敗した: %v", err)
				}
			}
		})
		b.Run(src.name+"/native", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := generateThumbJpegNative(context.Background(), src.path, dst, benchDstW, benchDstH, 85); err != nil {
					b.Fatalf("生成に失敗した: %v", err)
				}
			}
		})
		b.Run(src.name+"/routed", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := generateThumbJpeg(context.Background(), src.path, dst, benchDstW, benchDstH, 85); err != nil {
					b.Fatalf("生成に失敗した: %v", err)
				}
			}
		})
	}
}
