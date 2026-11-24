package gkill_server_api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// 出力 HTML のメディアは「鍵が画面の引き方と一字一句同じ」でないと引けない。
// 外れても例外は出ず、**出力 HTML がその画像を取りに通信する**（通信しない約束が破れる）。

// TestExportThumbQuery_MatchesClient は、サムネイルの鍵がクライアントの
// build_media_url と同じ形であることを確認する。
func TestExportThumbQuery_MatchesClient(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "client", "classes", "use-idf-kyou-view.ts"))
	if err != nil {
		t.Fatalf("クライアントのソースが読めない: %v", err)
	}
	client := string(source)

	for _, want := range []string{
		`"?` + exportThumbQuery + `"`,
		`"?` + exportVideoThumbQuery + `"`,
	} {
		if !strings.Contains(client, want) {
			t.Errorf("クライアントの build_media_url に %s が無い（鍵が食い違うと出力 HTML が通信する）", want)
		}
	}
}

// TestSplitIDFFileURL は /files/{rep}/{rel} の分け方を確認する。
func TestSplitIDFFileURL(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		fileURL string
		rep     string
		rel     string
		ok      bool
	}{
		{"普通の形", "/files/RepA/dir/a.jpg", "RepA", "dir/a.jpg", true},
		{"エスケープはそのまま渡す", "/files/RepA/%E3%81%82.jpg", "RepA", "%E3%81%82.jpg", true},
		{"rep だけ", "/files/RepA/", "", "", false},
		{"rel だけ", "/files/a.jpg", "", "", false},
		{"別の接頭辞", "/api/get_kyous", "", "", false},
		{"空", "", "", "", false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rep, rel, ok := splitIDFFileURL(testCase.fileURL)
			if ok != testCase.ok || rep != testCase.rep || rel != testCase.rel {
				t.Errorf("splitIDFFileURL(%q) = (%q, %q, %v), want (%q, %q, %v)",
					testCase.fileURL, rep, rel, ok, testCase.rep, testCase.rel, testCase.ok)
			}
		})
	}
}

// TestIsExportTextFile は、本文を焼き込む対象の判定がクライアントと同じであることを確認する。
// ずれると「画面には出るのに本文だけ空」になる。
func TestIsExportTextFile(t *testing.T) {
	for _, testCase := range []struct {
		name string
		idf  reps.IDFKyou
		want bool
	}{
		{"txt", reps.IDFKyou{TargetFile: "memo.txt"}, true},
		{"md", reps.IDFKyou{TargetFile: "memo.md"}, true},
		{"markdown", reps.IDFKyou{TargetFile: "memo.markdown"}, true},
		{"大文字でも同じ", reps.IDFKyou{TargetFile: "MEMO.TXT"}, true},
		{"画像は対象外", reps.IDFKyou{TargetFile: "a.txt", IsImage: true}, false},
		{"動画は対象外", reps.IDFKyou{TargetFile: "a.txt", IsVideo: true}, false},
		{"音声は対象外", reps.IDFKyou{TargetFile: "a.txt", IsAudio: true}, false},
		{"zip は対象外", reps.IDFKyou{TargetFile: "a.txt", IsZip: true}, false},
		{"別の拡張子", reps.IDFKyou{TargetFile: "a.pdf"}, false},
		{"拡張子なし", reps.IDFKyou{TargetFile: "README"}, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := isExportTextFile(testCase.idf); got != testCase.want {
				t.Errorf("isExportTextFile(%+v) = %v, want %v", testCase.idf, got, testCase.want)
			}
		})
	}
}

// TestExportMediaContentType は data: URI に書く種別を確認する。
// 引けないものを image/jpeg などに決め打ちすると、別の種別のファイルが壊れて表示される。
func TestExportMediaContentType(t *testing.T) {
	for _, testCase := range []struct {
		fileName string
		want     string
	}{
		{"a.jpg", "image/jpeg"},
		{"a.JPEG", "image/jpeg"},
		{"a.png", "image/png"},
		{"a.mp4", "video/mp4"},
		{"a.unknown", "application/octet-stream"},
		{"a", "application/octet-stream"},
	} {
		t.Run(testCase.fileName, func(t *testing.T) {
			got := exportMediaContentType(reps.IDFKyou{TargetFile: testCase.fileName})
			if got != testCase.want {
				t.Errorf("exportMediaContentType(%q) = %q, want %q", testCase.fileName, got, testCase.want)
			}
		})
	}
}

// TestExportPluginHTMLKey_MatchesClient は、プラグイン本文の鍵がクライアントの
// plugin_html_key と同じ形であることを確認する。
// 外れても例外は出ず、本文が引けずに空で出る。
func TestExportPluginHTMLKey_MatchesClient(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "client", "classes", "api", "gkill-api.ts"))
	if err != nil {
		t.Fatalf("クライアントのソースが読めない: %v", err)
	}
	// クライアント側は `${rep_name}\u0000${kyou_id}`
	if !strings.Contains(string(source), `${rep_name}\u0000${kyou_id}`) {
		t.Error("クライアントの plugin_html_key が変わっている（鍵が食い違うと本文が空で出る）")
	}
	if got := exportPluginHTMLKey("RepA", "k1"); got != "RepA\x00k1" {
		t.Errorf("exportPluginHTMLKey = %q, want %q", got, "RepA\x00k1")
	}
}

// newExportMediaTestIDFRep は collectExportMedia のテスト用に実ファイルを載せた IDFDirRep を組む。
func newExportMediaTestIDFRep(t *testing.T, files map[string][]byte) (string, reps.IDFKyouRepository) {
	t.Helper()
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "RepMedia")
	if err := os.MkdirAll(filepath.Join(dir, ".gkill"), os.ModePerm); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	for name, data := range files {
		fullPath := filepath.Join(dir, name)
		if err := os.WriteFile(fullPath, data, 0o600); err != nil {
			t.Fatalf("write fixture %s failed: %v", name, err)
		}
	}
	idDB := filepath.Join(dir, ".gkill", "gkill_id.db")
	ignore := []string{}
	idfRep, err := reps.NewIDFDirRep(ctx, "testuser", dir, idDB, true, mux.NewRouter(), false, &ignore, nil)
	if err != nil {
		t.Fatalf("NewIDFDirRep failed: %v", err)
	}
	t.Cleanup(func() { _ = idfRep.Close(ctx) })
	repName, err := idfRep.GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName failed: %v", err)
	}
	return repName, idfRep
}

func tinyPNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	return buf.Bytes()
}

// TestCollectExportMedia_ModesBudgetAndSidecar は、テキスト本文の焼き込み・メディア種別ごとのモード動作・
// 予算超過時の dropped 印と警告・同梱フォルダ（sidecar）・読み込み失敗警告を確認する。
func TestCollectExportMedia_ModesBudgetAndSidecar(t *testing.T) {
	ctx := context.Background()
	pngData := tinyPNGBytes(t)
	pdfData := []byte("%PDF-1.4\nsample attachment payload")
	txtData := []byte("テキスト本文のテスト")

	repName, idfRep := newExportMediaTestIDFRep(t, map[string][]byte{
		"note.txt":  txtData,
		"doc.pdf":   pdfData,
		"photo.png": pngData,
	})

	repositories := &reps.GkillRepositories{
		IDFKyouReps: reps.IDFKyouRepositories{idfRep},
	}
	g := &GkillServerAPI{}

	makeIDFs := func() []reps.IDFKyou {
		return []reps.IDFKyou{
			{ID: "idf-txt", TargetFile: "note.txt", FileURL: "/files/" + repName + "/note.txt"},
			{ID: "idf-pdf", TargetFile: "doc.pdf", FileURL: "/files/" + repName + "/doc.pdf"},
			{ID: "idf-img", TargetFile: "photo.png", FileURL: "/files/" + repName + "/photo.png", IsImage: true},
			{ID: "idf-missing", TargetFile: "missing.pdf", FileURL: "/files/" + repName + "/missing.pdf"},
		}
	}

	t.Run("MediaModeNone でもテキスト本文は焼き込み、メディアは焼かない", func(t *testing.T) {
		payload := minimalExportPayload("none")
		payload.IDFKyous = makeIDFs()
		warnings, err := g.collectExportMedia(ctx, &req_res.ExportKyouHTMLRequest{
			MediaMode: req_res.ExportMediaModeNone,
		}, repositories, payload, nil)
		if err != nil {
			t.Fatalf("collectExportMedia failed: %v", err)
		}
		if len(warnings) != 0 {
			t.Errorf("unexpected warnings: %v", warnings)
		}
		entry, ok := payload.IDFTextContents["idf-txt"]
		if !ok {
			t.Fatal("MediaModeNone でもテキスト本文が焼き込まれるべき")
		}
		decoded, err := base64.StdEncoding.DecodeString(entry.BytesBase64)
		if err != nil || string(decoded) != string(txtData) {
			t.Errorf("decoded text = %q (err=%v), want %q", string(decoded), err, string(txtData))
		}
		if len(payload.Media) != 0 {
			t.Errorf("MediaModeNone でメディアが焼かれている: %+v", payload.Media)
		}
	})

	t.Run("MediaModeFull + sidecar で添付・原寸が相対パスになり、欠損ファイルは警告になる", func(t *testing.T) {
		payload := minimalExportPayload("sidecar")
		payload.IDFKyous = makeIDFs()
		sidecar := newExportSidecar("export_test")
		warnings, err := g.collectExportMedia(ctx, &req_res.ExportKyouHTMLRequest{
			MediaMode: req_res.ExportMediaModeFull,
		}, repositories, payload, sidecar)
		if err != nil {
			t.Fatalf("collectExportMedia failed: %v", err)
		}
		assertExportWarnings(t, warnings, "1 件のファイルを読み込めませんでした")
		pdfEntry, ok := payload.Media["/files/"+repName+"/doc.pdf"]
		// 同名フォルダの中の名前は内容のハッシュ（export_sidecar.go）。メディアの収集がその名前を src に使う
		pdfSum := sha256.Sum256(pdfData)
		wantPDFSrc := "export_test/" + hex.EncodeToString(pdfSum[:8]) + ".pdf"
		if !ok || pdfEntry.Kind != "attachment" || pdfEntry.Src != wantPDFSrc {
			t.Errorf("pdf entry = %+v, want attachment %q", pdfEntry, wantPDFSrc)
		}
		fullEntry, ok := payload.Media["/files/"+repName+"/photo.png"]
		if !ok || fullEntry.Kind != "full" || !strings.HasPrefix(fullEntry.Src, "export_test/") {
			t.Errorf("full image entry = %+v, want full in sidecar", fullEntry)
		}
	})

	t.Run("予算超過時は dropped 印が付き警告が出る", func(t *testing.T) {
		payload := minimalExportPayload("budget")
		payload.IDFKyous = []reps.IDFKyou{
			{ID: "idf-pdf", TargetFile: "doc.pdf", FileURL: "/files/" + repName + "/doc.pdf"},
		}
		warnings, err := g.collectExportMedia(ctx, &req_res.ExportKyouHTMLRequest{
			MediaMode:           req_res.ExportMediaModeFull,
			MediaBudgetBytes:    10,
			MediaItemLimitBytes: 10,
		}, repositories, payload, nil)
		if err != nil {
			t.Fatalf("collectExportMedia failed: %v", err)
		}
		assertExportWarnings(t, warnings, "予算に収まらないため 1 件のメディア・添付ファイルを含めませんでした")
		dropped, ok := payload.Media["/files/"+repName+"/doc.pdf"]
		if !ok || dropped.Kind != exportMediaKindDropped || dropped.Src != "" {
			t.Errorf("dropped entry = %+v, want kind=%q src=\"\"", dropped, exportMediaKindDropped)
		}
	})

	// サムネイルの鍵はクライアントの build_media_url と同じ `?thumb=400x400`。外れると出力 HTML が通信する
	t.Run("MediaModeThumbnail は画像をサムネイルの鍵で焼き、原寸は焼かない", func(t *testing.T) {
		payload := minimalExportPayload("thumb")
		payload.IDFKyous = []reps.IDFKyou{
			{ID: "idf-img", TargetFile: "photo.png", FileURL: "/files/" + repName + "/photo.png", IsImage: true},
		}
		warnings, err := g.collectExportMedia(ctx, &req_res.ExportKyouHTMLRequest{
			LocaleName: "ja",
			MediaMode:  req_res.ExportMediaModeThumbnail,
		}, repositories, payload, nil)
		if err != nil {
			t.Fatalf("collectExportMedia failed: %v", err)
		}
		if len(warnings) != 0 {
			t.Errorf("unexpected warnings: %v", warnings)
		}
		thumbKey := "/files/" + repName + "/photo.png?" + exportThumbQuery
		thumb, ok := payload.Media[thumbKey]
		if !ok || thumb.Kind != "thumb" || !strings.HasPrefix(thumb.Src, "data:image/jpeg;base64,") {
			t.Errorf("thumb entry = %+v (ok=%v), want thumb data URI under %q", thumb, ok, thumbKey)
		}
		if _, exist := payload.Media["/files/"+repName+"/photo.png"]; exist {
			t.Error("サムネイルのみなのに原寸が焼かれている")
		}
	})
}

// assertExportWarnings は警告が1件だけ、warning の MSG000096 で、差し込みを済ませた文言で返ることを確かめる。
func assertExportWarnings(t *testing.T, warnings []*message.GkillMessage, wantPrefix string) {
	t.Helper()
	if len(warnings) != 1 {
		t.Fatalf("warnings = %d 件, want 1件: %+v", len(warnings), warnings)
	}
	warning := warnings[0]
	if warning.MessageCode != message.ExportKyouHTMLPartialWarningMessage || warning.Level != message.MessageLevelWarning {
		t.Errorf("warning = %+v, want code %s level %s", warning, message.ExportKyouHTMLPartialWarningMessage, message.MessageLevelWarning)
	}
	if !strings.HasPrefix(warning.Message, wantPrefix) || strings.Contains(warning.Message, "{") {
		t.Errorf("warning message = %q, want prefix %q（差し込み済み）", warning.Message, wantPrefix)
	}
}
