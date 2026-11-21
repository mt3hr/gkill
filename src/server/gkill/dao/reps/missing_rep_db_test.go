package reps

// 消えた rep のDBファイルを gkill が空のDBとして作り直さないことの回帰テスト。
//
// 読むだけの rep のファイルは端末同期で運ばれてくるもので、gkill が作ってよいのは書き込み rep を
// 初めて置くときだけ。消えたファイルを開き直して空のDBを作ると、その空のDBが「新しいファイル」として
// 同期で全保存側の中身のある版を上書きする。
// documents/adr/0223-never-recreate-a-missing-rep-db.md

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

// assertNotCreated は filename がまだ無いことを確かめる。
func assertNotCreated(t *testing.T, filename string) {
	t.Helper()
	if _, err := os.Stat(filename); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s が作られている（stat err = %v）。消えた rep のファイルを作り直してはいけない", filename, err)
	}
}

// 読むだけの rep（fullConnect=false）のコンストラクタは、無いファイルを作らずにエラーを返す。
// 書き込み rep（fullConnect=true）は、初めて置くときに作れる（今までどおり）。
func TestRepConstructors_CreateMissingFileOnlyForWriteRep(t *testing.T) {
	ctx := context.Background()
	type closer interface {
		Close(ctx context.Context) error
	}
	cases := []struct {
		name string
		open func(filename string, fullConnect bool) (closer, error)
	}{
		{"kmemo", func(f string, c bool) (closer, error) { return NewKmemoRepositorySQLite3Impl(ctx, f, c) }},
		{"kc", func(f string, c bool) (closer, error) { return NewKCRepositorySQLite3Impl(ctx, f, c) }},
		{"urlog", func(f string, c bool) (closer, error) { return NewURLogRepositorySQLite3Impl(ctx, f, c) }},
		{"timeis", func(f string, c bool) (closer, error) { return NewTimeIsRepositorySQLite3Impl(ctx, f, c) }},
		{"mi", func(f string, c bool) (closer, error) { return NewMiRepositorySQLite3Impl(ctx, f, c) }},
		{"nlog", func(f string, c bool) (closer, error) { return NewNlogRepositorySQLite3Impl(ctx, f, c) }},
		{"lantana", func(f string, c bool) (closer, error) { return NewLantanaRepositorySQLite3Impl(ctx, f, c) }},
		{"tag", func(f string, c bool) (closer, error) { return NewTagRepositorySQLite3Impl(ctx, f, c) }},
		{"text", func(f string, c bool) (closer, error) { return NewTextRepositorySQLite3Impl(ctx, f, c) }},
		{"notification", func(f string, c bool) (closer, error) { return NewNotificationRepositorySQLite3Impl(ctx, f, c) }},
		{"rekyou", func(f string, c bool) (closer, error) { return NewReKyouRepositorySQLite3Impl(ctx, f, c, nil) }},
		{"mirekyou", func(f string, c bool) (closer, error) { return NewMiReKyouRepositorySQLite3Impl(ctx, f, c, nil) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			missing := filepath.Join(t.TempDir(), "missing.db")
			if rep, err := c.open(missing, false); err == nil {
				_ = rep.Close(ctx)
				t.Fatalf("読むだけの rep が無いファイルで作れてしまった")
			} else if !errors.Is(err, os.ErrNotExist) {
				t.Errorf("err = %v, want os.ErrNotExist を包んだエラー", err)
			}
			assertNotCreated(t, missing)

			created := filepath.Join(t.TempDir(), "created.db")
			rep, err := c.open(created, true)
			if err != nil {
				t.Fatalf("書き込み rep を初めて置けない: %v", err)
			}
			_ = rep.Close(ctx)
			if _, err := os.Stat(created); err != nil {
				t.Errorf("書き込み rep のファイルが作られていない: %v", err)
			}
		})
	}
}

// 読むだけのディレクトリ rep も、ID表（.gkill/gkill_id.db）が無ければ作らない。
func TestIDFDirRep_ReadOnlyDoesNotCreateMissingIDDB(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	idDB := filepath.Join(dir, ".gkill", "gkill_id.db")
	if err := os.MkdirAll(filepath.Dir(idDB), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	ignore := []string{}
	if rep, err := NewIDFDirRep(ctx, "testuser", dir, idDB, false, mux.NewRouter(), false, &ignore, nil); err == nil {
		_ = rep.Close(ctx)
		t.Fatalf("読むだけのディレクトリ rep が、ID表の無いディレクトリで作れてしまった")
	}
	assertNotCreated(t, idDB)
}

// 読み込んだあとに読むだけの rep のファイルが消えたら、検索はエラーを返し、ファイルは作り直さない。
// 以前はここで空のDBができ、利用者が消したファイルが「新しいファイル」として同期に乗った。
func TestReadOnlyRep_DeletedAfterLoadIsNotRecreated(t *testing.T) {
	ctx := context.Background()
	type finder interface {
		FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error)
		Close(ctx context.Context) error
	}
	cases := []struct {
		name string
		open func(filename string, fullConnect bool) (finder, error)
	}{
		{"kmemo", func(f string, c bool) (finder, error) { return NewKmemoRepositorySQLite3Impl(ctx, f, c) }},
		{"kc", func(f string, c bool) (finder, error) { return NewKCRepositorySQLite3Impl(ctx, f, c) }},
		{"urlog", func(f string, c bool) (finder, error) { return NewURLogRepositorySQLite3Impl(ctx, f, c) }},
		{"timeis", func(f string, c bool) (finder, error) { return NewTimeIsRepositorySQLite3Impl(ctx, f, c) }},
		{"mi", func(f string, c bool) (finder, error) { return NewMiRepositorySQLite3Impl(ctx, f, c) }},
		{"nlog", func(f string, c bool) (finder, error) { return NewNlogRepositorySQLite3Impl(ctx, f, c) }},
		{"lantana", func(f string, c bool) (finder, error) { return NewLantanaRepositorySQLite3Impl(ctx, f, c) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "rep.db")
			writer, err := c.open(filename, true)
			if err != nil {
				t.Fatalf("元のファイルを作れない: %v", err)
			}
			_ = writer.Close(ctx)

			reader, err := c.open(filename, false)
			if err != nil {
				t.Fatalf("読むだけの rep を開けない: %v", err)
			}
			defer func() { _ = reader.Close(ctx) }()

			if err := os.Remove(filename); err != nil {
				t.Fatalf("元のファイルを消せない: %v", err)
			}
			if _, err := reader.FindKyous(ctx, &find.FindQuery{}); err == nil {
				t.Errorf("消えたファイルの検索がエラーにならない")
			}
			assertNotCreated(t, filename)
		})
	}
}

// ローカルキャッシュ版（--cache_reps_local=true）は、元のファイルが消えていたら UpdateCache をエラーにし、
// 使えていたローカルキャッシュを閉じて消さない。元のファイルも作り直さない。
func TestLocalCachedRep_UpdateCacheWithMissingOriginalKeepsCache(t *testing.T) {
	useTempLocalRepCacheDir(t)
	ctx := context.Background()
	filename := newLocalCachedTestDBFile(t, "TimeIs.db", func(ctx context.Context, filename string) error {
		rep, err := NewTimeIsRepositorySQLite3Impl(ctx, filename, true)
		if err != nil {
			return err
		}
		return rep.Close(ctx)
	})
	rep, err := NewTimeIsRepositorySQLite3ImplLocalCached(ctx, "testuser", filename, false)
	if err != nil {
		t.Fatalf("ローカルキャッシュ版を開けない: %v", err)
	}
	defer func() { _ = rep.Close(ctx) }()

	if err := os.Remove(filename); err != nil {
		t.Fatalf("元のファイルを消せない: %v", err)
	}
	if err := rep.UpdateCache(ctx); err == nil {
		t.Errorf("元のファイルが無いのに UpdateCache がエラーにならない")
	}
	assertNotCreated(t, filename)
	if _, err := rep.FindKyous(ctx, &find.FindQuery{}); err != nil {
		t.Errorf("UpdateCache の失敗でローカルキャッシュまで使えなくなった: %v", err)
	}
}
