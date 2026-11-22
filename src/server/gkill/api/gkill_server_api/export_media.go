package gkill_server_api

// 出力 HTML へメディア（画像・動画・添付ファイル）と IDF のテキスト本文を焼き込む。
//
// **新しい読み出し経路を書かないこと。** サムネイルの生成・動画のポスター・原寸の配信は
// どれも rep の HandleFileServe が持っている1実装で、同じ判定をもう一度書くと
// 「画面では出るのに出力では出ない」形でずれる。ここではその配信ハンドラを
// メモリ上の ResponseWriter で呼ぶ。認証と CSP は上位の HandleFileServe
// （GkillServerAPI 側）の仕事なので、rep のハンドラを直に呼んでもそこは通らない。

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

const (
	// exportThumbQuery はクライアントの build_media_url と同じ形。
	// ここを変えると焼き込みの鍵が画面の引き方と食い違い、**出力 HTML が通信してしまう**。
	exportThumbQuery      = "thumb=400x400"
	exportVideoThumbQuery = "is_video=true&thumb=400x400"

	// exportMediaBudgetBytes はメディアに使える生バイトの上限。
	// base64 は素のバイト数の 4/3 になるので、応答はこれより大きくなる。
	exportMediaBudgetBytes = 48 * 1024 * 1024

	// exportTextBudgetBytes は IDF のテキスト / Markdown 本文に使える生バイトの上限。
	// テキストは軽いので、メディアとは別枠にする（埋め込まないを選んでも本文は焼く）。
	exportTextBudgetBytes = 8 * 1024 * 1024
)

// bufferResponseWriter は配信ハンドラの出力をメモリへ受ける。
// httptest を本番のコードへ持ち込まないために自前で持つ。
type bufferResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	code   int
}

func newBufferResponseWriter() *bufferResponseWriter {
	return &bufferResponseWriter{header: http.Header{}, code: http.StatusOK}
}

func (b *bufferResponseWriter) Header() http.Header { return b.header }

func (b *bufferResponseWriter) Write(p []byte) (int, error) { return b.body.Write(p) }

func (b *bufferResponseWriter) WriteHeader(statusCode int) { b.code = statusCode }

// splitIDFFileURL は /files/{rep}/{rel} を rep 名とそこから先に分ける。
// どちらもパーセントエスケープされたまま返す（配信ハンドラへ渡す形）。
func splitIDFFileURL(fileURL string) (repName string, escapedRel string, ok bool) {
	rest, found := strings.CutPrefix(fileURL, "/files/")
	if !found {
		return "", "", false
	}
	repName, escapedRel, found = strings.Cut(rest, "/")
	if !found || repName == "" || escapedRel == "" {
		return "", "", false
	}
	return repName, escapedRel, true
}

// readRepFileBytes は rep の配信ハンドラをメモリ上で呼んでバイト列を得る。
func readRepFileBytes(ctx context.Context, idfRep reps.IDFKyouRepository, escapedRel string, rawQuery string) ([]byte, error) {
	target := "/" + escapedRel
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("error at build file serve request %s: %w", target, err)
	}
	writer := newBufferResponseWriter()
	idfRep.HandleFileServe(writer, request)
	if writer.code != http.StatusOK {
		return nil, fmt.Errorf("error at file serve %s: status %d", target, writer.code)
	}
	return writer.body.Bytes(), nil
}

// idfRepsByName は rep 名から配信ハンドラを引けるようにする。
func idfRepsByName(ctx context.Context, repositories *reps.GkillRepositories) (map[string]reps.IDFKyouRepository, error) {
	byName := map[string]reps.IDFKyouRepository{}
	idfReps, err := repositories.IDFKyouReps.UnWrapTyped()
	if err != nil {
		return nil, fmt.Errorf("error at unwrap idf kyou reps: %w", err)
	}
	for _, idfRep := range idfReps {
		repName, err := idfRep.GetRepName(ctx)
		if err != nil {
			return nil, fmt.Errorf("error at get rep name: %w", err)
		}
		byName[repName] = idfRep
	}
	return byName, nil
}

// isExportTextFile は本文を焼き込む対象かを返す。
//
// **クライアントの判定と同じにすること**（use-idf-kyou-view.ts の is_text / is_markdown、
// markdown-to-html.ts の MARKDOWN_EXTENSIONS）。ずれると「画面には出るのに本文だけ空」になる。
func isExportTextFile(idf reps.IDFKyou) bool {
	if idf.IsImage || idf.IsVideo || idf.IsAudio || idf.IsZip {
		return false
	}
	switch strings.ToLower(strings.TrimPrefix(path.Ext(idf.TargetFile), ".")) {
	case "txt", "md", "markdown":
		return true
	}
	return false
}

// collectExportMedia はメディアと IDF のテキスト本文を焼き込む。
//
// 予算に収まらなかったものは焼かずに警告へ回す（画面は出る。その記録のメディアが出ないだけ）。
// 関連時刻の新しい順に詰めるので、payload.IDFKyous は検索の並び（新しい順）のままで渡すこと。
// sidecar が nil でなければ、メディアは data: URI ではなく同名フォルダの相対パスで参照する。
func (g *GkillServerAPI) collectExportMedia(ctx context.Context, request *req_res.ExportRykvHTMLRequest, repositories *reps.GkillRepositories, payload *exportedKyouPayload, sidecar *exportSidecar) ([]string, error) {
	warnings := []string{}
	if len(payload.IDFKyous) == 0 {
		return warnings, nil
	}

	byName, err := idfRepsByName(ctx, repositories)
	if err != nil {
		return warnings, err
	}

	mediaUsed := 0
	textUsed := 0
	droppedMedia := 0
	droppedText := 0
	failedMedia := 0

	for _, idf := range payload.IDFKyous {
		repName, escapedRel, ok := splitIDFFileURL(idf.FileURL)
		if !ok {
			continue
		}
		idfRep, exist := byName[repName]
		if !exist {
			continue
		}

		// テキスト / Markdown の本文。**バイト列のまま焼く**（文字コードの判定は
		// クライアントの detect_and_decode_text が唯一の判定者。ADR-0416）。
		// メディアを埋め込まない指定でも焼く（これが無いと本文が空になる）
		if isExportTextFile(idf) {
			if _, already := payload.IDFTextContents[idf.ID]; already {
				continue
			}
			body, err := readRepFileBytes(ctx, idfRep, escapedRel, "")
			if err != nil {
				failedMedia++
				continue
			}
			if textUsed+len(body) > exportTextBudgetBytes {
				droppedText++
				continue
			}
			textUsed += len(body)
			payload.IDFTextContents[idf.ID] = exportedIDFTextContent{
				BytesBase64: base64.StdEncoding.EncodeToString(body),
				Truncated:   false,
			}
			continue
		}

		if request.MediaMode == req_res.ExportMediaModeNone {
			continue
		}
		if !idf.IsImage && !idf.IsVideo {
			continue
		}

		// サムネイル。鍵はクライアントの build_media_url と同じ形にする
		thumbQuery := exportThumbQuery
		if idf.IsVideo {
			thumbQuery = exportVideoThumbQuery
		}
		thumbKey := idf.FileURL + "?" + thumbQuery
		if _, already := payload.Media[thumbKey]; !already {
			body, err := readRepFileBytes(ctx, idfRep, escapedRel, thumbQuery)
			if err != nil {
				// サムネイルは作れないことがある（ffmpeg が無い・デコードできない）。
				// 画面は出るので、落として警告に回す
				failedMedia++
			} else if mediaUsed+len(body) > exportMediaBudgetBytes {
				droppedMedia++
			} else {
				mediaUsed += len(body)
				payload.Media[thumbKey] = exportedMediaEntry{
					Src:  exportMediaSrc(sidecar, body, ".jpg", "image/jpeg"),
					Kind: "thumb",
				}
			}
		}

		if request.MediaMode != req_res.ExportMediaModeFull {
			continue
		}

		// 原寸。件数しだいで容易に巨大になるので、予算に当たったらそこで止める
		if _, already := payload.Media[idf.FileURL]; already {
			continue
		}
		body, err := readRepFileBytes(ctx, idfRep, escapedRel, "")
		if err != nil {
			failedMedia++
			continue
		}
		if mediaUsed+len(body) > exportMediaBudgetBytes {
			droppedMedia++
			continue
		}
		mediaUsed += len(body)
		payload.Media[idf.FileURL] = exportedMediaEntry{
			Src:  exportMediaSrc(sidecar, body, sidecarExtension(idf.TargetFile), exportMediaContentType(idf)),
			Kind: "full",
		}
	}

	if droppedMedia != 0 {
		warnings = append(warnings, fmt.Sprintf("メディアが大きすぎるため %d 件を含めませんでした", droppedMedia))
	}
	if droppedText != 0 {
		warnings = append(warnings, fmt.Sprintf("本文が大きすぎるため %d 件を含めませんでした", droppedText))
	}
	if failedMedia != 0 {
		warnings = append(warnings, fmt.Sprintf("%d 件のファイルを読み込めませんでした", failedMedia))
	}
	return warnings, nil
}

// exportMediaSrc は HTML から参照する src を作る。
//
// 単一 HTML なら data: URI、同名フォルダなら相対パス。
// **スキーマはどちらも同じ**（クライアントは src をそのまま使うだけ）。
func exportMediaSrc(sidecar *exportSidecar, body []byte, ext string, contentType string) string {
	if sidecar != nil {
		return sidecar.add(body, ext)
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(body)
}

// exportMediaContentType は data: URI に書く種別を返す。
// 拡張子から引けないものは application/octet-stream にする（ブラウザが中身で判断する）。
func exportMediaContentType(idf reps.IDFKyou) string {
	switch strings.ToLower(strings.TrimPrefix(path.Ext(idf.TargetFile), ".")) {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "bmp":
		return "image/bmp"
	case "svg":
		return "image/svg+xml"
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	case "mov":
		return "video/quicktime"
	default:
		return "application/octet-stream"
	}
}
