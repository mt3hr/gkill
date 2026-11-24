package main

import (
	"cmp"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 読むファイルの名前。種別はファイル名だけで決める。
const (
	// transcriptFileName は会話の本文。brain/<会話ID>/.system_generated/logs/ にある。
	transcriptFileName = "transcript_full.jsonl"
	// summariesFileName は会話タイトルとワークスペースの対応表。任意。
	summariesFileName = "conversation_summaries.db"
	// truncatedTranscriptFileName は transcript_full.jsonl の省略版。
	// 同じ行数で content が切られているだけなので、読むと会話が二重になる。読まない。
	truncatedTranscriptFileName = "transcript.jsonl"
)

// conversationIDPattern は会話IDの形(UUID)。
//
// 会話IDは transcript_full.jsonl のどのレコードにも入っていないので、
// フォルダ名から採るしかない。UUID の形に限ることで、
// 同期先の `Antigravity_<端末>_<日付>` のようなフォルダ名を誤って会話IDにしない。
var conversationIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// scannedFile は走査で見つけた1ファイル。
type scannedFile struct {
	Path           string
	MtimeUnix      int64
	Size           int64
	Kind           string
	ConversationID string // kindTranscript のときだけ入る
}

// scanProblems は設定画面に出す走査の問題。取り込み自体は止めない。
type scanProblems struct {
	// MissingID は会話IDのフォルダに入っていないため読まなかった transcript_full.jsonl。
	MissingID []string
	// Shadowed は同じ会話が複数の場所にあり、小さいほうなので読まなかった件数。
	Shadowed int
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
		filepath.Join(home, ".gemini", "antigravity", "brain"),
		filepath.Join(home, ".gemini", "antigravity", summariesFileName),
	}
}

// defaultSourcePatternsForConfig は config.json に書き出す既定値。
// pluginDir にも実行アカウントのホームにも依存しない ~ 表記にする
// (--gkill-print-config から同じものを出すため)。
func defaultSourcePatternsForConfig() []string {
	return []string{
		"~/.gemini/antigravity/brain",
		"~/.gemini/antigravity/" + summariesFileName,
	}
}

// classifyFileName はファイル名だけで種別を決める。
//
// 中身を読んで判定しないのは、ログが数百KBに達することがあり、
// 変わっていないファイルまで開くと走査だけで時間を使うため。
//
// Antigravity は同じ会話の本文を3つの形で書く。読むのは1つだけにすること。
//   - transcript_full.jsonl        … これを読む
//   - transcript.jsonl            … 省略版。読むと二重になる
//   - logs/chunks/**/<8桁連番>.jsonl … 100KB ごとの写し。読むと二重になる
func classifyFileName(filePath string) string {
	base := filepath.Base(filePath)
	if strings.EqualFold(base, transcriptFileName) {
		return kindTranscript
	}
	if strings.EqualFold(base, summariesFileName) {
		return kindSummaries
	}
	return kindOther
}

// conversationIDOf はファイルから見て直近の、UUID の形をした祖先フォルダ名を返す。
//
// これがKyouIDの土台になる。1つの規則で2つの置き方の両方に効く。
//
//	生の置き場 : <ホーム>/.gemini/antigravity/brain/<会話ID>/.system_generated/logs/transcript_full.jsonl
//	同期先     : <ホーム>/Kyou/Antigravity_<端末>_<日付>/<会話ID>/transcript_full.jsonl
//
// 見つからなければ空を返す。呼び出し側はそのファイルを読まずに設定画面の警告へ出すこと
// (パスから作った代用IDを振ると、名前が変わっただけで別の会話として二重に出る)。
func conversationIDOf(filePath string) string {
	dir := filepath.Dir(filePath)
	for {
		base := filepath.Base(dir)
		if conversationIDPattern.MatchString(base) {
			return strings.ToLower(base)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// scanSources は取り込み対象のファイルを集める。
//
// ここでは stat しかしない。中身を読むのは差分があったファイルだけ。
// 同じ会話が複数の場所で見つかったら、一番大きいファイルだけを残す。
// ログは追記のみなので一番大きいものが最新で、生の置き場と同期先の両方を
// 指定したときに同じ会話が二重に出るのを防げる。
func scanSources(src sdk.ExpandedSource) ([]scannedFile, scanProblems, error) {
	found := map[string]scannedFile{}
	problems := scanProblems{}
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
		file := scannedFile{
			Path:      absolute,
			MtimeUnix: info.ModTime().Unix(),
			Size:      info.Size(),
			Kind:      kind,
		}
		if kind == kindTranscript {
			file.ConversationID = conversationIDOf(absolute)
			if file.ConversationID == "" {
				problems.MissingID = append(problems.MissingID, absolute)
				return
			}
		}
		found[absolute] = file
	}

	for _, dir := range src.Dirs {
		err := filepath.WalkDir(dir, func(walkPath string, entry fs.DirEntry, err error) error {
			if err != nil {
				// 途中の1フォルダが読めなくても走査は続ける
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
	files, shadowed := pickLargestPerConversation(files)
	problems.Shadowed = shadowed

	// 新しい順。直近の会話が数秒で見えるようにするため
	slices.SortFunc(files, func(a, b scannedFile) int {
		if c := cmp.Compare(b.MtimeUnix, a.MtimeUnix); c != 0 {
			return c
		}
		return cmp.Compare(b.Path, a.Path)
	})

	return files, problems, errors.Join(walkErrs...)
}

// pickLargestPerConversation は同じ会話IDの transcript が複数あれば一番大きいものだけを残す。
// 大きさが同じならパスの小さいほうを採る(走査順に依らず同じ結果にするため)。
func pickLargestPerConversation(files []scannedFile) ([]scannedFile, int) {
	best := map[string]scannedFile{}
	others := make([]scannedFile, 0, len(files))
	shadowed := 0
	for _, file := range files {
		if file.Kind != kindTranscript {
			others = append(others, file)
			continue
		}
		previous, exist := best[file.ConversationID]
		if !exist {
			best[file.ConversationID] = file
			continue
		}
		shadowed++
		if file.Size > previous.Size || (file.Size == previous.Size && file.Path < previous.Path) {
			best[file.ConversationID] = file
		}
	}
	for _, file := range best {
		others = append(others, file)
	}
	return others, shadowed
}
