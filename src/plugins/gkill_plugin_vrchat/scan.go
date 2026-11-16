package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// logFileNamePattern は VRChat の出力ログのファイル名。output_log_<起動日時>.txt
var logFileNamePattern = regexp.MustCompile(`^output_log_.*\.txt$`)

// scannedFile は走査で見つけた1ファイル。
type scannedFile struct {
	Path      string
	MtimeUnix int64
	Size      int64
	// SessionKey は file_cache から読んだときだけ入る（取り込み時に決まる）。
	SessionKey string
}

// expandPatterns は設定のパターンを実在するパスへ展開する。
func expandPatterns(patterns []string) sdk.ExpandedSource {
	return sdk.ExpandSourcePatterns(patterns)
}

// defaultSourcePatterns は source_dirs が空のときに見る場所（VRChat の既定の書き出し先）。
func defaultSourcePatterns() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{filepath.Join(home, "AppData", "LocalLow", "VRChat", "VRChat")}
}

// isLogFileName はファイル名だけで出力ログかを決める。
// 同じフォルダには Player.log やスクリーンショットなどが同居している。
func isLogFileName(filePath string) bool {
	return logFileNamePattern.MatchString(filepath.Base(filePath))
}

// scanSources は取り込み対象のファイルを集める。
//
// ここでは stat しかしない。中身を読むのは差分があったファイルだけ。
func scanSources(src sdk.ExpandedSource) ([]scannedFile, error) {
	found := map[string]scannedFile{}
	var walkErrs []error

	add := func(filePath string, info fs.FileInfo) {
		if !isLogFileName(filePath) {
			return
		}
		absolute, err := filepath.Abs(filePath)
		if err != nil {
			absolute = filePath
		}
		found[absolute] = scannedFile{
			Path:      absolute,
			MtimeUnix: info.ModTime().Unix(),
			Size:      info.Size(),
		}
	}

	for _, dir := range src.Dirs {
		err := filepath.WalkDir(dir, func(walkPath string, entry fs.DirEntry, err error) error {
			if err != nil {
				// 途中の1ディレクトリが読めなくても走査は続ける
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return nil
			}
			add(walkPath, info)
			return nil
		})
		if err != nil {
			walkErrs = append(walkErrs, err)
		}
	}

	for _, filePath := range src.Files {
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}
		add(filePath, info)
	}

	files := make([]scannedFile, 0, len(found))
	for _, file := range found {
		files = append(files, file)
	}
	// 新しい順(ファイル名の起動日時の降順)。直近のデータが先に見えるようにするため
	sort.Slice(files, func(i, j int) bool {
		bi, bj := filepath.Base(files[i].Path), filepath.Base(files[j].Path)
		if bi != bj {
			return bi > bj
		}
		return files[i].Path > files[j].Path
	})

	return files, errors.Join(walkErrs...)
}
