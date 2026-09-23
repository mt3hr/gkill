package skills

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"
)

// ignoredZipBaseNames は zip に紛れ込みやすい OS の管理ファイル。取り込まずに ignored として返す。
var ignoredZipBaseNames = map[string]bool{
	"Thumbs.db":   true,
	"desktop.ini": true,
}

// uploadedFile はアップロードされた zip の中の1ファイル（スキル内パスへ正規化済み）。
type uploadedFile struct {
	path     string
	zipEntry *zip.File
	revision string
}

// uploadedSkill はアップロードされた zip を検証して読み取った結果。
type uploadedSkill struct {
	name    string
	files   []*uploadedFile
	ignored []string
}

// parseUploadedZip は zip の中身をスキルとして検証する。書き込みはしない。
//
// 中身がフォルダ1段で包まれている（ダウンロードした zip をそのまま上げ直した等）なら、その1段を剥がす。
// OS の管理ファイルやドットで始まるものは無視し、規則に合わない名前・絶対パス・".."・
// シンボリックリンクは拒否する。展開後のサイズに上限は設けない（個人利用の方針。ADR-0634）。
func parseUploadedZip(zipBytes []byte) (*uploadedSkill, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, detailError(ErrInvalidZip, fmt.Sprintf("the file could not be read as a zip archive: %v", err))
	}

	type entry struct {
		segments []string
		zipEntry *zip.File
	}
	entries := []entry{}
	ignored := []string{}
	rejected := []string{}
	for _, zipEntry := range reader.File {
		name := strings.ReplaceAll(zipEntry.Name, `\`, "/")
		if strings.HasSuffix(name, "/") {
			// フォルダの項目。フォルダは実体を持たないので読み飛ばす
			continue
		}
		if zipEntry.Mode()&fs.ModeSymlink != 0 {
			rejected = append(rejected, zipEntry.Name)
			continue
		}
		if strings.HasPrefix(name, "/") || (len(name) >= 2 && name[1] == ':') {
			rejected = append(rejected, zipEntry.Name)
			continue
		}
		segments := strings.Split(name, "/")
		isIgnored := false
		hasParentRef := false
		for _, segment := range segments {
			if segment == ".." {
				hasParentRef = true
			}
			if strings.HasPrefix(segment, ".") || segment == "__MACOSX" {
				isIgnored = true
			}
		}
		if hasParentRef {
			rejected = append(rejected, zipEntry.Name)
			continue
		}
		if isIgnored || ignoredZipBaseNames[segments[len(segments)-1]] {
			ignored = append(ignored, name)
			continue
		}
		entries = append(entries, entry{segments: segments, zipEntry: zipEntry})
	}
	if len(rejected) != 0 {
		return nil, detailError(ErrInvalidZip, "absolute paths, '..' and symbolic links are not allowed", rejected...)
	}
	if len(entries) == 0 {
		return nil, detailError(ErrInvalidZip, "the zip has no files")
	}

	// 包んでいるフォルダ1段を剥がす。ルートに SKILL.md があれば剥がさない
	hasRootManifest := slices.ContainsFunc(entries, func(e entry) bool {
		return len(e.segments) == 1 && e.segments[0] == ManifestFileName
	})
	if !hasRootManifest {
		wrapper := entries[0].segments[0]
		allWrapped := true
		for _, e := range entries {
			if len(e.segments) < 2 || e.segments[0] != wrapper {
				allWrapped = false
				break
			}
		}
		if allWrapped {
			for i := range entries {
				entries[i].segments = entries[i].segments[1:]
			}
		}
	}

	skill := &uploadedSkill{ignored: ignored}
	paths := []string{}
	invalid := []string{}
	for _, e := range entries {
		path := strings.Join(e.segments, "/")
		if _, err := NormalizeFilePath(path); err != nil {
			invalid = append(invalid, path)
			continue
		}
		if conflict := findCaseConflict(paths, path); conflict != "" {
			invalid = append(invalid, path)
			continue
		}
		if conflict := findFileDirConflict(paths, path); conflict != "" {
			invalid = append(invalid, path)
			continue
		}
		paths = append(paths, path)
		skill.files = append(skill.files, &uploadedFile{path: path, zipEntry: e.zipEntry})
	}
	if len(invalid) != 0 {
		return nil, detailError(ErrInvalidZip, "file names must use only letters, digits, '.', '_' and '-', must not start with '.', and must not differ only in case", invalid...)
	}

	var manifestFile *uploadedFile
	for _, f := range skill.files {
		if f.path == ManifestFileName {
			manifestFile = f
		}
	}
	if manifestFile == nil {
		return nil, detailError(ErrInvalidZip, "SKILL.md is missing at the top level of the zip")
	}

	for _, f := range skill.files {
		if f == manifestFile {
			content, err := readZipEntry(f.zipEntry)
			if err != nil {
				return nil, detailError(ErrInvalidZip, fmt.Sprintf("a file could not be read: %v", err), f.path)
			}
			manifest, err := ParseManifest(content, "")
			if err != nil {
				return nil, err
			}
			skill.name = manifest.Name
			f.revision = RevisionOf(content)
			continue
		}
		// SKILL.md 以外は中身を保持せずに流し読みで revision だけ求める（大きな添付でメモリを倍にしない）
		inspected, err := inspectZipEntry(f.zipEntry)
		if err != nil {
			return nil, detailError(ErrInvalidZip, fmt.Sprintf("a file could not be read: %v", err), f.path)
		}
		f.revision = inspected.revision
	}
	slices.SortFunc(skill.files, func(a, b *uploadedFile) int { return strings.Compare(a.path, b.path) })
	slices.Sort(skill.ignored)
	return skill, nil
}

// readZipEntry は zip の1項目を読み出す。宣言サイズや CRC の食い違いは archive/zip がエラーにする。
func readZipEntry(zipEntry *zip.File) ([]byte, error) {
	reader, err := zipEntry.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// inspectZipEntry は zip の1項目を流し読みして revision とテキスト判定を求める。
func inspectZipEntry(zipEntry *zip.File) (inspection, error) {
	reader, err := zipEntry.Open()
	if err != nil {
		return inspection{}, err
	}
	defer reader.Close()
	return inspectReader(reader)
}

// writeSkillZip は files（スキル内パス → 中身の読み出し関数）を、スキル名のフォルダ1段で包んだ zip にする。
// アップロードはこの1段を剥がすので、ダウンロードした zip をそのまま上げ直せる。
func writeSkillZip(name string, files []zipSource) ([]byte, error) {
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, file := range files {
		header := &zip.FileHeader{
			Name:     name + "/" + file.path,
			Method:   zip.Deflate,
			Modified: file.modTime,
		}
		entryWriter, err := writer.CreateHeader(header)
		if err != nil {
			return nil, fmt.Errorf("error at create zip entry %s: %w", file.path, err)
		}
		if err := file.copyTo(entryWriter); err != nil {
			return nil, fmt.Errorf("error at write zip entry %s: %w", file.path, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("error at close skill zip: %w", err)
	}
	return buf.Bytes(), nil
}
