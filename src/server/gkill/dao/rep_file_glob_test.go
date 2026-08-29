package dao

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mattn/go-zglob"
)

// setupRepFilePatternTree は展開の検証用に、実運用のrep置き場と同じ形の木を作る。
// rep ディレクトリの中には .gkill と実ファイルが入っていて、
// 「再帰すると余計なものに当たるが、1階層の * では当たらない」状況を再現する。
func setupRepFilePatternTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	dirs := []string{
		"Archive_All_20200101/.gkill",
		"Archive_All_20210101/.gkill",
		"Archive_Sub/nested/Archive_Deep",
		"Photo_All_20200101",
		"other",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), os.ModePerm); err != nil {
			t.Fatalf("error at make directory %s: %v", dir, err)
		}
	}

	files := []string{
		"Archive_All_20200101/.gkill/gkill_id.db",
		"Archive_All_20200101/photo.jpg",
		"KC_X.db",
		"KC_Y.db",
		"KC_.db",
		"KC_X.dbx",
		"Kmemo.db",
		"literal[a].db",
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(file)), []byte("x"), os.ModePerm); err != nil {
			t.Fatalf("error at write file %s: %v", file, err)
		}
	}
	return root
}

func toSlashSorted(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, p := range paths {
		normalized = append(normalized, filepath.ToSlash(filepath.Clean(p)))
	}
	slices.Sort(normalized)
	return normalized
}

// TestExpandRepFilePatternMatchesZglob は、非再帰展開が go-zglob と同じ集合を返すことを固定する。
// これが崩れると、repが黙って増えたり消えたりする（どちらもエラーにならない）。
func TestExpandRepFilePatternMatchesZglob(t *testing.T) {
	root := filepath.ToSlash(setupRepFilePatternTree(t))
	ctx := context.Background()

	patterns := []string{
		root + "/Archive_*",
		root + "/Photo_*",
		root + "/KC_*.db",
		root + "/Kmemo.db",
		root + "/*",
		root + "/NotExist_*",
		root + "/NotExist.db",
		root + "/**/Archive_Deep",
	}

	for _, pattern := range patterns {
		t.Run(pattern, func(t *testing.T) {
			want, err := zglob.Glob(pattern)
			if err != nil && !strings.Contains(err.Error(), "file does not exist") {
				t.Fatalf("error at zglob.Glob %s: %v", pattern, err)
			}
			got := expandRepFilePattern(ctx, pattern)

			wantSorted, gotSorted := toSlashSorted(want), toSlashSorted(got)
			if !slices.Equal(wantSorted, gotSorted) {
				t.Errorf("expandRepFilePattern(%s) = %v, zglob.Glob = %v", pattern, gotSorted, wantSorted)
			}
		})
	}
}

// TestExpandRepFilePatternDoesNotRecurse は、* が / をまたがないことを固定する。
// またいでしまうと、rep置き場の中の入れ子ディレクトリまでrepとして読み込まれる。
func TestExpandRepFilePatternDoesNotRecurse(t *testing.T) {
	root := filepath.ToSlash(setupRepFilePatternTree(t))

	got := toSlashSorted(expandRepFilePattern(context.Background(), root+"/Archive_*"))
	want := toSlashSorted([]string{
		root + "/Archive_All_20200101",
		root + "/Archive_All_20210101",
		root + "/Archive_Sub",
	})
	if !slices.Equal(got, want) {
		t.Errorf("expandRepFilePattern = %v, want %v", got, want)
	}
}

// TestExpandRepFilePatternTreatsBracketAsLiteral は
// [ をメタ文字にしないことを固定する（? は Windows でファイル名に使えないので matchRepFileSegment 側で見る）。go-zglob は "*{" だけを glob と見なすので、
// filepath.Match へ渡すと意味論が変わり、実在しないファイルにマッチしたことになる。
func TestExpandRepFilePatternTreatsBracketAsLiteral(t *testing.T) {
	root := filepath.ToSlash(setupRepFilePatternTree(t))
	ctx := context.Background()

	cases := []struct {
		pattern string
		want    []string
	}{
		{root + "/literal[a].db", []string{root + "/literal[a].db"}},
	}
	for _, c := range cases {
		got := toSlashSorted(expandRepFilePattern(ctx, c.pattern))
		if !slices.Equal(got, toSlashSorted(c.want)) {
			t.Errorf("expandRepFilePattern(%s) = %v, want %v", c.pattern, got, c.want)
		}
	}
}

// TestExpandRepFilePatternCollapsesDuplicatedSeparators は区切りの二連を畳むことを固定する。
// 設定に使われている環境変数のうち1つは末尾に区切りを持つ値で、展開すると必ずこの形になる。
func TestExpandRepFilePatternCollapsesDuplicatedSeparators(t *testing.T) {
	root := filepath.ToSlash(setupRepFilePatternTree(t))

	got := toSlashSorted(expandRepFilePattern(context.Background(), root+"//KC_*.db"))
	want := toSlashSorted([]string{root + "/KC_.db", root + "/KC_X.db", root + "/KC_Y.db"})
	if !slices.Equal(got, want) {
		t.Errorf("expandRepFilePattern = %v, want %v", got, want)
	}
}

// TestExpandRepFilePatternSkipsFilesystemRoot は、環境変数が未設定のときに
// ファイルシステム全体を走査しないことを固定する。
// go-zglob はこの形（"/Archive_*"）でドライブ全体を歩いていた。
func TestExpandRepFilePatternSkipsFilesystemRoot(t *testing.T) {
	for _, pattern := range []string{"/Archive_*", "C:/Archive_*"} {
		if got := expandRepFilePattern(context.Background(), pattern); len(got) != 0 {
			t.Errorf("expandRepFilePattern(%s) = %v, want empty", pattern, got)
		}
	}
}

// TestExpandRepFilePatternMissingParent は親ディレクトリが無いときに0件を返すことを固定する。
func TestExpandRepFilePatternMissingParent(t *testing.T) {
	root := filepath.ToSlash(t.TempDir())
	if got := expandRepFilePattern(context.Background(), root+"/not_exist_dir/KC_*.db"); len(got) != 0 {
		t.Errorf("expandRepFilePattern = %v, want empty", got)
	}
}

// TestRepFilePatternNeedsWalk は go-zglob へ落とす条件を固定する。
// 落とし忘れると ** を含む設定が黙って0件になる。
func TestRepFilePatternNeedsWalk(t *testing.T) {
	cases := map[string]bool{
		"/a/b/Archive_*":  false,
		"/a/b/KC_*.db":    false,
		"/a/b/Kmemo.db":   false,
		"/a/b/*":          false,
		"/a/b/**":         true,
		"/a/**/c":         true,
		"/a/*/c":          true,
		"/a/{x,y}/c":      true,
		"/a/!(x)/c":       true,
		"/a/b/literal?.d": false,
		"/a/b/lit[a].db":  false,
	}
	for pattern, want := range cases {
		if got := repFilePatternNeedsWalk(pattern); got != want {
			t.Errorf("repFilePatternNeedsWalk(%s) = %v, want %v", pattern, got, want)
		}
	}
}

// TestMatchRepFileSegmentIgnoreCase は大小無視の有無で結果が変わることを固定する。
// go-zglob は windows / darwin で大小を無視しており、綴りを間違えた設定を救っている。
func TestMatchRepFileSegmentIgnoreCase(t *testing.T) {
	original := repFilePatternIgnoreCase
	t.Cleanup(func() { repFilePatternIgnoreCase = original })

	repFilePatternIgnoreCase = true
	if !matchRepFileSegment("archive_*", "Archive_All_20200101") {
		t.Error("matchRepFileSegment should ignore case when repFilePatternIgnoreCase is true")
	}

	repFilePatternIgnoreCase = false
	if matchRepFileSegment("archive_*", "Archive_All_20200101") {
		t.Error("matchRepFileSegment should respect case when repFilePatternIgnoreCase is false")
	}
	if !matchRepFileSegment("Archive_*", "Archive_All_20200101") {
		t.Error("matchRepFileSegment should match the same case")
	}
}

// TestMatchRepFileSegment は1セグメント照合の境界を固定する。
func TestMatchRepFileSegment(t *testing.T) {
	original := repFilePatternIgnoreCase
	t.Cleanup(func() { repFilePatternIgnoreCase = original })
	repFilePatternIgnoreCase = false

	cases := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"KC_*.db", "KC_X.db", true},
		{"KC_*.db", "KC_.db", true},
		{"KC_*.db", "KC_X.dbx", false},
		{"KC_*.db", "Kmemo.db", false},
		{"*", "anything", true},
		{"*", "", true},
		{"Photo_*", "Photo_All", true},
		{"Photo_*", "PhotoOff_All", false},
		{"Kmemo.db", "Kmemo.db", true},
		{"Kmemo.db", "Kmemo.db2", false},
		{"a*b*c", "axxbyyc", true},
		{"a*b*c", "axxc", false},
		{"*.db", ".db", true},
		// ? と [ はメタ文字ではない（go-zglob の glob 判定が "*{" だけを見るため）
		{"literal?name.db", "literal?name.db", true},
		{"literal?name.db", "literalXname.db", false},
		{"lit[a].db", "lit[a].db", true},
		{"lit[a].db", "lita.db", false},
	}
	for _, c := range cases {
		if got := matchRepFileSegment(c.pattern, c.name); got != c.want {
			t.Errorf("matchRepFileSegment(%q, %q) = %v, want %v", c.pattern, c.name, got, c.want)
		}
	}
}
