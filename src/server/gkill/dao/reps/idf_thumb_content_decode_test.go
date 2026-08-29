package reps

// サムネイルを「拡張子ではなく中身」で作ることの回帰テスト。
//
// isImage が通す37拡張子のうち、Go に image.Decode のデコーダがあるのは
// jpeg/png/gif/webp の4形式だけ。heic/avif/bmp/ico/tiff はデコーダが無く、
// 拡張子が実体と食い違っているファイル（中身がBMPの .jpg など）も同じところで落ちる。
// どちらも ffmpeg へ落として拾う。
//
// ここが壊れると、サムネイルが出ないだけでなく、失敗の印が残らないぶん
// 一括生成のたびに同じファイルを全部やり直すようになる。どちらも例外にならない。

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestBMP は最小構成の24bit BMP を書く。
//
// golang.org/x/image/bmp を使わないのは、あれを import すると init() が
// image.RegisterFormat で BMP デコーダを登録してしまい、
// テストの中だけ image.Decode が BMP を読めるようになるため。
// 検証したいのは「Go が読めない形式を ffmpeg で拾う」経路なので、それでは意味がない。
func writeTestBMP(t *testing.T, filePath string) {
	t.Helper()
	const w, h = 16, 16
	rowSize := w * 3 // 4の倍数なのでパディング不要
	pixelSize := rowSize * h

	buf := &bytes.Buffer{}
	buf.WriteString("BM")
	for _, v := range []any{
		uint32(14 + 40 + pixelSize), // ファイルサイズ
		uint16(0), uint16(0),        // 予約
		uint32(14 + 40), // 画素データまでのオフセット
		uint32(40),      // BITMAPINFOHEADER
		int32(w), int32(h),
		uint16(1), uint16(24), // プレーン数, ビット深度
		uint32(0), uint32(pixelSize), // 無圧縮, 画素データ長
		int32(2835), int32(2835), // 解像度
		uint32(0), uint32(0), // 使用色数, 重要色数
	} {
		if err := binary.Write(buf, binary.LittleEndian, v); err != nil {
			t.Fatalf("binary.Write failed: %v", err)
		}
	}
	for y := range h {
		for x := range w {
			buf.WriteByte(uint8(x * 16)) // B
			buf.WriteByte(uint8(y * 16)) // G
			buf.WriteByte(0x40)          // R
		}
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filePath, buf.Bytes(), os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

// writePNGPaddedTo はデコードできるPNGを、ちょうど size バイトになるよう0で埋めて書く。
// キャッシュ名にはファイルサイズが入るので、サイズを保ったまま中身だけ差し替えたいときに使う。
func writePNGPaddedTo(t *testing.T, filePath string, size int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			img.Set(x, y, color.RGBA{R: uint8(x * 16), G: 0, B: uint8(y * 16), A: 255})
		}
	}
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	if buf.Len() > size {
		t.Fatalf("PNGが %d バイトあり、%d バイトに収まらない", buf.Len(), size)
	}
	buf.Write(make([]byte, size-buf.Len()))
	if err := os.WriteFile(filePath, buf.Bytes(), os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func requireFFMPEG(t *testing.T) {
	t.Helper()
	if !existFFMPEG {
		t.Skip("ffmpeg が無い環境なのでスキップする")
	}
}

// Go にデコーダが無い形式（BMP）でも、拡張子が実体と食い違っていても、
// ffmpeg 経由でサムネイルが作られること。
//
// 手元では、サムネイル生成に失敗するファイルの多くがこの2つの形だった。
func TestGenerateThumbCacheFallsBackToFFmpeg(t *testing.T) {
	requireFFMPEG(t)

	repo, contentDir, thumbCacheDir := newIDFRepForThumbBatchTest(t)
	ctx := context.Background()

	// 拡張子どおりのBMPと、中身がBMPなのに .jpg を名乗るファイル
	writeTestBMP(t, filepath.Join(contentDir, "honest.bmp"))
	writeTestBMP(t, filepath.Join(contentDir, "liar.jpg"))
	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("IDF failed: %v", err)
	}

	if err := repo.GenerateThumbCache(ctx); err != nil {
		t.Fatalf("GenerateThumbCache failed: %v", err)
	}

	for _, rel := range []string{"honest.bmp", "liar.jpg"} {
		name := thumbCacheNameForTest(t, repo, contentDir, rel)
		got, err := os.ReadFile(filepath.Join(thumbCacheDir, name))
		if err != nil {
			t.Errorf("%s のサムネイルが作られていない: %v", rel, err)
			continue
		}
		if !isJPEG(got) {
			t.Errorf("%s のサムネイルがJPEGでない: %v", rel, got[:min(4, len(got))])
		}
		if _, err := os.Stat(filepath.Join(thumbCacheDir, name+thumbFailedMarkerSuffix)); err == nil {
			t.Errorf("%s は成功しているのに失敗の印が残っている", rel)
		}
	}
}

// ffmpeg が exit 0 のまま何も書かないことがある。
// 検出しないと直後の os.Rename が「指定されたファイルが見つかりません」という、
// ffmpeg が何もしなかったことを隠したエラーになる。
//
// 出力が無いときに ffmpeg 自身が非ゼロで終わるかはビルドと素材で変わるので、
// 終了コードではなく出力の有無で判定する側を直接固定する。
func TestFinalizeFFmpegThumbOutputDetectsEmptyOutput(t *testing.T) {
	base := t.TempDir()
	src := filepath.Join(base, "source.mov")
	dst := filepath.Join(base, "thumb.jpg")
	tmp := dst + ".tmp"

	// tmp がそもそも作られていない（exit 0 で何も書かなかった場合）
	err := finalizeFFmpegThumbOutput(src, tmp, dst, "ffmpeg said nothing")
	if err == nil {
		t.Fatal("出力が無いのに成功している")
	}
	if !strings.Contains(err.Error(), "wrote no output") {
		t.Errorf("空出力として扱われていない: %v", err)
	}
	if _, statErr := os.Stat(dst); statErr == nil {
		t.Error("出力が無いのに dst が作られている")
	}

	// tmp はあるが0バイト
	if err := os.WriteFile(tmp, nil, os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := finalizeFFmpegThumbOutput(src, tmp, dst, ""); err == nil {
		t.Error("0バイトの出力を成功として扱っている")
	}
	if _, statErr := os.Stat(tmp); statErr == nil {
		t.Error("失敗したのに .tmp が残っている")
	}

	// 中身があれば dst へ移すこと
	if err := os.WriteFile(tmp, []byte{0xFF, 0xD8, 0xFF, 0x00}, os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := finalizeFFmpegThumbOutput(src, tmp, dst, ""); err != nil {
		t.Fatalf("中身があるのに失敗している: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("dst へ移されていない: %v", err)
	}
	if !isJPEG(got) {
		t.Errorf("移された中身が違う: %v", got[:min(4, len(got))])
	}
	if _, statErr := os.Stat(tmp); statErr == nil {
		t.Error("成功したのに .tmp が残っている")
	}
}

// 動画の名前が付いた静止画でも、シーク無しでやり直してサムネイルが作られること。
// 手元にも中身が JPEG の .mp4 があり、いずれも rename エラーで落ちていた。
func TestGenerateVideoThumbRetriesWithoutSeek(t *testing.T) {
	requireFFMPEG(t)

	base := t.TempDir()
	// 中身はPNGだが .mp4 を名乗る。動画として扱われて generateVideoThumbJpeg へ入る
	src := filepath.Join(base, "actually_an_image.mp4")
	writePNGPaddedTo(t, src, 512)
	dst := filepath.Join(base, "thumb.jpg")

	if err := generateVideoThumbJpeg(context.Background(), src, dst, 64, 64); err != nil {
		t.Fatalf("静止画の .mp4 からサムネイルが作れていない: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !isJPEG(got) {
		t.Errorf("生成されたサムネイルがJPEGでない: %v", got[:min(4, len(got))])
	}
}

// 縮小フィルタを付けられない入力でも、原寸で起こしてから Go 側で縮小できること。
//
// タイルに分割された HEIF は ffmpeg が内部で complex filtergraph を組んでタイルを貼り合わせる。
// そこへ簡易フィルタ（-vf）を足すと「Simple and complex filtering cannot be used together」で
// 拒否され、**何も書かないまま exit 0** になる。実際に手元の .HEIC がこれで、
// 失敗するファイルの中で最も多い形だった。
func TestRunFFmpegDecodeFullWritesOriginalSize(t *testing.T) {
	requireFFMPEG(t)

	base := t.TempDir()
	src := filepath.Join(base, "src.png")
	writePNGPaddedTo(t, src, 512)
	dst := filepath.Join(base, "full.jpg")

	if err := runFFmpegDecodeFull(context.Background(), src, dst); err != nil {
		t.Fatalf("原寸の書き出しに失敗した: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !isJPEG(got) {
		t.Errorf("書き出されたものがJPEGでない: %v", got[:min(4, len(got))])
	}
	// 縮小していないこと（元は 8x8）
	cfg, _, err := image.DecodeConfig(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("DecodeConfig failed: %v", err)
	}
	if cfg.Width != 8 || cfg.Height != 8 {
		t.Errorf("原寸で書かれていない: %dx%d", cfg.Width, cfg.Height)
	}
	// 書きかけの一時ファイルを残さないこと
	for _, e := range mustReadDir(t, base) {
		if strings.HasSuffix(e, ".tmp") {
			t.Errorf("一時ファイルが残っている: %s", e)
		}
	}
}

func mustReadDir(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	names := make([]string, 0, len(es))
	for _, e := range es {
		names = append(names, e.Name())
	}
	return names
}

// 生成に失敗したら印を残し、2回目は生成そのものを試みないこと。
//
// 印が無いと、デコードできないファイルは一括生成のたびに全件やり直しになる。
// ffmpeg の有無に関わらず、どちらの経路でも失敗すればここへ来る。
func TestGenerateThumbCacheMarksFailureAndSkipsRetry(t *testing.T) {
	repo, contentDir, thumbCacheDir := newIDFRepForThumbBatchTest(t)
	ctx := context.Background()

	const brokenSize = 4096
	target := filepath.Join(contentDir, "broken.png")
	if err := os.WriteFile(target, bytes.Repeat([]byte{0x5a}, brokenSize), os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("IDF failed: %v", err)
	}

	name := thumbCacheNameForTest(t, repo, contentDir, "broken.png")
	if err := repo.GenerateThumbCache(ctx); err != nil {
		t.Fatalf("GenerateThumbCache failed: %v", err)
	}

	markerPath := filepath.Join(thumbCacheDir, name+thumbFailedMarkerSuffix)
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("失敗の印が残っていない: %v", err)
	}
	if _, err := os.Stat(filepath.Join(thumbCacheDir, name)); err == nil {
		t.Fatal("失敗したのにサムネイルが作られている")
	}

	// 中身だけ読めるPNGへ差し替える。サイズを保つのでキャッシュ名は変わらない。
	// 印が効いていれば、読めるようになっても生成は走らない。
	writePNGPaddedTo(t, target, brokenSize)
	if newName := thumbCacheNameForTest(t, repo, contentDir, "broken.png"); newName != name {
		t.Fatalf("差し替えでキャッシュ名が変わっている: %s -> %s", name, newName)
	}
	if err := repo.GenerateThumbCache(ctx); err != nil {
		t.Fatalf("GenerateThumbCache failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(thumbCacheDir, name)); err == nil {
		t.Error("失敗の印があるのに生成をやり直している")
	}
}

// ブラウザの切断を恒久的な失敗として焼かないこと。
// HTTP経由では待ちきれずに接続を切られると ffmpeg も一緒に落ちる。
func TestMarkThumbFailedSkipsWhenContextCancelled(t *testing.T) {
	base := t.TempDir()
	thumbPath := filepath.Join(base, "thumb.jpg")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	markThumbFailed(ctx, thumbPath, os.ErrClosed)
	if _, err := os.Stat(thumbPath + thumbFailedMarkerSuffix); err == nil {
		t.Error("ctx が切れているのに失敗の印を残している")
	}

	markThumbFailed(context.Background(), thumbPath, os.ErrClosed)
	if _, err := os.Stat(thumbPath + thumbFailedMarkerSuffix); err != nil {
		t.Errorf("ctx が生きているのに失敗の印が残っていない: %v", err)
	}
}

// 書きかけのサムネイルの置き場は、呼ぶたびに違う名前になること。
//
// サーバ本体・MCPサーバ・generate_thumb_cache は別プロセスで同じキャッシュ置き場を共有する。
// 固定名の .tmp を共有すると、片方の rename がもう片方の消した tmp を掴んで失敗し、
// 実際には作れるサムネイルに失敗の印が焼かれる。
func TestThumbTmpPathIsUnique(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "thumb.jpg")
	seen := map[string]struct{}{}
	for range 100 {
		p := thumbTmpPath(dst)
		if !strings.HasPrefix(p, dst+".") || !strings.HasSuffix(p, ".tmp") {
			t.Fatalf("想定した形でない: %s", p)
		}
		if _, dup := seen[p]; dup {
			t.Fatalf("同じ一時ファイル名が2回出た: %s", p)
		}
		seen[p] = struct{}{}
	}
}

// SVG はサムネイルを作らないこと。
// ブラウザは img でそのまま描けるが、image.Decode も ffmpeg もラスタ化できないので、
// 対象にすると必ず失敗して原本へフォールバックするぶんの無駄が毎回かかる。
func TestLooksLikeThumbTargetSkipsSVG(t *testing.T) {
	for _, rel := range []string{"icon.svg", "sub/ICON.SVG"} {
		if looksLikeThumbTarget(rel, false) {
			t.Errorf("%s がサムネイル対象になっている", rel)
		}
	}
	for _, rel := range []string{"photo.jpg", "photo.HEIC", "shot.bmp"} {
		if !looksLikeThumbTarget(rel, false) {
			t.Errorf("%s がサムネイル対象から外れている", rel)
		}
	}
}

// 動画の拡張子の取りこぼしが無いこと。
//
// .mpeg は長く抜けていて、中身が MPEG の .mpeg が動画として扱われていなかった。
// .mwv は .wmv の打ち間違いで、実在しない綴りのぶん判定が1つ無駄になっていた。
func TestIsVideoCoversMpegAndDropsTypo(t *testing.T) {
	for _, name := range []string{"a.mpg", "a.mpeg", "a.MPEG", "a.wmv", "a.mp4", "a.mov"} {
		if !isVideo(name) {
			t.Errorf("%s が動画として扱われていない", name)
		}
	}
	if isVideo("a.mwv") {
		t.Error(".mwv（.wmv の打ち間違い）がまだ残っている")
	}
}

// 失敗の印を「生成済みサムネイル」として数えないこと。
// 混ぜると一括生成側が件数を取り違える。
func TestCachedThumbNamesExcludesFailedMarkers(t *testing.T) {
	repo, contentDir, thumbCacheDir := newIDFRepForThumbBatchTest(t)

	if err := os.MkdirAll(thumbCacheDir, os.ModePerm); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	writeTestImage(t, filepath.Join(contentDir, "ok.png"), 0)
	name := thumbCacheNameForTest(t, repo, contentDir, "ok.png")
	if err := os.WriteFile(filepath.Join(thumbCacheDir, name), []byte("thumb"), os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(thumbCacheDir, name+thumbFailedMarkerSuffix), []byte("failed"), os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	names, err := repo.thumbGenerator.CachedThumbNames()
	if err != nil {
		t.Fatalf("CachedThumbNames failed: %v", err)
	}
	if _, ok := names[name]; !ok {
		t.Error("生成済みのサムネイルが集合に入っていない")
	}
	if _, ok := names[name+thumbFailedMarkerSuffix]; ok {
		t.Error("失敗の印が生成済みとして数えられている")
	}
}
