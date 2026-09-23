// Package skills は利用者が AI 向けに書くスキル（SKILL.md と付属ファイル）のファイルストア。
//
// 置き場所は $GKILL_HOME/skills/<user_id>/<skill-name>/ で、読み書きするのは gkill_server だけ
// （MCP も画面もこのパッケージを包んだ HTTP API を呼ぶ）。記録（rep）ではないので dao/reps の
// 4層構成には乗せない。履歴は持たない（記録アプリの本質ではないため。ADR-0634）。
//
// フォルダは実体として扱わず、ファイルのパスの一部とみなす。書けばフォルダができ、
// 中身が無くなれば消える。スキル名・パスの規則は path.go、SKILL.md の書式は frontmatter.go。
package skills

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// 利用者ディレクトリの中で、スキル名にはなりえない作業用の名前の接頭辞。
// スキル名は英数字で始まるので衝突しない。前回の中断で残ったものは次の置き換えで掃除する。
const (
	replaceTempPrefix = "_tmp-"
	replaceOldPrefix  = "_old-"
)

// Store はスキルのファイルストア。利用者ごとの RWMutex で、置き換え・書き込みと読み取りを直列化する。
type Store struct {
	root string

	locksMutex sync.Mutex
	locks      map[string]*sync.RWMutex
}

// NewStore は root（展開済みの $GKILL_HOME/skills）を根とするストアを作る。ディレクトリは書き込み時に作る。
func NewStore(root string) *Store {
	return &Store{root: root, locks: map[string]*sync.RWMutex{}}
}

// Root は根のディレクトリを返す。
func (s *Store) Root() string {
	return s.root
}

// SkillSummary は一覧の1行。
type SkillSummary struct {
	Name        string
	Description string
	UpdatedTime time.Time
	FileCount   int
	// InvalidReason は SKILL.md が無い・frontmatter が壊れている等の理由。正常なら空。
	InvalidReason string
}

// FileInfo はスキル内の1ファイルの情報。
type FileInfo struct {
	Path        string
	Size        int64
	IsText      bool
	Revision    string
	UpdatedTime time.Time
}

// Skill は1つのスキルの中身（SKILL.md の全文とファイル一覧）。
type Skill struct {
	Name          string
	Description   string
	InvalidReason string
	// Manifest は SKILL.md の全文（frontmatter を含む）。SKILL.md が無ければ nil。
	Manifest         []byte
	ManifestRevision string
	UpdatedTime      time.Time
	Files            []*FileInfo
}

// FileContent は1ファイルの中身。Omitted のときは Content が nil。
type FileContent struct {
	FileInfo
	Content []byte
	// Omitted は maxBytes を超えたので中身を返さなかった印。
	Omitted bool
}

// ReplacePlan は zip で置き換えたときに何が起きるか（起きたか）。
type ReplacePlan struct {
	Name    string
	IsNew   bool
	Added   []string
	Removed []string
	Changed []string
	Ignored []string
}

// walkedFile は走査で見つけたファイル。abs は実パス（利用者入力を含まない）。
type walkedFile struct {
	path    string
	abs     string
	size    int64
	modTime time.Time
}

// zipSource はダウンロード用 zip に入れる1ファイル。
type zipSource struct {
	path    string
	abs     string
	modTime time.Time
}

func (z zipSource) copyTo(w io.Writer) error {
	file, err := os.Open(z.abs)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(w, file)
	return err
}

func (s *Store) userLock(userID string) *sync.RWMutex {
	s.locksMutex.Lock()
	defer s.locksMutex.Unlock()
	// Windows では大文字小文字だけ違う利用者IDが同じディレクトリを指すので、錠も共有する
	key := strings.ToLower(userID)
	lock, ok := s.locks[key]
	if !ok {
		lock = &sync.RWMutex{}
		s.locks[key] = lock
	}
	return lock
}

// resolveUserDir は利用者のディレクトリと、それが既にあるかを返す。
// 利用者IDはアカウント作成時にしか形式を検査していないので、ここでパス要素として安全かを確かめる。
func (s *Store) resolveUserDir(userID string) (string, bool, error) {
	if !isSingleSafePathElement(userID) {
		return "", false, detailError(ErrInvalidUserID, fmt.Sprintf("%q", userID))
	}
	// ".." 除去。直前の検証で弾いているため実行時には常に no-op だが、
	// CodeQL の path-injection はこの形しか値サニタイザとして認識しない（dao/reps/local_rep_cache_path.go と同じ）。
	safeUserID := strings.ReplaceAll(userID, "..", "")
	exists, err := exactChildDir(s.root, safeUserID)
	if err != nil {
		return "", false, err
	}
	return filepath.Join(s.root, safeUserID), exists, nil
}

// exactChildDir は parent の直下に name というディレクトリが（大文字小文字まで一致で）あるかを返す。
// Windows では大文字小文字だけ違う名前でも Stat が通ってしまうので、列挙して名前を突き合わせる。
func exactChildDir(parent string, name string) (bool, error) {
	entries, err := os.ReadDir(parent)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error at read directory %s: %w", parent, err)
	}
	for _, entry := range entries {
		if entry.Name() == name {
			return entry.IsDir(), nil
		}
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), name) {
			return false, detailError(ErrInvalidName, fmt.Sprintf("%q conflicts with the existing directory %q (they differ only in case)", name, entry.Name()))
		}
	}
	return false, nil
}

// walkSkill はスキルのディレクトリ内の通常ファイルを列挙する（パス順）。
// ドットで始まるもの（書き込み途中の一時ファイルや利用者が置いた .git 等）とシンボリックリンクは無視する。
func walkSkill(skillDir string) ([]*walkedFile, error) {
	files := []*walkedFile{}
	err := filepath.WalkDir(skillDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == skillDir {
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return err
		}
		files = append(files, &walkedFile{path: filepath.ToSlash(rel), abs: path, size: info.Size(), modTime: info.ModTime()})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error at walk skill directory %s: %w", skillDir, err)
	}
	slices.SortFunc(files, func(a, b *walkedFile) int { return strings.Compare(a.path, b.path) })
	return files, nil
}

func findWalked(files []*walkedFile, path string) *walkedFile {
	for _, file := range files {
		if file.path == path {
			return file
		}
	}
	return nil
}

func walkedPaths(files []*walkedFile) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.path)
	}
	return paths
}

// resolveSkillDir はスキルのディレクトリを返す。無ければ ErrSkillNotFound。
func (s *Store) resolveSkillDir(userID string, name string) (string, error) {
	if err := ValidateSkillName(name); err != nil {
		return "", err
	}
	userDir, userExists, err := s.resolveUserDir(userID)
	if err != nil {
		return "", err
	}
	if !userExists {
		return "", detailError(ErrSkillNotFound, name)
	}
	exists, err := exactChildDir(userDir, name)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", detailError(ErrSkillNotFound, name)
	}
	return filepath.Join(userDir, name), nil
}

// List は利用者のスキルの一覧を名前順で返す。スキル名の規則に合わないディレクトリと、
// "_" やドットで始まるもの（予約名・作業用・.git）は出さない。
func (s *Store) List(userID string) ([]*SkillSummary, error) {
	lock := s.userLock(userID)
	lock.RLock()
	defer lock.RUnlock()

	userDir, exists, err := s.resolveUserDir(userID)
	if err != nil {
		return nil, err
	}
	summaries := []*SkillSummary{}
	if !exists {
		return summaries, nil
	}
	entries, err := os.ReadDir(userDir)
	if err != nil {
		return nil, fmt.Errorf("error at read skills directory %s: %w", userDir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || ValidateSkillName(entry.Name()) != nil {
			continue
		}
		summaries = append(summaries, summarizeSkill(filepath.Join(userDir, entry.Name()), entry.Name()))
	}
	slices.SortFunc(summaries, func(a, b *SkillSummary) int { return strings.Compare(a.Name, b.Name) })
	return summaries, nil
}

// summarizeSkill は一覧の1行を作る。読めない・壊れている場合も行は返し、理由を InvalidReason に入れる
// （一覧から消すと、利用者が壊れたスキルに気づけず直せない）。
func summarizeSkill(skillDir string, name string) *SkillSummary {
	summary := &SkillSummary{Name: name}
	if info, err := os.Stat(skillDir); err == nil {
		summary.UpdatedTime = info.ModTime()
	}
	files, err := walkSkill(skillDir)
	if err != nil {
		summary.InvalidReason = err.Error()
		return summary
	}
	summary.FileCount = len(files)
	for _, file := range files {
		if file.modTime.After(summary.UpdatedTime) {
			summary.UpdatedTime = file.modTime
		}
	}
	manifestFile := findWalked(files, ManifestFileName)
	if manifestFile == nil {
		summary.InvalidReason = ManifestFileName + " is missing"
		return summary
	}
	content, err := os.ReadFile(manifestFile.abs)
	if err != nil {
		summary.InvalidReason = fmt.Sprintf("%s could not be read: %v", ManifestFileName, err)
		return summary
	}
	manifest, err := ParseManifest(content, name)
	if err != nil {
		summary.InvalidReason = err.Error()
		return summary
	}
	summary.Description = manifest.Description
	return summary
}

// Get はスキルの SKILL.md 全文とファイル一覧（revision・テキスト判定つき）を返す。
func (s *Store) Get(userID string, name string) (*Skill, error) {
	lock := s.userLock(userID)
	lock.RLock()
	defer lock.RUnlock()

	skillDir, err := s.resolveSkillDir(userID, name)
	if err != nil {
		return nil, err
	}
	summary := summarizeSkill(skillDir, name)
	skill := &Skill{
		Name:          name,
		Description:   summary.Description,
		InvalidReason: summary.InvalidReason,
		UpdatedTime:   summary.UpdatedTime,
		Files:         []*FileInfo{},
	}
	files, err := walkSkill(skillDir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		inspected, err := inspectFile(file.abs)
		if err != nil {
			return nil, err
		}
		skill.Files = append(skill.Files, &FileInfo{
			Path:        file.path,
			Size:        inspected.size,
			IsText:      inspected.isText,
			Revision:    inspected.revision,
			UpdatedTime: file.modTime,
		})
		if file.path == ManifestFileName {
			content, err := os.ReadFile(file.abs)
			if err != nil {
				return nil, fmt.Errorf("error at read %s: %w", file.abs, err)
			}
			skill.Manifest = content
			skill.ManifestRevision = RevisionOf(content)
		}
	}
	return skill, nil
}

// ReadFile はスキル内の1ファイルを返す。maxBytes が正でそれを超えるときは中身を省く（Omitted）。
// 保存量には上限を設けないが、AI へ返すときだけ大きすぎるファイルを渡さないために使う。
func (s *Store) ReadFile(userID string, name string, path string, maxBytes int64) (*FileContent, error) {
	path, err := NormalizeFilePath(path)
	if err != nil {
		return nil, err
	}
	lock := s.userLock(userID)
	lock.RLock()
	defer lock.RUnlock()

	skillDir, err := s.resolveSkillDir(userID, name)
	if err != nil {
		return nil, err
	}
	files, err := walkSkill(skillDir)
	if err != nil {
		return nil, err
	}
	// 走査で見つかったものだけを読む（大文字小文字違いやシンボリックリンクを辿らない）
	file := findWalked(files, path)
	if file == nil {
		return nil, detailError(ErrFileNotFound, path)
	}
	result := &FileContent{FileInfo: FileInfo{Path: file.path, UpdatedTime: file.modTime}}
	if maxBytes > 0 && file.size > maxBytes {
		inspected, err := inspectFile(file.abs)
		if err != nil {
			return nil, err
		}
		result.Size = inspected.size
		result.IsText = inspected.isText
		result.Revision = inspected.revision
		result.Omitted = true
		return result, nil
	}
	content, err := os.ReadFile(file.abs)
	if err != nil {
		return nil, fmt.Errorf("error at read skill file %s: %w", file.abs, err)
	}
	result.Size = int64(len(content))
	result.IsText = IsText(content)
	result.Revision = RevisionOf(content)
	result.Content = content
	return result, nil
}

// WriteFile はスキル内の1ファイルを書く（AI の書き込み用）。新しい revision を返す。
//
// revision が空なら新規作成だけを許す（既にあれば ErrAlreadyExists）。空でなければ
// 今の中身の revision と一致したときだけ上書きする（食い違えば ErrRevisionConflict）。
// スキルが無いときは SKILL.md の新規作成だけがスキルを作れる。SKILL.md は frontmatter を検査する。
func (s *Store) WriteFile(ctx context.Context, userID string, name string, path string, content []byte, revision string) (string, error) {
	if err := ValidateSkillName(name); err != nil {
		return "", err
	}
	path, err := NormalizeFilePath(path)
	if err != nil {
		return "", err
	}
	if path == ManifestFileName {
		if _, err := ParseManifest(content, name); err != nil {
			return "", err
		}
	}

	lock := s.userLock(userID)
	lock.Lock()
	defer lock.Unlock()

	userDir, _, err := s.resolveUserDir(userID)
	if err != nil {
		return "", err
	}
	skillExists, err := exactChildDir(userDir, name)
	if err != nil {
		return "", err
	}
	skillDir := filepath.Join(userDir, name)
	files := []*walkedFile{}
	if skillExists {
		files, err = walkSkill(skillDir)
		if err != nil {
			return "", err
		}
	} else if path != ManifestFileName || revision != "" {
		return "", detailError(ErrSkillNotFound, name)
	}

	paths := walkedPaths(files)
	if conflict := findCaseConflict(paths, path); conflict != "" {
		return "", detailError(ErrInvalidPath, "differs only in case from the existing file "+conflict, path)
	}
	if conflict := findFileDirConflict(paths, path); conflict != "" {
		return "", detailError(ErrInvalidPath, "conflicts with the existing file "+conflict+" (a file and a folder with the same name)", path)
	}
	existing := findWalked(files, path)
	if revision == "" && existing != nil {
		return "", detailError(ErrAlreadyExists, path)
	}
	if revision != "" {
		if existing == nil {
			return "", detailError(ErrFileNotFound, path)
		}
		inspected, err := inspectFile(existing.abs)
		if err != nil {
			return "", err
		}
		if inspected.revision != revision {
			return "", detailError(ErrRevisionConflict, fmt.Sprintf("%s: the current revision is %s (re-read the file before writing)", path, inspected.revision))
		}
	}

	target, err := joinWithin(skillDir, path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(target), fs.ModePerm); err != nil {
		return "", fmt.Errorf("error at create skill directory %s: %w", filepath.Dir(target), err)
	}
	if err := writeFileAtomic(ctx, target, content); err != nil {
		return "", err
	}
	return RevisionOf(content), nil
}

// writeFileAtomic は同じディレクトリの一時ファイルへ書いてから rename する（途中で落ちても半端な中身を残さない）。
// 一時ファイルはドットで始まるので、走査には現れない。
func writeFileAtomic(ctx context.Context, target string, content []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(target), ".tmp-*")
	if err != nil {
		return fmt.Errorf("error at create temp file for %s: %w", target, err)
	}
	tmpName := tmp.Name()
	_, writeErr := tmp.Write(content)
	closeErr := tmp.Close()
	if writeErr != nil || closeErr != nil {
		removeQuietly(ctx, tmpName)
		return fmt.Errorf("error at write temp file for %s: %w", target, errors.Join(writeErr, closeErr))
	}
	if err := os.Rename(tmpName, target); err != nil {
		removeQuietly(ctx, tmpName)
		return fmt.Errorf("error at rename temp file to %s: %w", target, err)
	}
	return nil
}

// removeQuietly は後始末の削除。失敗しても本来のエラーを優先して返すので、ログだけ残す。
func removeQuietly(ctx context.Context, path string) {
	if err := os.RemoveAll(path); err != nil {
		slog.Log(ctx, gkill_log.Warn, "error at remove leftover skill path", "path", path, "error", fmt.Sprintf("%q", err))
	}
}

// DeleteFile はスキル内の1ファイルを消す。SKILL.md は消せない（ErrManifestDelete）。
// revision が空でなければ一致したときだけ消す。空になったフォルダも消す。
func (s *Store) DeleteFile(ctx context.Context, userID string, name string, path string, revision string) error {
	path, err := NormalizeFilePath(path)
	if err != nil {
		return err
	}
	if path == ManifestFileName {
		return detailError(ErrManifestDelete, path)
	}
	lock := s.userLock(userID)
	lock.Lock()
	defer lock.Unlock()

	skillDir, err := s.resolveSkillDir(userID, name)
	if err != nil {
		return err
	}
	files, err := walkSkill(skillDir)
	if err != nil {
		return err
	}
	existing := findWalked(files, path)
	if existing == nil {
		return detailError(ErrFileNotFound, path)
	}
	if revision != "" {
		inspected, err := inspectFile(existing.abs)
		if err != nil {
			return err
		}
		if inspected.revision != revision {
			return detailError(ErrRevisionConflict, fmt.Sprintf("%s: the current revision is %s (re-read the file before deleting)", path, inspected.revision))
		}
	}
	if err := os.Remove(existing.abs); err != nil {
		return fmt.Errorf("error at remove skill file %s: %w", existing.abs, err)
	}
	removeEmptyParents(filepath.Dir(existing.abs), skillDir)
	return nil
}

// removeEmptyParents は dir から skillDir の手前まで、空になったフォルダを消す。
func removeEmptyParents(dir string, skillDir string) {
	for dir != skillDir && strings.HasPrefix(dir, skillDir) {
		if os.Remove(dir) != nil {
			// 空でない（または消せない）ならそこで止める
			return
		}
		dir = filepath.Dir(dir)
	}
}

// DeleteSkill はスキルを丸ごと消す。一覧に半端な状態が見えないよう、先に作業用の名前へ退避してから消す。
func (s *Store) DeleteSkill(ctx context.Context, userID string, name string) error {
	lock := s.userLock(userID)
	lock.Lock()
	defer lock.Unlock()

	skillDir, err := s.resolveSkillDir(userID, name)
	if err != nil {
		return err
	}
	old := filepath.Join(filepath.Dir(skillDir), replaceOldPrefix+randomSuffix())
	if err := os.Rename(skillDir, old); err != nil {
		return fmt.Errorf("error at move skill directory %s aside: %w", skillDir, err)
	}
	removeQuietly(ctx, old)
	return nil
}

// BuildZip はスキルを「スキル名のフォルダ1段で包んだ zip」にして返す（ファイル名、中身）。
func (s *Store) BuildZip(userID string, name string) (string, []byte, error) {
	lock := s.userLock(userID)
	lock.RLock()
	defer lock.RUnlock()

	skillDir, err := s.resolveSkillDir(userID, name)
	if err != nil {
		return "", nil, err
	}
	files, err := walkSkill(skillDir)
	if err != nil {
		return "", nil, err
	}
	sources := make([]zipSource, 0, len(files))
	for _, file := range files {
		sources = append(sources, zipSource{path: file.path, abs: file.abs, modTime: file.modTime})
	}
	data, err := writeSkillZip(name, sources)
	if err != nil {
		return "", nil, err
	}
	return name + ".zip", data, nil
}

// PlanReplace は zip で置き換えたら何が起きるかを返す（書き込みはしない）。画面の確認の1段目。
func (s *Store) PlanReplace(userID string, zipBytes []byte) (*ReplacePlan, error) {
	uploaded, err := parseUploadedZip(zipBytes)
	if err != nil {
		return nil, err
	}
	lock := s.userLock(userID)
	lock.RLock()
	defer lock.RUnlock()
	return s.planLocked(userID, uploaded)
}

func (s *Store) planLocked(userID string, uploaded *uploadedSkill) (*ReplacePlan, error) {
	userDir, _, err := s.resolveUserDir(userID)
	if err != nil {
		return nil, err
	}
	exists, err := exactChildDir(userDir, uploaded.name)
	if err != nil {
		return nil, err
	}
	plan := &ReplacePlan{
		Name:    uploaded.name,
		IsNew:   !exists,
		Added:   []string{},
		Removed: []string{},
		Changed: []string{},
		Ignored: append([]string{}, uploaded.ignored...),
	}
	existingRevisions := map[string]string{}
	if exists {
		files, err := walkSkill(filepath.Join(userDir, uploaded.name))
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			inspected, err := inspectFile(file.abs)
			if err != nil {
				return nil, err
			}
			existingRevisions[file.path] = inspected.revision
		}
	}
	uploadedPaths := map[string]bool{}
	for _, file := range uploaded.files {
		uploadedPaths[file.path] = true
		revision, ok := existingRevisions[file.path]
		switch {
		case !ok:
			plan.Added = append(plan.Added, file.path)
		case revision != file.revision:
			plan.Changed = append(plan.Changed, file.path)
		}
	}
	for path := range existingRevisions {
		if !uploadedPaths[path] {
			plan.Removed = append(plan.Removed, path)
		}
	}
	slices.Sort(plan.Removed)
	return plan, nil
}

// Replace は zip の中身でスキルを丸ごと置き換える（DELETE_WRITE）。新規なら作る。
//
// 一時ディレクトリへ展開してから、既存を作業用の名前へ退避 → 新しいものを正式な名前へ rename → 退避を削除、
// の順で入れ替える。途中で失敗したら元に戻し、既存のスキルには手を付けない（Windows で誰かがファイルを
// 開いていると rename が失敗する）。スキルのディレクトリに利用者が置いたドットファイル（.git 等）も消える。
func (s *Store) Replace(ctx context.Context, userID string, zipBytes []byte) (*ReplacePlan, error) {
	uploaded, err := parseUploadedZip(zipBytes)
	if err != nil {
		return nil, err
	}
	lock := s.userLock(userID)
	lock.Lock()
	defer lock.Unlock()

	plan, err := s.planLocked(userID, uploaded)
	if err != nil {
		return nil, err
	}
	userDir, _, err := s.resolveUserDir(userID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(userDir, fs.ModePerm); err != nil {
		return nil, fmt.Errorf("error at create skills directory %s: %w", userDir, err)
	}
	cleanupLeftovers(ctx, userDir)

	tmpDir, err := os.MkdirTemp(userDir, replaceTempPrefix)
	if err != nil {
		return nil, fmt.Errorf("error at create temp directory in %s: %w", userDir, err)
	}
	if err := extractUploaded(uploaded, tmpDir); err != nil {
		removeQuietly(ctx, tmpDir)
		return nil, err
	}

	skillDir := filepath.Join(userDir, uploaded.name)
	if plan.IsNew {
		if err := os.Rename(tmpDir, skillDir); err != nil {
			removeQuietly(ctx, tmpDir)
			return nil, fmt.Errorf("error at move uploaded skill into %s: %w", skillDir, err)
		}
		return plan, nil
	}
	old := filepath.Join(userDir, replaceOldPrefix+randomSuffix())
	if err := os.Rename(skillDir, old); err != nil {
		removeQuietly(ctx, tmpDir)
		return nil, fmt.Errorf("error at move skill directory %s aside: %w", skillDir, err)
	}
	if err := os.Rename(tmpDir, skillDir); err != nil {
		if restoreErr := os.Rename(old, skillDir); restoreErr != nil {
			slog.Log(ctx, gkill_log.Error, "error at restore skill directory after failed replace", "skill_dir", skillDir, "moved_to", old, "error", fmt.Sprintf("%q", restoreErr))
		}
		removeQuietly(ctx, tmpDir)
		return nil, fmt.Errorf("error at move uploaded skill into %s: %w", skillDir, err)
	}
	removeQuietly(ctx, old)
	return plan, nil
}

// extractUploaded は検証済みの zip の中身を dir へ書き出す。
func extractUploaded(uploaded *uploadedSkill, dir string) error {
	for _, file := range uploaded.files {
		target, err := joinWithin(dir, file.path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), fs.ModePerm); err != nil {
			return fmt.Errorf("error at create directory %s: %w", filepath.Dir(target), err)
		}
		if err := extractZipEntry(file, target); err != nil {
			return err
		}
	}
	return nil
}

func extractZipEntry(file *uploadedFile, target string) error {
	reader, err := file.zipEntry.Open()
	if err != nil {
		return fmt.Errorf("error at open zip entry %s: %w", file.path, err)
	}
	defer reader.Close()
	out, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("error at create %s: %w", target, err)
	}
	_, copyErr := io.Copy(out, reader)
	closeErr := out.Close()
	if copyErr != nil || closeErr != nil {
		return fmt.Errorf("error at extract zip entry %s: %w", file.path, errors.Join(copyErr, closeErr))
	}
	return nil
}

// cleanupLeftovers は前回の置き換え・削除の中断で残った作業用ディレクトリを消す。
func cleanupLeftovers(ctx context.Context, userDir string) {
	entries, err := os.ReadDir(userDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() && (strings.HasPrefix(entry.Name(), replaceTempPrefix) || strings.HasPrefix(entry.Name(), replaceOldPrefix)) {
			removeQuietly(ctx, filepath.Join(userDir, entry.Name()))
		}
	}
}

func randomSuffix() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
