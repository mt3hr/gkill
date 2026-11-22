package gkill_server_api

// 「HTML ＋ 同名フォルダ」の出力。HTML 1枚とフォルダをまとめて ZIP で返す。
//
// フォルダの中のファイル名は**内容のハッシュ＋元の拡張子**にする。元のファイル名を使うと
// 4つ問題がある: Windows の MAX_PATH、使えない文字と予約名、ZIP の中の日本語ファイル名の
// 文字化け、そして**実ファイル名そのものが個人情報になりうる**こと。
// ハッシュ名なら ASCII の英数字と `.` だけになって4つとも消え、内容ハッシュなので
// 同じ画像が複数の記録から参照されても1エントリで済む。
//
// 表示名は失われない。元のファイル名は IDFKyou.file_name として焼き込みの JSON の中にある。

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"slices"
	"strings"
)

// exportSidecar は同名フォルダに入れるファイルを集める。
type exportSidecar struct {
	// dirName は HTML と同じ名前のフォルダ（拡張子なし）。
	dirName string
	// files は公開パス（dirName/xxxx.jpg）→ バイト列。
	files map[string][]byte
}

func newExportSidecar(baseName string) *exportSidecar {
	return &exportSidecar{dirName: baseName, files: map[string][]byte{}}
}

// add はバイト列を1件入れて、HTML から参照する相対パスを返す。
// 同じ内容を2回入れても1エントリにしかならない。
func (s *exportSidecar) add(body []byte, ext string) string {
	sum := sha256.Sum256(body)
	name := hex.EncodeToString(sum[:8]) + ext
	publishedPath := s.dirName + "/" + name
	s.files[publishedPath] = body
	return publishedPath
}

// sidecarExtension は元のファイル名から、フォルダの中で使う拡張子を決める。
//
// 英数字だけの短いものに限る。外れたら .bin にする（ZIP に入れられない名前を作らない）。
func sidecarExtension(fileName string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(fileName), "."))
	if ext == "" || len(ext) > 10 {
		return ".bin"
	}
	for _, r := range ext {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') {
			return ".bin"
		}
	}
	return "." + ext
}

// buildExportZip は HTML 1枚と同名フォルダを ZIP にまとめる。
//
// エントリの順は名前順に固定する（同じ入力なら同じバイト列になり、出力を比べられる）。
func buildExportZip(baseName string, html string, sidecar *exportSidecar) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)

	htmlEntry, err := writer.Create(baseName + ".html")
	if err != nil {
		return nil, fmt.Errorf("error at create zip entry %s.html: %w", baseName, err)
	}
	if _, err := htmlEntry.Write([]byte(html)); err != nil {
		return nil, fmt.Errorf("error at write zip entry %s.html: %w", baseName, err)
	}

	paths := make([]string, 0, len(sidecar.files))
	for publishedPath := range sidecar.files {
		paths = append(paths, publishedPath)
	}
	slices.Sort(paths)

	for _, publishedPath := range paths {
		entry, err := writer.Create(publishedPath)
		if err != nil {
			return nil, fmt.Errorf("error at create zip entry %s: %w", publishedPath, err)
		}
		if _, err := entry.Write(sidecar.files[publishedPath]); err != nil {
			return nil, fmt.Errorf("error at write zip entry %s: %w", publishedPath, err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("error at close zip writer: %w", err)
	}
	return buffer.Bytes(), nil
}
