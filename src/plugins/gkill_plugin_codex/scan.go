package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// sessionIndexFileName は セッションuuid -> スレッド名 の対応表。
// ロールアウトとは別の無効化単位にする(名前が付くたび書き換わるので、
// 同じ扱いにすると毎回すべてのスレッドを読み直すことになる)。
const sessionIndexFileName = "session_index.jsonl"

// rolloutFileNamePattern は rollout-<日付>T<時刻>-<uuid>.jsonl。
// 末尾のuuidがスレッドID。全体を通して一意で、実データ52ファイルで重複が無いことを確認済み。
var rolloutFileNamePattern = regexp.MustCompile(`^rollout-.*-([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.jsonl$`)

// scannedFile は走査で見つけた1ファイル。
type scannedFile struct {
	Path      string
	MtimeUnix int64
	Size      int64
	Kind      string
	Meta      rolloutMeta // kindRollout のときだけ意味がある

	// 「追記のみ」の前提が破れていないかを見るための控え。file_cache から読んだときだけ入る。
	userCount     int64
	firstUserUnix int64
}

// expandPatterns は設定のパターンを実在するパスへ展開する。
func expandPatterns(patterns []string) sdk.ExpandedSource {
	return sdk.ExpandSourcePatterns(patterns)
}

// defaultSourcePatterns は source_dirs が空のときに見る場所。
func defaultSourcePatterns() []string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{
		filepath.Join(home, ".codex", "sessions"),
		filepath.Join(home, ".codex", sessionIndexFileName),
	}
}

// classifyFileName はファイル名だけで種別を決める。
//
// 中身を読んで判定しないのは、ロールアウトが最大45MBあり、
// 変わっていないファイルまで開くと走査だけで時間を使い切るため。
// ロールアウトの実体判定(session_meta があるか)は取り込み時に行う。
//
// ~/.codex には session_index.jsonl・transcription-history.jsonl・
// .codex-global-state.json が同居しているので、明示的に弾く必要がある。
func classifyFileName(filePath string) string {
	base := lastPathElement(filePath)
	if strings.EqualFold(base, sessionIndexFileName) {
		return kindIndex
	}
	if rolloutFileNamePattern.MatchString(base) {
		return kindRollout
	}
	return kindOther
}

// threadIDFromFileName はロールアウトのファイル名からスレッドIDを取り出す。
//
// これがKyouIDの土台になる。session_meta.session_id は使えない ――
// 実データ52ファイル中23ファイルに存在せず、存在してもサブエージェントでは
// 親のIDが入っているため、親子のKyouIDが衝突する。
// session_meta.id とファイル名のuuidは52/52で一致することを確認済み。
func threadIDFromFileName(filePath string) string {
	matched := rolloutFileNamePattern.FindStringSubmatch(lastPathElement(filePath))
	if matched == nil {
		return ""
	}
	return strings.ToLower(matched[1])
}

// scanSources は取り込み対象のファイルを集める。
//
// ここでは stat しかしない。中身を読むのは差分があったファイルだけ。
func scanSources(src sdk.ExpandedSource) ([]scannedFile, error) {
	found := map[string]scannedFile{}
	var walkErrs []error

	add := func(filePath string, info fs.FileInfo) {
		kind := classifyFileName(filePath)
		if kind == kindOther {
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
			Kind:      kind,
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
	// 新しい順(パス降順 ≒ 日付降順)。直近のデータが数秒で見えるようにするため
	sort.Slice(files, func(i, j int) bool { return files[i].Path > files[j].Path })

	return files, errors.Join(walkErrs...)
}

// sessionIndexEntry は session_index.jsonl の1行。
type sessionIndexEntry struct {
	ID         string `json:"id"`
	ThreadName string `json:"thread_name"`
}

// readSessionIndex は スレッドID -> スレッド名 を読む。
//
// 実データでは52セッション中33件しか載っていないので、あくまで補助。
// 読めなくても取り込みは続ける。
func readSessionIndex(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	titles := map[string]string{}
	reader := newLineReader(file)
	for {
		_, data, _, readErr := reader.next(nil)
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return titles, readErr
		}
		if len(data) == 0 {
			continue
		}
		var entry sessionIndexEntry
		if json.Unmarshal(data, &entry) != nil {
			continue
		}
		if entry.ID == "" || entry.ThreadName == "" {
			continue
		}
		titles[strings.ToLower(entry.ID)] = entry.ThreadName
	}
	return titles, nil
}
