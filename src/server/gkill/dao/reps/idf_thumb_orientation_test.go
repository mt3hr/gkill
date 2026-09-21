package reps

// EXIF Orientation を持つ JPEG のサムネイルが正しい向きで作られることの回帰テスト。
//
// Go 経路は 2026-09-22 に「原寸を回してから縮小」から「縮小してから回す」へ変えた
// （縦向きのスマホ写真で回転が縮小より重かった）。回転で軸が入れ替わる向き（5〜8）は
// 切り抜きのアスペクトを源座標で計算し直す必要があり、間違えると
// エラーも出ずに縦横比の崩れたサムネイルになる。vips 経路は vips 自身が回すが、
// 出力から Orientation タグを落とさないとブラウザがもう一度回すので、同じ検査に通す。
//
// 被写体は 64×64 で左半分が赤・右半分が青。中央切り抜きでどのアスペクトでも境界が真ん中に残る。

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// writeTestJPEGWithOrientation は左半分が赤・右半分が青の 64×64 JPEG に、
// EXIF の Orientation だけを持つ APP1 セグメントを手組みで挿して書く。
// goexif は復号専用で書けないので、TIFF ヘッダと IFD0 の1エントリを直接並べる。
func writeTestJPEGWithOrientation(t *testing.T, filePath string, orient int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			c := color.RGBA{R: 32, G: 32, B: 200, A: 255}
			if x < 32 {
				c = color.RGBA{R: 220, G: 30, B: 30, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	raw := &bytes.Buffer{}
	if err := jpeg.Encode(raw, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("jpeg.Encode failed: %v", err)
	}

	tiff := &bytes.Buffer{}
	tiff.WriteString("II") // リトルエンディアン
	for _, v := range []any{
		uint16(0x2A), uint32(8), // TIFF マジックと IFD0 のオフセット
		uint16(1),                 // エントリ数
		uint16(0x0112), uint16(3), // Orientation, SHORT
		uint32(1), uint16(orient), uint16(0), // 個数, 値, 詰め物
		uint32(0), // 次の IFD 無し
	} {
		if err := binary.Write(tiff, binary.LittleEndian, v); err != nil {
			t.Fatalf("binary.Write failed: %v", err)
		}
	}
	app1 := append([]byte("Exif\x00\x00"), tiff.Bytes()...)

	out := &bytes.Buffer{}
	out.Write(raw.Bytes()[:2]) // SOI
	out.Write([]byte{0xFF, 0xE1})
	if err := binary.Write(out, binary.BigEndian, uint16(len(app1)+2)); err != nil {
		t.Fatalf("binary.Write failed: %v", err)
	}
	out.Write(app1)
	out.Write(raw.Bytes()[2:])

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filePath, out.Bytes(), os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func decodeThumbForTest(t *testing.T, thumbPath string) image.Image {
	t.Helper()
	f, err := os.Open(thumbPath)
	if err != nil {
		t.Fatalf("サムネイルが作られていない: %v", err)
	}
	defer func() { _ = f.Close() }()
	img, format, err := image.Decode(f)
	if err != nil {
		t.Fatalf("サムネイルを復号できない: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("サムネイルが JPEG でない: %s", format)
	}
	return img
}

func isReddish(c color.Color) bool {
	r, _, b, _ := c.RGBA()
	return r>>8 > 150 && b>>8 < 100
}

func isBluish(c color.Color) bool {
	r, _, b, _ := c.RGBA()
	return b>>8 > 150 && r>>8 < 100
}

// orientationExpectation は Orientation ごとに「赤い半分がどちら側に来るか」。
// 1: そのまま（左）、3: 180度（右）、6: 90度時計回り（上）、8: 270度時計回り（下）。
var orientationExpectation = map[int]string{1: "left", 3: "right", 6: "top", 8: "bottom"}

func assertThumbOrientation(t *testing.T, img image.Image, w, h int, redSide string) {
	t.Helper()
	b := img.Bounds()
	if b.Dx() != w || b.Dy() != h {
		t.Fatalf("寸法が %dx%d でない: %dx%d", w, h, b.Dx(), b.Dy())
	}
	left, right := img.At(w/4, h/2), img.At(3*w/4, h/2)
	top, bottom := img.At(w/2, h/4), img.At(w/2, 3*h/4)
	var red, blue color.Color
	switch redSide {
	case "left":
		red, blue = left, right
	case "right":
		red, blue = right, left
	case "top":
		red, blue = top, bottom
	case "bottom":
		red, blue = bottom, top
	}
	if !isReddish(red) {
		t.Errorf("赤い半分が %s 側に無い: %v", redSide, red)
	}
	if !isBluish(blue) {
		t.Errorf("青い半分が %s の反対側に無い: %v", redSide, blue)
	}
}

// Go 経路（vips 無し）で、Orientation 1/3/6/8 と正方形・横長・縦長の要求サイズの全組み合わせを固定する。
func TestGenerateThumbNativeAppliesExifOrientation(t *testing.T) {
	origVIPS := existVIPS
	existVIPS = false
	t.Cleanup(func() { existVIPS = origVIPS })

	base := t.TempDir()
	for orient, redSide := range orientationExpectation {
		src := filepath.Join(base, "src.jpg")
		writeTestJPEGWithOrientation(t, src, orient)
		for _, size := range [][2]int{{40, 40}, {40, 20}, {20, 40}} {
			dst := filepath.Join(base, "thumb.jpg")
			_ = os.Remove(dst)
			if err := generateThumbJpeg(context.Background(), src, dst, size[0], size[1], 85); err != nil {
				t.Fatalf("orient=%d %dx%d: 生成に失敗した: %v", orient, size[0], size[1], err)
			}
			t.Run(fmt.Sprintf("orient%d_%dx%d", orient, size[0], size[1]), func(t *testing.T) {
				assertThumbOrientation(t, decodeThumbForTest(t, dst), size[0], size[1], redSide)
			})
		}
	}
}

// scaleThumbImage が回転で軸の入れ替わる向きでも要求どおりの寸法を返すこと。
// 切り抜きのアスペクトを源座標で入れ替え忘れると、縦横比が崩れたまま (h,w) を回して (w,h) にならない。
func TestScaleThumbImageSwapsAxesForRotatedOrientations(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 300, 100))
	for orient := 1; orient <= 8; orient++ {
		got := scaleThumbImage(src, 80, 30, orient)
		if b := got.Bounds(); b.Dx() != 80 || b.Dy() != 30 {
			t.Errorf("orient=%d: 寸法が 80x30 でない: %dx%d", orient, b.Dx(), b.Dy())
		}
	}
}
