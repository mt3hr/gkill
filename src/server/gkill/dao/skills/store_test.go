package skills

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
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

func TestBuildManifestRoundTrip(t *testing.T) {
	description := "週次ダッシュボード: \"引用\" と # 記号、改行なし"
	content, err := BuildManifest("weekly", description, "# 手順\n1. 取る")
	if err != nil {
		t.Fatalf("BuildManifest: %v", err)
	}
	parsed, err := ParseManifest(content, "weekly")
	if err != nil {
		t.Fatalf("ParseManifest: %v\n%s", err, content)
	}
	if parsed.Description != description {
		t.Errorf("description = %q, want %q", parsed.Description, description)
	}
	if !strings.HasSuffix(string(content), "# 手順\n1. 取る\n") {
		t.Errorf("body not preserved: %q", content)
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

func TestUserDirCaseConflictIsRejected(t *testing.T) {
	store := newTestStore(t)
	mustWrite(t, store, "TestUser", "weekly", ManifestFileName, manifest("weekly", "d"), "")
	// 大文字小文字だけ違う利用者は、別の利用者のディレクトリを読まない
	if _, err := store.Get("testuser", "weekly"); err == nil {
		t.Errorf("case-different user could read the skill")
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
