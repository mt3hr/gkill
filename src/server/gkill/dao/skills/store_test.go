package skills

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

const testUser = "testuser"

func manifest(name string, description string) []byte {
	return []byte("---\nname: " + name + "\ndescription: " + description + "\n---\n# " + name + "\n")
}

type zipItem struct {
	name    string
	content string
	mode    fs.FileMode
}

func makeZip(t *testing.T, items ...zipItem) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, item := range items {
		header := &zip.FileHeader{Name: item.name, Method: zip.Deflate}
		if item.mode != 0 {
			header.SetMode(item.mode)
		}
		w, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("CreateHeader(%s): %v", item.name, err)
		}
		if _, err := w.Write([]byte(item.content)); err != nil {
			t.Fatalf("Write(%s): %v", item.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return buf.Bytes()
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	return NewStore(t.TempDir())
}

func mustWrite(t *testing.T, store *Store, user string, name string, path string, content []byte, revision string) string {
	t.Helper()
	newRevision, err := store.WriteFile(context.Background(), user, name, path, content, revision)
	if err != nil {
		t.Fatalf("WriteFile(%s, %s): %v", name, path, err)
	}
	return newRevision
}

func TestValidateSkillName(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
	}{
		{"weekly-dashboard", true},
		{"a", true},
		{"a1-b2", true},
		{strings.Repeat("a", 64), true},
		{strings.Repeat("a", 65), false},
		{"", false},
		{"-a", false},
		{"a-", false},
		{"Weekly", false},
		{"_global", false},
		{".git", false},
		{"a_b", false},
		{"a.b", false},
		{"日本語", false},
	}
	for _, c := range cases {
		err := ValidateSkillName(c.name)
		if (err == nil) != c.ok {
			t.Errorf("ValidateSkillName(%q) = %v, want ok=%v", c.name, err, c.ok)
		}
		if err != nil && !errors.Is(err, ErrInvalidName) {
			t.Errorf("ValidateSkillName(%q) error is not ErrInvalidName: %v", c.name, err)
		}
	}
}

func TestNormalizeFilePath(t *testing.T) {
	cases := []struct {
		path string
		ok   bool
	}{
		{"SKILL.md", true},
		{"scripts/build_html.py", true},
		{"assets/logo.v2.png", true},
		{"a/b/c/d/e/f.txt", true},
		{"a..b.txt", true},
		{"", false},
		{"../evil", false},
		{"a/../b", false},
		{"/abs", false},
		{`scripts\build.py`, false},
		{".hidden", false},
		{"dir/.env", false},
		{"a//b", false},
		{"trailing.", false},
		{"CON", false},
		{"nul.txt", false},
		{"dir/com1.md", false},
		{"skill.md", false},
		{"Skill.MD", false},
		{"docs/skill.md", true},
		{"日本語.md", false},
		{"with space.md", false},
	}
	for _, c := range cases {
		_, err := NormalizeFilePath(c.path)
		if (err == nil) != c.ok {
			t.Errorf("NormalizeFilePath(%q) = %v, want ok=%v", c.path, err, c.ok)
		}
		if err != nil && !errors.Is(err, ErrInvalidPath) {
			t.Errorf("NormalizeFilePath(%q) error is not ErrInvalidPath: %v", c.path, err)
		}
	}
}

func TestJoinWithin(t *testing.T) {
	root := t.TempDir()
	// ".." を含む正しい名前は、そのままの名前で繋がる（".." を取り除いて別名にしない）
	for _, rel := range []string{ManifestFileName, "a..b.txt", "dir/a..b.md"} {
		joined, err := joinWithin(root, rel)
		if err != nil {
			t.Errorf("joinWithin(%q) = %v", rel, err)
			continue
		}
		if expected := filepath.Join(root, filepath.FromSlash(rel)); joined != expected {
			t.Errorf("joinWithin(%q) = %q, want %q", rel, joined, expected)
		}
	}
	for _, rel := range []string{"", ".", "../evil", "a/../../evil", filepath.ToSlash(filepath.Join(root, "abs"))} {
		if _, err := joinWithin(root, rel); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("joinWithin(%q) = %v, want ErrInvalidPath", rel, err)
		}
	}
}

func TestParseManifest(t *testing.T) {
	cases := []struct {
		label    string
		content  string
		expected string
		ok       bool
	}{
		{"basic", "---\nname: weekly\ndescription: 週次まとめ\n---\nbody\n", "weekly", true},
		{"crlf", "---\r\nname: weekly\r\ndescription: d\r\n---\r\nbody", "weekly", true},
		{"bom", "\uFEFF---\nname: weekly\ndescription: d\n---\n", "weekly", true},
		{"folded description", "---\nname: weekly\ndescription: >\n  line one\n  line two\nlicense: MIT\n---\n", "weekly", true},
		{"no expected name", "---\nname: weekly\ndescription: d\n---\n", "", true},
		// MCP の gkill_add_skill は値を JSON の文字列で書く（encoding/json は < > & を \u003c 等にする）
		{"json quoted (mcp gkill_add_skill)", "---\nname: \"weekly\"\ndescription: \"週次: \\\"まとめ\\\" # 夜 \\u003c3\"\n---\n# 手順\n", "weekly", true},
		{"no frontmatter", "# weekly\n", "weekly", false},
		{"unclosed", "---\nname: weekly\ndescription: d\n", "weekly", false},
		{"name mismatch", "---\nname: other\ndescription: d\n---\n", "weekly", false},
		{"empty description", "---\nname: weekly\ndescription: \"  \"\n---\n", "weekly", false},
		{"missing name", "---\ndescription: d\n---\n", "", false},
		{"invalid name", "---\nname: Weekly Report\ndescription: d\n---\n", "", false},
		{"broken yaml", "---\nname: [weekly\ndescription: d\n---\n", "", false},
	}
	for _, c := range cases {
		manifest, err := ParseManifest([]byte(c.content), c.expected)
		if (err == nil) != c.ok {
			t.Errorf("%s: ParseManifest = %v, want ok=%v", c.label, err, c.ok)
			continue
		}
		if err != nil && !errors.Is(err, ErrInvalidFrontmatter) {
			t.Errorf("%s: error is not ErrInvalidFrontmatter: %v", c.label, err)
		}
		if err == nil && manifest.Name != "weekly" {
			t.Errorf("%s: name = %q", c.label, manifest.Name)
		}
	}
}

func TestInspectReaderKeepsUTF8AcrossChunks(t *testing.T) {
	// 64KiB のチャンク境界で「あ」（3バイト）が割れる位置に置く
	content := append(bytes.Repeat([]byte("a"), 64*1024-1), []byte("あいう")...)
	inspected, err := inspectReader(bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if !inspected.isText {
		t.Errorf("multi-byte char split across chunks was judged binary")
	}
	if inspected.revision != RevisionOf(content) || inspected.size != int64(len(content)) {
		t.Errorf("revision/size mismatch: %+v", inspected)
	}

	for label, binary := range map[string][]byte{
		"nul":        []byte("abc\x00def"),
		"invalid":    {0xff, 0xfe, 0x00},
		"cut at end": append([]byte("abc"), 0xe3, 0x81),
	} {
		inspected, err := inspectReader(bytes.NewReader(binary))
		if err != nil {
			t.Fatal(err)
		}
		if inspected.isText {
			t.Errorf("%s: judged text", label)
		}
		if IsText(binary) {
			t.Errorf("%s: IsText = true", label)
		}
	}
}

func TestWriteFileCreateUpdateAndConflicts(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// スキルが無いときは SKILL.md 以外は書けない
	if _, err := store.WriteFile(ctx, testUser, "weekly", "notes.md", []byte("x"), ""); !errors.Is(err, ErrSkillNotFound) {
		t.Fatalf("write to missing skill: %v", err)
	}
	// スキルが無いときは SKILL.md でも revision 付き（上書きのつもり）なら作らない
	// （ErrFileNotFound や ErrRevisionConflict ではなく、スキルそのものが無いと伝える）
	if _, err := store.WriteFile(ctx, testUser, "weekly", ManifestFileName, manifest("weekly", "d"), "0000000000000000"); !errors.Is(err, ErrSkillNotFound) {
		t.Fatalf("update manifest of missing skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), testUser, "weekly")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("rejected write created the skill directory: %v", err)
	}
	// name がディレクトリと違う SKILL.md は拒否
	if _, err := store.WriteFile(ctx, testUser, "weekly", ManifestFileName, manifest("other", "d"), ""); !errors.Is(err, ErrInvalidFrontmatter) {
		t.Fatalf("mismatched manifest: %v", err)
	}

	revision := mustWrite(t, store, testUser, "weekly", ManifestFileName, manifest("weekly", "d"), "")
	if revision != RevisionOf(manifest("weekly", "d")) {
		t.Errorf("revision = %s", revision)
	}
	// revision 省略は新規作成だけ
	if _, err := store.WriteFile(ctx, testUser, "weekly", ManifestFileName, manifest("weekly", "d2"), ""); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("create over existing: %v", err)
	}
	// 食い違う revision は拒否し、今の revision を伝える
	_, err := store.WriteFile(ctx, testUser, "weekly", ManifestFileName, manifest("weekly", "d2"), "0000000000000000")
	if !errors.Is(err, ErrRevisionConflict) || !strings.Contains(DescribeError(err), revision) {
		t.Fatalf("stale revision: %v", err)
	}
	updated := mustWrite(t, store, testUser, "weekly", ManifestFileName, manifest("weekly", "d2"), revision)
	if updated == revision {
		t.Errorf("revision did not change")
	}
	// 無いファイルへ revision 付きで書くのは ErrFileNotFound
	if _, err := store.WriteFile(ctx, testUser, "weekly", "missing.md", []byte("x"), "0000000000000000"); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("update missing file: %v", err)
	}

	mustWrite(t, store, testUser, "weekly", "scripts/build.py", []byte("print(1)\r\n"), "")
	// 大文字小文字だけ違うパスは重複
	if _, err := store.WriteFile(ctx, testUser, "weekly", "Scripts/build.py", []byte("x"), ""); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("case conflict: %v", err)
	}
	// ファイルとフォルダの衝突
	if _, err := store.WriteFile(ctx, testUser, "weekly", "scripts", []byte("x"), ""); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("file/dir conflict: %v", err)
	}
	if _, err := store.WriteFile(ctx, testUser, "weekly", "scripts/build.py/x", []byte("x"), ""); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("dir/file conflict: %v", err)
	}

	// 改行コードはそのまま残る
	content, err := store.ReadFile(testUser, "weekly", "scripts/build.py", 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(content.Content) != "print(1)\r\n" {
		t.Errorf("content changed: %q", content.Content)
	}
	// 一時ファイルが残っていない
	entries, _ := os.ReadDir(filepath.Join(store.Root(), testUser, "weekly"))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tmp-") {
			t.Errorf("temp file left: %s", entry.Name())
		}
	}
}

func TestListAndGet(t *testing.T) {
	store := newTestStore(t)
	mustWrite(t, store, testUser, "zeta", ManifestFileName, manifest("zeta", "last"), "")
	mustWrite(t, store, testUser, "alpha", ManifestFileName, manifest("alpha", "first"), "")
	mustWrite(t, store, testUser, "alpha", "references/tags.md", []byte("# tags"), "")
	mustWrite(t, store, testUser, "alpha", "assets/logo.png", []byte{0x89, 'P', 'N', 'G', 0x00}, "")

	userDir := filepath.Join(store.Root(), testUser)
	// 手で作った壊れたスキル・規則外のディレクトリ・予約名・ドットで始まるもの
	for _, dir := range []string{"broken", "Not A Skill", "_global", "_tmp-x", ".git"} {
		if err := os.MkdirAll(filepath.Join(userDir, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(userDir, "broken", "notes.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// スキル内のドットファイルは無視される
	if err := os.WriteFile(filepath.Join(userDir, "alpha", ".DS_Store"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(testUser)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, summary := range list {
		names = append(names, summary.Name)
	}
	if !slices.Equal(names, []string{"alpha", "broken", "zeta"}) {
		t.Fatalf("names = %v", names)
	}
	if list[0].Description != "first" || list[0].FileCount != 3 || list[0].InvalidReason != "" {
		t.Errorf("alpha = %+v", list[0])
	}
	if list[1].InvalidReason == "" || list[1].Description != "" {
		t.Errorf("broken skill should carry a reason: %+v", list[1])
	}

	skill, err := store.Get(testUser, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(skill.Manifest, manifest("alpha", "first")) || skill.ManifestRevision != RevisionOf(skill.Manifest) {
		t.Errorf("manifest = %q", skill.Manifest)
	}
	paths := []string{}
	for _, file := range skill.Files {
		paths = append(paths, file.Path)
		if file.Path == "assets/logo.png" && file.IsText {
			t.Errorf("png judged text")
		}
		if file.Path == "references/tags.md" && !file.IsText {
			t.Errorf("markdown judged binary")
		}
	}
	if !slices.Equal(paths, []string{ManifestFileName, "assets/logo.png", "references/tags.md"}) {
		t.Errorf("paths = %v", paths)
	}

	if _, err := store.Get(testUser, "missing"); !errors.Is(err, ErrSkillNotFound) {
		t.Errorf("missing skill: %v", err)
	}
	// 利用者ディレクトリがまだ無ければ空の一覧
	empty, err := store.List("someone-else")
	if err != nil || len(empty) != 0 {
		t.Errorf("empty list = %v, %v", empty, err)
	}
}

func TestReadFile(t *testing.T) {
	store := newTestStore(t)
	mustWrite(t, store, testUser, "weekly", ManifestFileName, manifest("weekly", "d"), "")
	mustWrite(t, store, testUser, "weekly", "references/big.md", bytes.Repeat([]byte("x"), 100), "")

	content, err := store.ReadFile(testUser, "weekly", "references/big.md", 0)
	if err != nil || content.Omitted || len(content.Content) != 100 || !content.IsText {
		t.Fatalf("read = %+v, %v", content, err)
	}
	omitted, err := store.ReadFile(testUser, "weekly", "references/big.md", 99)
	if err != nil || !omitted.Omitted || omitted.Content != nil || omitted.Size != 100 || omitted.Revision != content.Revision {
		t.Fatalf("omitted = %+v, %v", omitted, err)
	}
	// 上限ちょうどのサイズは省かない（「超える」だけが省く条件。>= にすると上限と同じ大きさのファイルが読めなくなる）
	exact, err := store.ReadFile(testUser, "weekly", "references/big.md", 100)
	if err != nil || exact.Omitted || len(exact.Content) != 100 || exact.Size != 100 || exact.Revision != content.Revision {
		t.Fatalf("maxBytes == size = %+v, %v", exact, err)
	}
	if _, err := store.ReadFile(testUser, "weekly", "references/missing.md", 0); !errors.Is(err, ErrFileNotFound) {
		t.Errorf("missing file: %v", err)
	}
	// 大文字小文字だけ違うパスでは読めない（Windows でも同じ結果になる）
	if _, err := store.ReadFile(testUser, "weekly", "References/big.md", 0); !errors.Is(err, ErrFileNotFound) {
		t.Errorf("case-different path: %v", err)
	}
	if _, err := store.ReadFile(testUser, "weekly", "../weekly/SKILL.md", 0); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("traversal: %v", err)
	}
}

// ".." を含む正しいファイル名が、書き込みでも zip の置き換えでも同じ名前のまま保存されること。
func TestFileNameWithDoubleDotKeepsItsName(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	mustWrite(t, store, testUser, "weekly", ManifestFileName, manifest("weekly", "d"), "")
	mustWrite(t, store, testUser, "weekly", "notes/v1..2.md", []byte("written"), "")
	content, err := store.ReadFile(testUser, "weekly", "notes/v1..2.md", 0)
	if err != nil || string(content.Content) != "written" {
		t.Fatalf("read after write = %+v, %v", content, err)
	}

	data := makeZip(t,
		zipItem{name: "weekly/" + ManifestFileName, content: string(manifest("weekly", "d"))},
		zipItem{name: "weekly/a..b.txt", content: "uploaded"},
	)
	if _, err := store.Replace(ctx, testUser, data); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if _, err := os.Stat(filepath.Join(store.Root(), testUser, "weekly", "a..b.txt")); err != nil {
		t.Errorf("a..b.txt is not on disk under its own name: %v", err)
	}
	content, err = store.ReadFile(testUser, "weekly", "a..b.txt", 0)
	if err != nil || string(content.Content) != "uploaded" {
		t.Fatalf("read after replace = %+v, %v", content, err)
	}
}

func TestDeleteFileAndSkill(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	mustWrite(t, store, testUser, "weekly", ManifestFileName, manifest("weekly", "d"), "")
	revision := mustWrite(t, store, testUser, "weekly", "scripts/deep/run.py", []byte("x"), "")

	if err := store.DeleteFile(ctx, testUser, "weekly", ManifestFileName, ""); !errors.Is(err, ErrManifestDelete) {
		t.Errorf("delete manifest: %v", err)
	}
	if err := store.DeleteFile(ctx, testUser, "weekly", "scripts/deep/run.py", "0000000000000000"); !errors.Is(err, ErrRevisionConflict) {
		t.Errorf("stale revision: %v", err)
	}
	if err := store.DeleteFile(ctx, testUser, "weekly", "scripts/deep/run.py", revision); err != nil {
		t.Fatal(err)
	}
	// 空になったフォルダも消える
	if _, err := os.Stat(filepath.Join(store.Root(), testUser, "weekly", "scripts")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("empty folder left: %v", err)
	}
	if err := store.DeleteFile(ctx, testUser, "weekly", "scripts/deep/run.py", ""); !errors.Is(err, ErrFileNotFound) {
		t.Errorf("delete missing: %v", err)
	}

	if err := store.DeleteSkill(ctx, testUser, "weekly"); err != nil {
		t.Fatal(err)
	}
	list, _ := store.List(testUser)
	if len(list) != 0 {
		t.Errorf("list after delete = %v", list)
	}
	entries, _ := os.ReadDir(filepath.Join(store.Root(), testUser))
	if len(entries) != 0 {
		t.Errorf("leftovers after delete: %v", entries)
	}
	if err := store.DeleteSkill(ctx, testUser, "weekly"); !errors.Is(err, ErrSkillNotFound) {
		t.Errorf("delete missing skill: %v", err)
	}
}

func TestUsersAreIsolated(t *testing.T) {
	store := newTestStore(t)
	mustWrite(t, store, "testuser_a", "weekly", ManifestFileName, manifest("weekly", "d"), "")

	list, err := store.List("testuser_b")
	if err != nil || len(list) != 0 {
		t.Errorf("other user's list = %v, %v", list, err)
	}
	if _, err := store.Get("testuser_b", "weekly"); !errors.Is(err, ErrSkillNotFound) {
		t.Errorf("other user's skill: %v", err)
	}
	for _, userID := range []string{"", ".", "..", "../testuser_a", "a/b", `a\b`, "C:", "a:b", "a\x00b"} {
		if _, err := store.List(userID); !errors.Is(err, ErrInvalidUserID) {
			t.Errorf("List(%q) = %v", userID, err)
		}
	}
}

// 大文字小文字だけ違う利用者IDは、既にある別の利用者のディレクトリを読みも書きもしない。
//
// Windows のファイルシステムは大小を区別しないので、exactChildDir が列挙して名前を突き合わせる
// （EqualFold の衝突検出）を外すと resolveUserDir が「無い」と判定し、その先の os.ReadDir / MkdirAll /
// rename は別の大小のディレクトリ（= 他人のスキル）へそのまま届く。resolveSkillDir を通る操作（Get /
// ReadFile / BuildZip / DeleteFile / DeleteSkill）は ErrSkillNotFound、List は空の一覧で止まるが、
// 利用者ディレクトリの有無を見ない WriteFile と Replace は他人のスキルを書き換え、PlanReplace は
// 他人のファイル名を計画に載せる。
// どの操作も ErrInvalidName で止まり、元の利用者のスキルは 1 バイトも変わらないことを固定する。
func TestUserDirCaseConflictIsRejected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	const owner = "TestUser"
	const impostor = "testuser"
	ownerManifest := manifest("weekly", "owner's description")
	ownerNotes := []byte("owner's notes\n")
	manifestRevision := mustWrite(t, store, owner, "weekly", ManifestFileName, ownerManifest, "")
	mustWrite(t, store, owner, "weekly", "references/notes.md", ownerNotes, "")

	// 読み取り系はどれも ErrInvalidName（ErrSkillNotFound や空の一覧で済ませない）
	if _, err := store.Get(impostor, "weekly"); !errors.Is(err, ErrInvalidName) {
		t.Errorf("Get = %v, want ErrInvalidName", err)
	}
	if list, err := store.List(impostor); !errors.Is(err, ErrInvalidName) {
		t.Errorf("List = %v, %v, want ErrInvalidName", list, err)
	}
	if _, err := store.ReadFile(impostor, "weekly", ManifestFileName, 0); !errors.Is(err, ErrInvalidName) {
		t.Errorf("ReadFile = %v, want ErrInvalidName", err)
	}
	if _, _, err := store.BuildZip(impostor, "weekly"); !errors.Is(err, ErrInvalidName) {
		t.Errorf("BuildZip = %v, want ErrInvalidName", err)
	}

	// 書き込み系。新規ファイルの追加、正しい revision を添えた SKILL.md の上書き、zip での置き換え、削除。
	// 衝突検出が無いと、Windows では WriteFile と Replace が TestUser/weekly へそのまま届き（PlanReplace は
	// その中身を計画に載せる）、DeleteFile と DeleteSkill は ErrSkillNotFound で止まる。止まる操作も ErrInvalidName に揃える。
	if _, err := store.WriteFile(ctx, impostor, "weekly", "injected.md", []byte("injected"), ""); !errors.Is(err, ErrInvalidName) {
		t.Errorf("WriteFile(new file) = %v, want ErrInvalidName", err)
	}
	if _, err := store.WriteFile(ctx, impostor, "weekly", ManifestFileName, manifest("weekly", "hijacked"), manifestRevision); !errors.Is(err, ErrInvalidName) {
		t.Errorf("WriteFile(manifest with the owner's revision) = %v, want ErrInvalidName", err)
	}
	if _, err := store.WriteFile(ctx, impostor, "brand-new", ManifestFileName, manifest("brand-new", "d"), ""); !errors.Is(err, ErrInvalidName) {
		t.Errorf("WriteFile(new skill) = %v, want ErrInvalidName", err)
	}
	replacement := makeZip(t,
		zipItem{name: "weekly/" + ManifestFileName, content: string(manifest("weekly", "replaced"))},
		zipItem{name: "weekly/references/notes.md", content: "replaced notes\n"},
	)
	if _, err := store.PlanReplace(impostor, replacement); !errors.Is(err, ErrInvalidName) {
		t.Errorf("PlanReplace = %v, want ErrInvalidName", err)
	}
	if _, err := store.Replace(ctx, impostor, replacement); !errors.Is(err, ErrInvalidName) {
		t.Errorf("Replace = %v, want ErrInvalidName", err)
	}
	if err := store.DeleteFile(ctx, impostor, "weekly", "references/notes.md", ""); !errors.Is(err, ErrInvalidName) {
		t.Errorf("DeleteFile = %v, want ErrInvalidName", err)
	}
	if err := store.DeleteSkill(ctx, impostor, "weekly"); !errors.Is(err, ErrInvalidName) {
		t.Errorf("DeleteSkill = %v, want ErrInvalidName", err)
	}

	// 元の利用者のスキルは中身もファイル構成も変わっていない
	skill, err := store.Get(owner, "weekly")
	if err != nil {
		t.Fatalf("owner's Get: %v", err)
	}
	if !bytes.Equal(skill.Manifest, ownerManifest) || skill.ManifestRevision != manifestRevision {
		t.Errorf("owner's manifest changed: %q", skill.Manifest)
	}
	paths := []string{}
	for _, file := range skill.Files {
		paths = append(paths, file.Path)
	}
	if !slices.Equal(paths, []string{ManifestFileName, "references/notes.md"}) {
		t.Errorf("owner's files changed: %v", paths)
	}
	notes, err := store.ReadFile(owner, "weekly", "references/notes.md", 0)
	if err != nil || !bytes.Equal(notes.Content, ownerNotes) {
		t.Errorf("owner's attached file changed: %+v, %v", notes, err)
	}
	// 元の利用者の一覧に、なりすましが作ろうとしたスキルが混ざっていない
	ownerList, err := store.List(owner)
	if err != nil || len(ownerList) != 1 || ownerList[0].Name != "weekly" || ownerList[0].Description != "owner's description" {
		t.Errorf("owner's list = %+v, %v", ownerList, err)
	}
	// 根の直下にも別名のディレクトリや作業用の残骸ができていない
	entries, err := os.ReadDir(store.Root())
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if !slices.Equal(names, []string{owner}) {
		t.Errorf("root entries = %v, want only %q", names, owner)
	}
}

// 一覧は、frontmatter の name がディレクトリ名と食い違うスキルを消さずに InvalidReason 付きで出す。
// 手で置いた・別名でコピーしたスキルは Get でも同じ理由を返し、description は空になる。
func TestListMarksManifestNameMismatchAsInvalid(t *testing.T) {
	store := newTestStore(t)
	mustWrite(t, store, testUser, "weekly", ManifestFileName, manifest("weekly", "d"), "")
	// ストア経由では書けない（WriteFile は name を検査する）ので、ディレクトリを直接作る
	copied := filepath.Join(store.Root(), testUser, "weekly-copy")
	if err := os.MkdirAll(copied, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(copied, ManifestFileName), manifest("weekly", "copied"), 0o644); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(testUser)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Name != "weekly" || list[1].Name != "weekly-copy" {
		t.Fatalf("list = %+v", list)
	}
	if list[0].InvalidReason != "" {
		t.Errorf("weekly should be valid: %+v", list[0])
	}
	mismatch := list[1]
	if !strings.Contains(mismatch.InvalidReason, "does not match") || mismatch.Description != "" || mismatch.FileCount != 1 {
		t.Errorf("weekly-copy should carry the name mismatch: %+v", mismatch)
	}
	skill, err := store.Get(testUser, "weekly-copy")
	if err != nil {
		t.Fatal(err)
	}
	if skill.InvalidReason != mismatch.InvalidReason || skill.Description != "" || skill.Manifest == nil {
		t.Errorf("Get of the mismatched skill = %+v", skill)
	}
}

func TestParseUploadedZip(t *testing.T) {
	good := manifest("weekly", "d")
	t.Run("wrapper folder is stripped and junk is ignored", func(t *testing.T) {
		uploaded, err := parseUploadedZip(makeZip(t,
			zipItem{name: "weekly/"},
			zipItem{name: "weekly/SKILL.md", content: string(good)},
			zipItem{name: `weekly\scripts\run.py`, content: "x"},
			zipItem{name: "weekly/.DS_Store", content: "x"},
			zipItem{name: "__MACOSX/weekly/._SKILL.md", content: "x"},
			zipItem{name: "weekly/assets/Thumbs.db", content: "x"},
		))
		if err != nil {
			t.Fatal(err)
		}
		if uploaded.name != "weekly" {
			t.Errorf("name = %q", uploaded.name)
		}
		paths := []string{}
		for _, file := range uploaded.files {
			paths = append(paths, file.path)
		}
		if !slices.Equal(paths, []string{ManifestFileName, "scripts/run.py"}) {
			t.Errorf("paths = %v", paths)
		}
		if len(uploaded.ignored) != 3 {
			t.Errorf("ignored = %v", uploaded.ignored)
		}
	})
	t.Run("root manifest is not stripped", func(t *testing.T) {
		uploaded, err := parseUploadedZip(makeZip(t,
			zipItem{name: "SKILL.md", content: string(good)},
			zipItem{name: "docs/a.md", content: "x"},
		))
		if err != nil {
			t.Fatal(err)
		}
		if uploaded.files[1].path != "docs/a.md" {
			t.Errorf("paths = %v", uploaded.files)
		}
	})
	t.Run("two top-level folders are not stripped", func(t *testing.T) {
		// 包みは「全項目が同じ1つのフォルダの下」のときだけ剥がす。トップに2つあれば剥がさず、
		// ルートに SKILL.md が無いとして拒否する（片方の SKILL.md を勝手に選ばない）
		for label, items := range map[string][]zipItem{
			"manifest in one of them": {{name: "a/SKILL.md", content: string(good)}, {name: "b/README.md", content: "x"}},
			"manifest in both":        {{name: "a/SKILL.md", content: string(good)}, {name: "b/SKILL.md", content: string(good)}},
		} {
			_, err := parseUploadedZip(makeZip(t, items...))
			if !errors.Is(err, ErrInvalidZip) || !strings.Contains(DescribeError(err), "missing at the top level") {
				t.Errorf("%s: err = %v, want ErrInvalidZip (SKILL.md missing at the top level)", label, err)
			}
		}
	})
	t.Run("entries starting with ./ are ignored as dot-prefixed (current behavior)", func(t *testing.T) {
		// "./SKILL.md" は要素 "." がドット始まりとして無視される。"./" を剥がす仕様ではなく、
		// 全項目がそうなら「the zip has no files」、一部なら残りだけで判定される。現状の挙動の固定
		cases := []struct {
			label  string
			items  []zipItem
			detail string
		}{
			{"all entries", []zipItem{{name: "./SKILL.md", content: string(good)}, {name: "./docs/a.md", content: "x"}}, "the zip has no files"},
			{"manifest only", []zipItem{{name: "./SKILL.md", content: string(good)}, {name: "docs/a.md", content: "x"}}, "missing at the top level"},
		}
		for _, c := range cases {
			_, err := parseUploadedZip(makeZip(t, c.items...))
			if !errors.Is(err, ErrInvalidZip) || !strings.Contains(DescribeError(err), c.detail) {
				t.Errorf("%s: err = %v, want ErrInvalidZip containing %q", c.label, err, c.detail)
			}
		}
	})

	rejects := []struct {
		label string
		items []zipItem
		kind  error
	}{
		{"not a zip", nil, ErrInvalidZip},
		{"zip slip", []zipItem{{name: "SKILL.md", content: string(good)}, {name: "../evil.sh", content: "x"}}, ErrInvalidZip},
		{"absolute", []zipItem{{name: "SKILL.md", content: string(good)}, {name: "/etc/passwd", content: "x"}}, ErrInvalidZip},
		{"drive letter", []zipItem{{name: "SKILL.md", content: string(good)}, {name: "C:/x", content: "x"}}, ErrInvalidZip},
		{"symlink", []zipItem{{name: "SKILL.md", content: string(good)}, {name: "link", content: "/etc", mode: fs.ModeSymlink | 0o777}}, ErrInvalidZip},
		{"no manifest", []zipItem{{name: "weekly/README.md", content: "x"}}, ErrInvalidZip},
		{"empty", []zipItem{{name: "weekly/"}}, ErrInvalidZip},
		{"non-ascii name", []zipItem{{name: "SKILL.md", content: string(good)}, {name: "メモ.md", content: "x"}}, ErrInvalidZip},
		{"case duplicate", []zipItem{{name: "SKILL.md", content: string(good)}, {name: "a.md", content: "x"}, {name: "A.md", content: "y"}}, ErrInvalidZip},
		{"file and folder", []zipItem{{name: "SKILL.md", content: string(good)}, {name: "a", content: "x"}, {name: "a/b", content: "y"}}, ErrInvalidZip},
		{"broken manifest", []zipItem{{name: "SKILL.md", content: "# no frontmatter"}}, ErrInvalidFrontmatter},
	}
	for _, c := range rejects {
		data := []byte("this is not a zip")
		if c.items != nil {
			data = makeZip(t, c.items...)
		}
		if _, err := parseUploadedZip(data); !errors.Is(err, c.kind) {
			t.Errorf("%s: err = %v, want %v", c.label, err, c.kind)
		}
	}
}

func TestPlanReplaceAndReplace(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	first := makeZip(t,
		zipItem{name: "weekly/SKILL.md", content: string(manifest("weekly", "v1"))},
		zipItem{name: "weekly/references/keep.md", content: "keep"},
		zipItem{name: "weekly/references/change.md", content: "before"},
		zipItem{name: "weekly/scripts/remove.py", content: "x"},
	)
	plan, err := store.PlanReplace(testUser, first)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.IsNew || len(plan.Added) != 4 || len(plan.Removed) != 0 || len(plan.Changed) != 0 {
		t.Fatalf("first plan = %+v", plan)
	}
	// 計画だけでは何も書かない
	if list, _ := store.List(testUser); len(list) != 0 {
		t.Fatalf("PlanReplace wrote something: %v", list)
	}
	if _, err := store.Replace(ctx, testUser, first); err != nil {
		t.Fatal(err)
	}

	// 手で置いたドットファイルも置き換えで消える（資料に書いてある挙動）
	skillDir := filepath.Join(store.Root(), testUser, "weekly")
	if err := os.WriteFile(filepath.Join(skillDir, ".notes"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// 前回の中断で残った作業用ディレクトリ
	for _, dir := range []string{"_tmp-leftover", "_old-leftover"} {
		if err := os.MkdirAll(filepath.Join(store.Root(), testUser, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	second := makeZip(t,
		zipItem{name: "SKILL.md", content: string(manifest("weekly", "v2"))},
		zipItem{name: "references/keep.md", content: "keep"},
		zipItem{name: "references/change.md", content: "after"},
		zipItem{name: "assets/new.png", content: "\x89PNG\x00"},
		zipItem{name: ".DS_Store", content: "x"},
	)
	plan, err = store.PlanReplace(testUser, second)
	if err != nil {
		t.Fatal(err)
	}
	want := &ReplacePlan{
		Name:    "weekly",
		Added:   []string{"assets/new.png"},
		Removed: []string{"scripts/remove.py"},
		Changed: []string{ManifestFileName, "references/change.md"},
		Ignored: []string{".DS_Store"},
	}
	if plan.IsNew || !slices.Equal(plan.Added, want.Added) || !slices.Equal(plan.Removed, want.Removed) ||
		!slices.Equal(plan.Changed, want.Changed) || !slices.Equal(plan.Ignored, want.Ignored) {
		t.Fatalf("second plan = %+v, want %+v", plan, want)
	}
	applied, err := store.Replace(ctx, testUser, second)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(applied.Changed, want.Changed) {
		t.Errorf("applied plan = %+v", applied)
	}

	skill, err := store.Get(testUser, "weekly")
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{}
	for _, file := range skill.Files {
		paths = append(paths, file.Path)
	}
	if !slices.Equal(paths, []string{ManifestFileName, "assets/new.png", "references/change.md", "references/keep.md"}) {
		t.Errorf("paths after replace = %v", paths)
	}
	if skill.Description != "v2" {
		t.Errorf("description = %q", skill.Description)
	}
	if _, err := os.Stat(filepath.Join(skillDir, ".notes")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("dot file survived replace: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(store.Root(), testUser))
	if len(entries) != 1 || entries[0].Name() != "weekly" {
		t.Errorf("user dir after replace = %v", entries)
	}
}

func TestBuildZipRoundTrip(t *testing.T) {
	store := newTestStore(t)
	mustWrite(t, store, testUser, "weekly", ManifestFileName, manifest("weekly", "d"), "")
	mustWrite(t, store, testUser, "weekly", "scripts/run.py", []byte("print('あ')\n"), "")

	fileName, data, err := store.BuildZip(testUser, "weekly")
	if err != nil {
		t.Fatal(err)
	}
	if fileName != "weekly.zip" {
		t.Errorf("file name = %q", fileName)
	}
	// ダウンロードした zip をそのまま上げ直しても何も変わらない
	plan, err := store.PlanReplace(testUser, data)
	if err != nil {
		t.Fatal(err)
	}
	if plan.IsNew || len(plan.Added)+len(plan.Removed)+len(plan.Changed) != 0 {
		t.Errorf("round trip plan = %+v", plan)
	}
	if _, _, err := store.BuildZip(testUser, "missing"); !errors.Is(err, ErrSkillNotFound) {
		t.Errorf("missing skill: %v", err)
	}
}

func TestReplaceKeepsExistingSkillWhenSwapFails(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("開いているファイルがあるとフォルダの rename が失敗するのは Windows だけ")
	}
	store := newTestStore(t)
	ctx := context.Background()
	mustWrite(t, store, testUser, "weekly", ManifestFileName, manifest("weekly", "v1"), "")
	opened, err := os.Open(filepath.Join(store.Root(), testUser, "weekly", ManifestFileName))
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()

	_, err = store.Replace(ctx, testUser, makeZip(t, zipItem{name: "SKILL.md", content: string(manifest("weekly", "v2"))}))
	if err == nil {
		t.Fatal("Replace succeeded while a file was open")
	}
	skill, err := store.Get(testUser, "weekly")
	if err != nil || skill.Description != "v1" {
		t.Errorf("existing skill changed: %+v, %v", skill, err)
	}
	entries, _ := os.ReadDir(filepath.Join(store.Root(), testUser))
	if len(entries) != 1 {
		t.Errorf("leftovers after failed replace: %v", entries)
	}
}

// makeTamperedZip は無圧縮の zip を作り、tamperName の項目の中身だけを同じ長さの別のバイト列へ
// 差し替える（ローカルヘッダ・セントラルディレクトリの CRC とサイズは元のまま）。
// zip としては開けるが、その項目を読み切ると archive/zip が CRC の食い違いを返す。
// tamperName が空なら差し替えない（同じ作り方の zip が正常に通ることの対照用）。
func makeTamperedZip(t *testing.T, tamperName string, items ...zipItem) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	var original []byte
	for _, item := range items {
		w, err := writer.CreateHeader(&zip.FileHeader{Name: item.name, Method: zip.Store})
		if err != nil {
			t.Fatalf("CreateHeader(%s): %v", item.name, err)
		}
		if _, err := w.Write([]byte(item.content)); err != nil {
			t.Fatalf("Write(%s): %v", item.name, err)
		}
		if item.name == tamperName {
			original = []byte(item.content)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	data := buf.Bytes()
	if tamperName == "" {
		return data
	}
	if len(original) == 0 {
		t.Fatalf("no entry named %q with content to tamper", tamperName)
	}
	at := bytes.Index(data, original)
	if at < 0 {
		t.Fatalf("stored content of %q not found in the archive", tamperName)
	}
	for i := range original {
		data[at+i] = original[i] ^ 0x55
	}
	return data
}

// 項目の中身がヘッダの CRC と食い違う zip は、SKILL.md（全文を読む経路）でも
// 付属ファイル（revision のために流し読みする経路）でも ErrInvalidZip で拒否する。
// archive/zip は開くときには気づかず読み切ったときに ErrChecksum を返すので、
// 読み出しの戻り値を見落とすと壊れた中身がそのまま revision を持って取り込まれる。
func TestParseUploadedZipRejectsEntryWithMismatchedCRC(t *testing.T) {
	items := []zipItem{
		{name: "SKILL.md", content: string(manifest("weekly", "d"))},
		{name: "references/notes.md", content: "notes body that will be tampered with\n"},
	}
	if _, err := parseUploadedZip(makeTamperedZip(t, "", items...)); err != nil {
		t.Fatalf("untampered stored zip should parse: %v", err)
	}
	for label, tamperName := range map[string]string{
		"manifest (read in full)":  "SKILL.md",
		"attached file (streamed)": "references/notes.md",
	} {
		_, err := parseUploadedZip(makeTamperedZip(t, tamperName, items...))
		if !errors.Is(err, ErrInvalidZip) {
			t.Errorf("%s: err = %v, want ErrInvalidZip", label, err)
			continue
		}
		if detail := DescribeError(err); !strings.Contains(detail, "could not be read") || !strings.Contains(detail, tamperName) {
			t.Errorf("%s: detail = %q, want the read failure and the path", label, detail)
		}
	}
}

// recordCapturingHandler は流れてきたレコードを控えるだけの slog.Handler。
// 出力の整形（TextHandler は改行を含む値を自分で引用する）に頼らず、渡された値そのものを見る。
type recordCapturingHandler struct {
	records []slog.Record
}

func (h *recordCapturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *recordCapturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r.Clone())
	return nil
}
func (h *recordCapturingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *recordCapturingHandler) WithGroup(_ string) slog.Handler      { return h }

// removeQuietly の後始末ログは、改行を含むパスでも1行に収まる値（%q で包んだもの）を渡す。
// 素の値を渡すと、行単位で読む・集計するログの仕組みが改行で行を切り、後半が別の行として混ざる。
func TestRemoveQuietlyQuotesPathInLog(t *testing.T) {
	captured := &recordCapturingHandler{}
	original := slog.Default()
	slog.SetDefault(slog.New(captured))
	t.Cleanup(func() { slog.SetDefault(original) })

	// 末尾の要素が "." のパスは os.RemoveAll がファイルシステムに触らずに EINVAL を返す
	// （rmdir(".") と同じ扱い。Windows / Unix 共通）ので、存在しないパスでも確実に「削除に失敗した」経路へ入る
	path := "first line\nsecond line" + string(filepath.Separator) + "."
	removeQuietly(context.Background(), path)

	var record *slog.Record
	for i := range captured.records {
		if captured.records[i].Message == "error at remove leftover skill path" {
			record = &captured.records[i]
		}
	}
	if record == nil {
		t.Fatalf("removeQuietly did not log the failure: %d records", len(captured.records))
	}
	if record.Level != gkill_log.Warn {
		t.Errorf("level = %v, want Warn", record.Level)
	}
	values := map[string]string{}
	record.Attrs(func(attr slog.Attr) bool {
		values[attr.Key] = attr.Value.String()
		return true
	})
	if values["path"] != strconv.Quote(path) {
		t.Errorf("path attr = %q, want %q", values["path"], strconv.Quote(path))
	}
	for _, key := range []string{"path", "error"} {
		if value, ok := values[key]; !ok || strings.Contains(value, "\n") {
			t.Errorf("%s attr = %q (present=%v), must be one line", key, value, ok)
		}
	}
}
