package reps

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// 派生キャッシュ（thumb / video / zip）のディレクトリは
// caches/{種別}/{userID}/{repName} でなければならない。
//
// rep名は filepath.Base(contentDir) で決まり、利用者間で一意ではない。
// 利用者IDの階層が抜けると、別の利用者の同名repとキャッシュが混ざる。
func TestDerivedCacheDirForUser(t *testing.T) {
	tmp := t.TempDir()
	orig := gkill_options.CacheDir
	gkill_options.CacheDir = tmp
	t.Cleanup(func() { gkill_options.CacheDir = orig })

	contentDir := filepath.Join("C:", "data", "testuser", "Archive_20141115")

	for _, cacheName := range []string{"thumb_cache", "video_cache", "zip_cache"} {
		got := derivedCacheDirForUser(cacheName, "testuser_all", contentDir)
		want := filepath.Join(tmp, cacheName, "testuser_all", "Archive_20141115")
		if got != want {
			t.Errorf("%s: got %q, want %q", cacheName, got, want)
		}
	}
}

func TestDerivedCacheDirForUserSeparatesUsers(t *testing.T) {
	tmp := t.TempDir()
	orig := gkill_options.CacheDir
	gkill_options.CacheDir = tmp
	t.Cleanup(func() { gkill_options.CacheDir = orig })

	// 別々のディレクトリだが rep名（末尾）は同じ、という実際にある構成
	aliceDir := filepath.Join(tmp, "alice", "Photos")
	bobDir := filepath.Join(tmp, "bob", "Photos")

	alice := derivedCacheDirForUser("thumb_cache", "alice", aliceDir)
	bob := derivedCacheDirForUser("thumb_cache", "bob", bobDir)

	if alice == bob {
		t.Fatalf("同名repを持つ別利用者が同じキャッシュディレクトリを指している: %q", alice)
	}
	if filepath.Base(alice) != filepath.Base(bob) {
		t.Fatalf("rep名の部分は同じであるべき: %q vs %q", alice, bob)
	}
}

func TestDerivedCacheDirForUserRejectsUnsafeUserID(t *testing.T) {
	tmp := t.TempDir()
	orig := gkill_options.CacheDir
	gkill_options.CacheDir = tmp
	t.Cleanup(func() { gkill_options.CacheDir = orig })

	root := filepath.Join(tmp, "thumb_cache")

	// 空・パス区切り・上位参照はキャッシュルートの外へ出てはいけない。
	// いずれも実在の利用者と衝突しない固定名へ寄せる。
	for _, userID := range []string{"", ".", "..", "../../etc", `a\b`, "a/b"} {
		got := derivedCacheDirForUser("thumb_cache", userID, filepath.Join(tmp, "Photos"))
		want := filepath.Join(root, noUserCacheDirName, "Photos")
		if got != want {
			t.Errorf("userID=%q: got %q, want %q", userID, got, want)
		}
	}
}

// サムネ・互換動画のサーバが、実際に利用者ごとのディレクトリを掴んでいることを確認する。
// コンストラクタで userID を渡し忘れても型は通ってしまうので、ここで固定する。
func TestFileServersUsePerUserCacheDir(t *testing.T) {
	tmp := t.TempDir()
	orig := gkill_options.CacheDir
	gkill_options.CacheDir = tmp
	t.Cleanup(func() { gkill_options.CacheDir = orig })

	contentDir := filepath.Join(tmp, "Photos")
	if err := os.MkdirAll(contentDir, os.ModePerm); err != nil {
		t.Fatalf("failed to create content dir: %v", err)
	}
	base := http.NotFoundHandler()

	thumb, ok := NewThumbFileServer("testuser_all", contentDir, base).(*thumbFileServer)
	if !ok {
		t.Fatal("NewThumbFileServer did not return *thumbFileServer")
	}
	if want := filepath.Join(tmp, "thumb_cache", "testuser_all", "Photos"); thumb.cacheDir != want {
		t.Errorf("thumb cacheDir = %q, want %q", thumb.cacheDir, want)
	}

	video, ok := NewVideoFileServer("testuser_all", contentDir, base).(*IDFVideoFileServer)
	if !ok {
		t.Fatal("NewVideoFileServer did not return *IDFVideoFileServer")
	}
	if want := filepath.Join(tmp, "video_cache", "testuser_all", "Photos"); video.cacheDir != want {
		t.Errorf("video cacheDir = %q, want %q", video.cacheDir, want)
	}
}
