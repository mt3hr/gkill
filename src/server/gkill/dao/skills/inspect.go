package skills

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

// revisionHexLength は revision に使う SHA-256 の hex の桁数。
// 楽観ロックの照合に使うだけなので、AI が引数に書き写しやすい長さに切る。
const revisionHexLength = 16

// RevisionOf は中身の revision（SHA-256 の hex 先頭 16 桁）を返す。
func RevisionOf(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])[:revisionHexLength]
}

// IsText は中身がテキストか（UTF-8 として正しく、NUL を含まない）を返す。拡張子では判定しない。
func IsText(content []byte) bool {
	return utf8.Valid(content) && bytes.IndexByte(content, 0) < 0
}

// inspection はファイルを読み通して得た revision とテキスト判定。
type inspection struct {
	revision string
	isText   bool
	size     int64
}

// inspectFile はファイルを流し読みして revision とテキスト判定を求める（中身は保持しない）。
func inspectFile(path string) (inspection, error) {
	file, err := os.Open(path)
	if err != nil {
		return inspection{}, fmt.Errorf("error at open skill file %s: %w", path, err)
	}
	defer file.Close()
	return inspectReader(file)
}

// inspectReader は r を読み通して revision とテキスト判定を求める。
// UTF-8 の判定はチャンク境界で文字が割れても誤らないよう、未完の末尾を次へ持ち越す。
func inspectReader(r io.Reader) (inspection, error) {
	hasher := sha256.New()
	buf := make([]byte, 64*1024)
	var carry []byte
	isText := true
	var size int64
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			hasher.Write(chunk)
			size += int64(n)
			if isText {
				data := append(carry, chunk...)
				complete, tail := splitIncompleteTail(data)
				if !utf8.Valid(complete) || bytes.IndexByte(complete, 0) >= 0 {
					isText = false
				}
				carry = append([]byte(nil), tail...)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return inspection{}, fmt.Errorf("error at read skill file: %w", err)
		}
	}
	if len(carry) != 0 {
		isText = false
	}
	return inspection{
		revision: hex.EncodeToString(hasher.Sum(nil))[:revisionHexLength],
		isText:   isText,
		size:     size,
	}, nil
}

// splitIncompleteTail は b の末尾で途中まで来ている UTF-8 の文字を切り分ける。
func splitIncompleteTail(b []byte) ([]byte, []byte) {
	for i := len(b) - 1; i >= 0 && i >= len(b)-utf8.UTFMax; i-- {
		if utf8.RuneStart(b[i]) {
			if !utf8.FullRune(b[i:]) {
				return b[:i], b[i:]
			}
			break
		}
	}
	return b, nil
}
