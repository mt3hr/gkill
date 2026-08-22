package reps

import (
	"os"
	"path/filepath"
	"testing"
)

// SecureJoin は「rootDir の真下」だけを許可する。
// rootDir 自身を返してしまうと、呼び出し元（アップロードの保存先・サムネ/動画の配信元・
// ZIP の展開先・ローカルキャッシュDBのパス）がディレクトリをファイルとして扱うことになる。
//
// 「rootDir 自身も許可」だった頃は、CodeQL の go/path-injection が
// その分岐（full == root）をバリアとして認識できず、
// handle_upload_files.go の3箇所が実際には安全なのにアラートになっていた。
func TestSecureJoin(t *testing.T) {
	root := filepath.Clean(filepath.Join(os.TempDir(), "gkill_secure_join_root"))
	sep := string(os.PathSeparator)

	t.Run("真下のパスは許可する", func(t *testing.T) {
		for _, rel := range []string{"a.txt", "sub/a.txt", "sub/../a.txt", "./a.txt"} {
			got, ok := SecureJoin(root, rel)
			if !ok {
				t.Errorf("rel=%q: ok=false になった。真下のパスは許可されるべき", rel)
				continue
			}
			if !filepath.IsAbs(got) && filepath.IsAbs(root) {
				t.Errorf("rel=%q: 絶対パスが返るべき: got=%q", rel, got)
			}
			if got == root {
				t.Errorf("rel=%q: root 自身が返った: got=%q", rel, got)
			}
		}
	})

	t.Run("root の外へ出るパスは拒否する", func(t *testing.T) {
		for _, rel := range []string{"..", "../a.txt", "../../etc/passwd", "sub/../../a.txt"} {
			got, ok := SecureJoin(root, rel)
			if ok {
				t.Errorf("rel=%q: ok=true になった。root の外なので拒否されるべき: got=%q", rel, got)
			}
			if got != "" {
				t.Errorf("rel=%q: 拒否時は空文字を返すべき: got=%q", rel, got)
			}
		}
	})

	// ここが今回の変更点。root 自身を指す rel は ok=false になる。
	t.Run("root 自身を指すパスは拒否する", func(t *testing.T) {
		for _, rel := range []string{"", ".", "./", "sub/..", "." + sep} {
			got, ok := SecureJoin(root, rel)
			if ok {
				t.Errorf("rel=%q: ok=true になった。root 自身は許可されるべきでない: got=%q", rel, got)
			}
			if got != "" {
				t.Errorf("rel=%q: 拒否時は空文字を返すべき: got=%q", rel, got)
			}
		}
	})

	t.Run("root の兄弟で前方一致するディレクトリは拒否する", func(t *testing.T) {
		// root="/tmp/x" のとき "/tmp/xy" を許してしまわないこと。
		// 区切り文字込みで前方一致を見ているので通らない。
		sibling := root + "_sibling"
		got, ok := SecureJoin(root, filepath.Join("..", filepath.Base(sibling)))
		if ok {
			t.Errorf("兄弟ディレクトリが許可された: got=%q", got)
		}
	})
}
