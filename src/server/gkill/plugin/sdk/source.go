package sdk

// source.go は Google Takeout のようなZIPで届く書き出しを走査する。
//
// **ZIPしか読まない。展開済みのフォルダは読まない。**
// そしてZIPを展開もしない。archive/zip のストリームからそのまま読む。
//
// gkill本体の handle_browse_zip_contents.go はZIPを caches/zip_cache/ へ丸ごと展開するが、
// あれは画面でZIPの中身を辿るための仕組みで、ここでは真似しない。
//   - Takeout は展開後 3.73GB あり、ディスクに二重に置くことになる
//   - あちらのキャッシュのキーは sha1(パス) だけなので、中身を差し替えても古い展開が残る
//
// fitbit と位置情報の2つが同じものを必要としたのでSDKに置いた。
// 先行する3つのプラグイン(chatgpt/claudeai/claudecode)はZIPを読まないので手を入れていない。

import (
	"archive/zip"
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-zglob"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const (
	// maxEntrySize は1エントリの展開後サイズの上限。これを名乗るエントリは読まない。
	// 実データの最大は Timeline Edits.json の5.4MB、旧形式の Records.json でも1GB程度なので、
	// 実データを弾かない位置に置いてある。
	maxEntrySize = 2 << 30 // 2GiB

	// maxEntryCount は1つのZIPから受け取るエントリ数の上限。
	// 実データは11,813件。桁が違うものは壊れているか作為的とみなす。
	maxEntryCount = 1 << 20

	// suspiciousExpandRatio は「展開後の合計 ÷ ZIPのサイズ」の警告しきい値。
	// 実データは 3.73GB / 271MB = 約14倍。展開はしないので止めはせず、知らせるだけ。
	suspiciousExpandRatio = 200

	// zipEntrySeparator はZIPのパスとエントリ名を繋ぐ区切り。
	// Java の jar URL と同じ慣習。この文字列を解析して元に戻すことはしない
	// （ArchivePath / EntryName を別フィールドで持つ）ので、
	// ファイル名に "!" が含まれていても問題にならない。
	zipEntrySeparator = "!/"
)

// SourceEntry は取り込み対象の1エントリ。ZIPの中のファイル1つを指す。
type SourceEntry struct {
	// Path は差分キャッシュのキーにする一意な名前。"<ZIPの絶対パス>!/<ZIP内のパス>"。
	// 設定画面や詳細画面にそのまま出るので読める形にしてある。
	Path string

	// Name はベース名。"steps_2024-04-01.csv" など。
	Name string

	// EntryName はZIP内のパス。区切りは常に "/"（ZIPの仕様）。
	// filepath ではなく path で扱うこと。
	EntryName string

	// ArchivePath はこのエントリを含むZIPの絶対パス。
	ArchivePath string

	// ExportID は取り込み世代の識別子。
	// 「同じ世代（分割された -001 -002 …）の寄与は足す。
	//   違う世代の同じ日は足さずに新しい世代を採る」の単位になる。
	ExportID string

	// MtimeUnix はエントリの更新時刻。
	//
	// Takeout は書き出し時刻を全エントリに同じ値で入れるので、
	// 実質「その世代がいつ作られたか」を表す。世代の新旧はこれで決める。
	// **エントリの中身が変わったかの判定には使えない**（同じ世代の全エントリが同値）。
	// そちらは CRC32 を使うこと。
	MtimeUnix int64

	// Size は展開後のサイズ。中央ディレクトリから取るので展開は要らない。
	Size int64

	// CRC32 は中央ディレクトリに書かれている中身のCRC32。**差分判定はこれで行う。**
	// 汎用フラグ bit 3 が立っていてもローカルヘッダ側が0になるだけで、
	// 中央ディレクトリには入っている。
	CRC32 uint32

	zipFile *zip.File
}

// Open はエントリの中身を読むReaderを返す。必ず Close すること。
//
// 途中でやめてよい（形式判定は先頭64KBしか読まない）。
// 同じ SourceSet の別のエントリと並行に開いてよい。
// archive/zip は Open ごとに io.SectionReader を作るので、
// 共有している *os.File の読み出し位置は動かない。
func (e SourceEntry) Open() (io.ReadCloser, error) {
	if e.zipFile == nil {
		return nil, fmt.Errorf("source entry %s is not openable", e.Path)
	}
	reader, err := e.zipFile.Open()
	if err != nil {
		return nil, fmt.Errorf("error at open zip entry %s: %w", e.Path, err)
	}
	// ヘッダが嘘をついているZIPを止める。
	// archive/zip は最後まで読んでからでないと食い違いに気づかない。
	return &limitedReadCloser{inner: reader, remain: e.Size + 1}, nil
}

// ReadHead は先頭 n バイトまでを読む。形式判定に使う。
// エントリが n より小さければその長さだけ返る。
func (e SourceEntry) ReadHead(n int) ([]byte, error) {
	if int64(n) > e.Size {
		n = int(e.Size)
	}
	if n <= 0 {
		return nil, nil
	}
	reader, err := e.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	head := make([]byte, n)
	read, err := io.ReadFull(reader, head)
	if err != nil && read == 0 {
		return nil, err
	}
	return head[:read], nil
}

// ReadAllLimited は中身を全部読む。limit を超えたらエラーにする。
// 丸ごとメモリに載せるパーサ（Timeline Edits.json など）のための上限。
func ReadAllLimited(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("entry is larger than %d bytes", limit)
	}
	return body, nil
}

// limitedReadCloser はヘッダが名乗ったサイズを超えて読めないようにする。
type limitedReadCloser struct {
	inner  io.ReadCloser
	remain int64
}

func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.remain <= 0 {
		return 0, errors.New("zip entry is longer than its header claims")
	}
	if int64(len(p)) > l.remain {
		p = p[:l.remain]
	}
	n, err := l.inner.Read(p)
	l.remain -= int64(n)
	return n, err
}

func (l *limitedReadCloser) Close() error { return l.inner.Close() }

// SourceProblemKind は走査で見つかった問題の種類。
type SourceProblemKind string

const (
	// ProblemSpannedArchive は分割ZIP（.z01 .z02 … と .zip の組）。
	ProblemSpannedArchive SourceProblemKind = "spanned_archive"
	// ProblemExtractedFolder はZIPが無く展開済みのファイルだけがある場所。
	ProblemExtractedFolder SourceProblemKind = "extracted_folder"
	// ProblemNestedZip はZIPの中のZIP。中には入らない。
	ProblemNestedZip SourceProblemKind = "nested_zip"
	// ProblemBrokenArchive は開けなかったZIP。
	ProblemBrokenArchive SourceProblemKind = "broken_archive"
	// ProblemEncryptedEntry はパスワード付きのエントリ。
	ProblemEncryptedEntry SourceProblemKind = "encrypted_entry"
	// ProblemHugeEntry は上限を超える大きさを名乗るエントリ。
	ProblemHugeEntry SourceProblemKind = "huge_entry"
	// ProblemMixedExports は1つのフォルダに時期の違う書き出しが混ざっている。
	ProblemMixedExports SourceProblemKind = "mixed_exports"
	// ProblemMissingPattern は何にもマッチしなかった指定。
	ProblemMissingPattern SourceProblemKind = "missing_pattern"
)

// SourceProblem は走査で見つかった問題。
//
// 走査は止めずに設定画面に出して知らせる。1つの壊れたZIPで全部を止めない。
type SourceProblem struct {
	Kind SourceProblemKind
	Path string
	// Message は日本語。そのまま設定画面に出せる。
	Message string
}

// ExportInfo は取り込み世代1つぶん。
type ExportInfo struct {
	// ExportID は世代の識別子。
	ExportID string
	// Dir はZIPを含むフォルダ。
	Dir string
	// ArchivePaths はこの世代に属するZIP（分割なら複数）。
	ArchivePaths []string
	// NewestMtimeUnix はエントリ更新時刻の最大。世代の新旧はこれで決める。
	NewestMtimeUnix int64
	// EntryCount は採用したエントリ数。
	EntryCount int
}

// SourceSet は開いたZIP一式。
// エントリを読み終わるまでZIPを開いたままにする必要があるので、必ず Close すること。
type SourceSet struct {
	entries  []SourceEntry
	exports  []ExportInfo
	problems []SourceProblem
	closers  []io.Closer
}

// Entries は見つかった全エントリを返す。
func (s *SourceSet) Entries() []SourceEntry { return s.entries }

// Exports は世代ごとの情報を新しい順に返す。
func (s *SourceSet) Exports() []ExportInfo { return s.exports }

// Problems は走査で見つかった問題を返す。
func (s *SourceSet) Problems() []SourceProblem { return s.problems }

// Close は開いたZIPを全部閉じる。
func (s *SourceSet) Close() error {
	if s == nil {
		return nil
	}
	var closeErrors []error
	for _, closer := range s.closers {
		if err := closer.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	s.closers = nil
	return errors.Join(closeErrors...)
}

// OpenSources は取り込み元のパターンからZIPを探して開き、中のエントリを列挙する。
//
// patterns はワイルドカード・先頭の ~ ・環境変数を展開したうえで解決する
// （ExpandSourcePatterns と同じ）。フォルダは再帰的に走査し、
// ファイルはそれ自体がZIPのときだけ使う。**ZIP以外は読まない。**
//
// accept が非nilならZIP内のパスを渡して選別する。11,813件のうち必要なのは1,964件、
// といった絞り込みをここでやると持つ構造体が減る。
//
// 返した *SourceSet は必ず Close すること。
func OpenSources(patterns []string, accept func(entryName string) bool) (*SourceSet, error) {
	set := &SourceSet{}
	expanded := ExpandSourcePatterns(patterns)
	for _, missing := range expanded.Missing {
		set.problems = append(set.problems, SourceProblem{
			Kind:    ProblemMissingPattern,
			Path:    missing,
			Message: "この指定は何にもマッチしませんでした。",
		})
	}

	archives, walkErr := set.findArchives(expanded.Dirs, expanded.Files)
	for _, archivePath := range archives {
		set.openArchive(archivePath, accept)
	}
	set.buildExports()
	return set, walkErr
}

// findArchives は走査対象のZIPを集める。
// ついでに「ZIPが無いのにデータらしきファイルがある場所」を見つけて報告する。
func (s *SourceSet) findArchives(dirs []string, files []string) ([]string, error) {
	found := map[string]bool{}
	archives := []string{}
	var walkErrors []error

	add := func(candidate string) {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			abs = candidate
		}
		if found[abs] {
			return
		}
		found[abs] = true
		archives = append(archives, abs)
	}

	for _, dir := range dirs {
		zipCount, looseCount := 0, 0
		err := filepath.WalkDir(dir, func(walkPath string, entry fs.DirEntry, err error) error {
			if err != nil {
				if entry != nil && entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(walkPath)) {
			case ".zip":
				zipCount++
				add(walkPath)
			case ".csv", ".json":
				looseCount++
			}
			return nil
		})
		if err != nil {
			walkErrors = append(walkErrors, err)
		}
		if zipCount == 0 && looseCount != 0 {
			abs, absErr := filepath.Abs(dir)
			if absErr != nil {
				abs = dir
			}
			s.problems = append(s.problems, SourceProblem{
				Kind: ProblemExtractedFolder,
				Path: abs,
				Message: "展開済みの Takeout フォルダのようですが、いまはZIPしか読みません。" +
					"ダウンロードした takeout-*.zip を展開せずにこのフォルダへ置いてください。" +
					"分割された -001 -002 … は全部同じフォルダに置きます。",
			})
		}
	}

	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".zip") {
			add(file)
			continue
		}
		s.problems = append(s.problems, SourceProblem{
			Kind:    ProblemExtractedFolder,
			Path:    file,
			Message: "ZIPではないので読みません。takeout-*.zip を指定してください。",
		})
	}

	slices.Sort(archives)
	return archives, errors.Join(walkErrors...)
}

// openArchive はZIPを1つ開いてエントリを取り込む。
func (s *SourceSet) openArchive(archivePath string, accept func(entryName string) bool) {
	// 分割ZIP（.z01 .z02 … と .zip の組）は archive/zip では読めない。
	// 最後の .zip だけは開けてしまうことがあり、そのときエントリの中身は
	// 前のボリュームにあるので読むと壊れた値が返る。開く前に見分けて丸ごと拒否する。
	//
	// Google Takeout の -001 -002 は分割ZIPではない。
	// あれはそれぞれ単独で開ける完全なZIPなのでここには引っかからない。
	if siblings := spannedSiblings(archivePath); len(siblings) != 0 {
		s.problems = append(s.problems, SourceProblem{
			Kind: ProblemSpannedArchive,
			Path: archivePath,
			Message: fmt.Sprintf("分割ZIP（%s があります）は読めません。"+
				"7-Zip などで1つのZIPにまとめ直してから置いてください。"+
				"Google Takeout の -001 -002 … は分割ZIPではないので、そのままで構いません。",
				filepath.Base(siblings[0])),
		})
		return
	}

	file, err := os.Open(archivePath)
	if err != nil {
		s.problems = append(s.problems, SourceProblem{
			Kind: ProblemBrokenArchive, Path: archivePath,
			Message: "開けませんでした: " + err.Error(),
		})
		return
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		s.problems = append(s.problems, SourceProblem{
			Kind: ProblemBrokenArchive, Path: archivePath,
			Message: "情報を取れませんでした: " + err.Error(),
		})
		return
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		_ = file.Close()
		s.problems = append(s.problems, SourceProblem{
			Kind: ProblemBrokenArchive, Path: archivePath,
			Message: "ZIPとして読めませんでした（壊れているかZIPではありません）: " + err.Error(),
		})
		return
	}
	if len(reader.File) > maxEntryCount {
		_ = file.Close()
		s.problems = append(s.problems, SourceProblem{
			Kind: ProblemBrokenArchive, Path: archivePath,
			Message: fmt.Sprintf("エントリが多すぎます(%d件)。読みません。", len(reader.File)),
		})
		return
	}
	// ここから先、file は SourceSet が閉じる。エントリを読むのに要る。
	s.closers = append(s.closers, file)

	exportID := ExportIDOf(archivePath)
	declaredTotal := int64(0)

	for _, zipFile := range reader.File {
		entryName := DecodeZipEntryName(zipFile)

		// ディレクトリのエントリ。Takeout には1件も無いが、他のZIPには有る。
		if strings.HasSuffix(entryName, "/") || zipFile.FileInfo().IsDir() {
			continue
		}
		// 通常ファイル以外は読まない。シンボリックリンクはこれで落ちる
		// （中身はリンク先の文字列で、辿るとZIPの外に出られる）。
		if !zipFile.Mode().IsRegular() {
			continue
		}
		// パスワード付きは読めない。
		if zipFile.Flags&0x1 != 0 {
			s.problems = append(s.problems, SourceProblem{
				Kind: ProblemEncryptedEntry, Path: archivePath + zipEntrySeparator + entryName,
				Message: "パスワード付きのエントリは読めません。",
			})
			continue
		}
		// ZIPの中のZIPには入らない。
		// 入るには内側を丸ごとメモリかディスクに置く必要があり、
		// 「展開しない」方針と噛み合わない。見つけたことだけ知らせる。
		if strings.EqualFold(path.Ext(entryName), ".zip") {
			s.problems = append(s.problems, SourceProblem{
				Kind: ProblemNestedZip, Path: archivePath + zipEntrySeparator + entryName,
				Message: "ZIPの中のZIPは読みません。中身が要るなら取り出して置いてください。",
			})
			continue
		}
		if zipFile.UncompressedSize64 > maxEntrySize {
			s.problems = append(s.problems, SourceProblem{
				Kind: ProblemHugeEntry, Path: archivePath + zipEntrySeparator + entryName,
				Message: fmt.Sprintf("大きすぎるので読みません(%d バイト)。", zipFile.UncompressedSize64),
			})
			continue
		}
		declaredTotal += int64(zipFile.UncompressedSize64)

		baseName := path.Base(entryName)
		if baseName == "" || baseName == "." || baseName == "/" {
			continue
		}
		if accept != nil && !accept(entryName) {
			continue
		}

		s.entries = append(s.entries, SourceEntry{
			Path:        archivePath + zipEntrySeparator + entryName,
			Name:        baseName,
			EntryName:   entryName,
			ArchivePath: archivePath,
			ExportID:    exportID,
			MtimeUnix:   zipFile.Modified.Unix(),
			Size:        int64(zipFile.UncompressedSize64),
			CRC32:       zipFile.CRC32,
			zipFile:     zipFile,
		})
	}

	if info.Size() > 0 && declaredTotal/info.Size() > suspiciousExpandRatio {
		s.problems = append(s.problems, SourceProblem{
			Kind: ProblemHugeEntry, Path: archivePath,
			Message: fmt.Sprintf("展開後が %d 倍になります。読みはしますが、意図したZIPか確かめてください。",
				declaredTotal/info.Size()),
		})
	}
}

// buildExports は世代ごとの情報をまとめ、新しい順に並べる。
func (s *SourceSet) buildExports() {
	byID := map[string]*ExportInfo{}
	for _, entry := range s.entries {
		export, exist := byID[entry.ExportID]
		if !exist {
			export = &ExportInfo{ExportID: entry.ExportID, Dir: filepath.Dir(entry.ArchivePath)}
			byID[entry.ExportID] = export
		}
		export.EntryCount++
		if entry.MtimeUnix > export.NewestMtimeUnix {
			export.NewestMtimeUnix = entry.MtimeUnix
		}
		if !slices.Contains(export.ArchivePaths, entry.ArchivePath) {
			export.ArchivePaths = append(export.ArchivePaths, entry.ArchivePath)
		}
	}
	for _, export := range byID {
		slices.Sort(export.ArchivePaths)
		s.exports = append(s.exports, *export)
	}
	// 新しい書き出しが先。同着はIDの降順（日付入りのフォルダ名で新しい方が勝つ）。
	slices.SortFunc(s.exports, func(a, b ExportInfo) int {
		if a.NewestMtimeUnix != b.NewestMtimeUnix {
			return cmp.Compare(b.NewestMtimeUnix, a.NewestMtimeUnix)
		}
		return cmp.Compare(b.ExportID, a.ExportID)
	})

	// 1つのフォルダに時期の違う書き出しが混ざっていたら知らせる。
	// 世代の識別子に書き出し時刻を混ぜてあるので合算はされないが、
	// 置き方としては分けたほうが分かりやすい。
	dirExports := map[string]int{}
	for _, export := range s.exports {
		dirExports[export.Dir]++
	}
	for dir, count := range dirExports {
		if count > 1 {
			s.problems = append(s.problems, SourceProblem{
				Kind: ProblemMixedExports,
				Path: dir,
				Message: fmt.Sprintf("1つのフォルダに時期の違う書き出しが %d 個あります。"+
					"重なる日は新しいほうだけを使うので二重にはなりませんが、"+
					"書き出しごとにフォルダを分けたほうが分かりやすくなります。", count),
			})
		}
	}
}

// SplitZipEntryPath は SourceEntry.Path をZIPのパスとエントリ名に分ける。
// ZIP内のエントリでなければ ok=false。表示用で、取り込みの判定には使わない。
func SplitZipEntryPath(entryPath string) (archivePath string, entryName string, ok bool) {
	index := strings.LastIndex(entryPath, zipEntrySeparator)
	if index < 0 {
		return "", "", false
	}
	return entryPath[:index], entryPath[index+len(zipEntrySeparator):], true
}

// DecodeZipEntryName はZIPエントリのファイル名をUTF-8に変換する。
//
// ZIP仕様では汎用フラグ bit 11 (0x800) が立っていればUTF-8。
// そうでない場合、日本語環境では Shift_JIS (CP932) が使われることが多いため、
// UTF-8でなければ Shift_JIS としてデコードを試みる。
// gkill本体の handle_browse_zip_contents.go と同じ判定にしてある。
//
// なお Google Takeout は bit 11 を立てて書くので、実データではこの変換は走らない。
func DecodeZipEntryName(zipFile *zip.File) string {
	name := zipFile.Name

	// UTF-8フラグが立っている場合はそのまま
	if !zipFile.NonUTF8 {
		return name
	}
	// 既にvalid UTF-8ならそのまま（pure ASCIIを含む）
	if utf8.ValidString(name) {
		return name
	}
	decoded, _, err := transform.String(japanese.ShiftJIS.NewDecoder(), name)
	if err != nil {
		return name // 変換失敗時は元のまま
	}
	return decoded
}

// spannedSiblings は分割ZIPの相棒（.z01 .z02 …）を探す。
//
// 拡張子が .z<数字> のものだけを相棒とみなす。
// Google Takeout の -001 -002 は拡張子が .zip なので引っかからない。
func spannedSiblings(archivePath string) []string {
	base := strings.TrimSuffix(archivePath, filepath.Ext(archivePath))
	siblings := []string{}
	for i := 1; i <= 99; i++ {
		candidate := fmt.Sprintf("%s.z%02d", base, i)
		if _, err := os.Stat(candidate); err != nil {
			break
		}
		siblings = append(siblings, candidate)
	}
	return siblings
}

// ExportIDOf はZIPのパスから取り込み世代の識別子を作る。
//
// 基本はフォルダで、利用者の ~/Kyou/GoogleTakeout_<端末>_<日付>/ という慣習に合う。
// 名前から書き出し時刻が読めるときは、それも混ぜる。
//
// **フォルダだけを世代の単位にしてはいけない。** 同じフォルダに翌月の書き出しを
// 足したときに両方が同じ世代になり、歩数のような合計する指標が2倍になる。
// 分割された -1-001 -1-002 … は書き出し時刻が同じなので、正しく1つの世代にまとまる。
func ExportIDOf(archivePath string) string {
	dir := filepath.Dir(archivePath)
	if stamp := TakeoutStamp(filepath.Base(archivePath)); stamp != "" {
		return dir + "|" + stamp
	}
	return dir
}

// TakeoutStamp は "takeout-20260808T230152Z-1-001.zip" から "20260808T230152Z" を取り出す。
// Takeout の名前でなければ空を返す。
func TakeoutStamp(baseName string) string {
	const prefix = "takeout-"
	const stampLen = len("20260808T230152Z")
	if len(baseName) < len(prefix)+stampLen {
		return ""
	}
	if !strings.EqualFold(baseName[:len(prefix)], prefix) {
		return ""
	}
	stamp := baseName[len(prefix) : len(prefix)+stampLen]
	for i := range stampLen {
		switch i {
		case 8:
			if stamp[i] != 'T' && stamp[i] != 't' {
				return ""
			}
		case stampLen - 1:
			if stamp[i] != 'Z' && stamp[i] != 'z' {
				return ""
			}
		default:
			if stamp[i] < '0' || stamp[i] > '9' {
				return ""
			}
		}
	}
	return strings.ToUpper(stamp)
}

// ---- 取り込み元の指定の展開 ----
//
// fitbit と位置情報で完全に同じものを持っていたのでここへ移した。

// ExpandedSource はパターンを展開した結果。
type ExpandedSource struct {
	// Dirs は再帰的に走査するディレクトリ。
	Dirs []string
	// Files は直接対象にするファイル。
	Files []string
	// Missing は実在しなかった、あるいは何にもマッチしなかったパターン。
	Missing []string
}

// ParseSourcePatterns は設定値をパターンのリストにする。
// 設定は文字列(改行区切り)でも配列でも書ける。
//
//	"source_dirs": "C:\\a\nC:\\b"
//	"source_dirs": ["C:\\a", "C:\\b"]
//
// 先頭の ~ と環境変数を展開する。空なら defaultPattern にフォールバックする。
func ParseSourcePatterns(value any, defaultPattern string) []string {
	var patterns []string
	add := func(s string) {
		s = strings.TrimSpace(strings.TrimSuffix(s, "\r"))
		if s == "" {
			return
		}
		patterns = append(patterns, ExpandHome(os.ExpandEnv(s)))
	}

	switch v := value.(type) {
	case nil:
	case string:
		for line := range strings.SplitSeq(v, "\n") {
			add(line)
		}
	case []string:
		for _, s := range v {
			for line := range strings.SplitSeq(s, "\n") {
				add(line)
			}
		}
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok {
				for line := range strings.SplitSeq(s, "\n") {
					add(line)
				}
			}
		}
	}

	if len(patterns) == 0 && defaultPattern != "" {
		patterns = append(patterns, ExpandHome(os.ExpandEnv(defaultPattern)))
	}
	return patterns
}

// ExpandHome は先頭の ~ をホームディレクトリに展開する。
//
// Windowsサービスとして動くgkillでは実行アカウントのホームになる点に注意
// (LocalSystemなら systemprofile)。確実に指定したいときは絶対パスを使う。
func ExpandHome(pathValue string) string {
	if pathValue != "~" && !strings.HasPrefix(pathValue, "~/") && !strings.HasPrefix(pathValue, `~\`) {
		return pathValue
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return pathValue
	}
	if pathValue == "~" {
		return home
	}
	return filepath.Join(home, pathValue[2:])
}

// HasGlobMeta はパターンにワイルドカードが含まれるかを判定する。
func HasGlobMeta(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// ExpandSourcePatterns はパターンを実在するパスへ展開する。
//
// ワイルドカード(* ** ? [])を含むパターンはグロブ展開し、
// マッチしたものがディレクトリなら再帰走査の対象、ファイルならそのまま対象にする。
func ExpandSourcePatterns(patterns []string) ExpandedSource {
	var result ExpandedSource
	seen := map[string]bool{}

	classify := func(candidate string) bool {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			abs = candidate
		}
		if seen[abs] {
			return true
		}
		info, err := os.Stat(abs)
		if err != nil {
			return false
		}
		seen[abs] = true
		if info.IsDir() {
			result.Dirs = append(result.Dirs, abs)
		} else {
			result.Files = append(result.Files, abs)
		}
		return true
	}

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		if !HasGlobMeta(pattern) {
			if !classify(pattern) {
				result.Missing = append(result.Missing, pattern)
			}
			continue
		}
		matches, err := zglob.Glob(pattern)
		if err != nil || len(matches) == 0 {
			result.Missing = append(result.Missing, pattern)
			continue
		}
		matched := false
		for _, match := range matches {
			if classify(match) {
				matched = true
			}
		}
		if !matched {
			result.Missing = append(result.Missing, pattern)
		}
	}
	return result
}
