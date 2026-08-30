package dvnf_cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// copyFile は copy.go / move.go 共用の実ファイルコピー本体。
// 書き込み側を明示的に Close してから Chtimes する順序がこの関数の存在理由で
// （Close 前に Chtimes すると、Close 時のフラッシュで mtime が上書きされて
// 反映されないケースがあった。copy.go の明示 Close の行コメント参照）、
// ここでは実ファイルを使って「内容が写ること」と「mtime が写ること」の両方を確認する。

// setCopyLastMod は copyOpt.copyLastMod を書き換え、テスト終了時に元へ戻す。
// copyOpt はパッケージ変数（CLIフラグの入れ物）なので、戻さないと他のテストへ漏れる。
func setCopyLastMod(t *testing.T, v bool) {
	t.Helper()
	prev := copyOpt.copyLastMod
	copyOpt.copyLastMod = v
	t.Cleanup(func() { copyOpt.copyLastMod = prev })
}

// TestCopyFile_CopiesContent は、コピー後にコピー先の内容がコピー元と一致することを確認する。
func TestCopyFile_CopiesContent(t *testing.T) {
	setCopyLastMod(t, false)

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	target := filepath.Join(dir, "target.txt")
	content := []byte("dvnfのコピー対象の中身\n2行目")

	if err := os.WriteFile(src, content, 0o600); err != nil {
		t.Fatalf("コピー元の作成に失敗: %v", err)
	}
	if err := copyFile(src, target); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("コピー先の読み取りに失敗: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("コピー先の内容がコピー元と一致しない: got %q, want %q", got, content)
	}
}

// TestCopyFile_CopiesLastModWhenEnabled は、copyLastMod 有効時にコピー先の mtime が
// コピー元と一致することを確認する。dvnf は世代ディレクトリ間の差分判定（--fast）を
// mtime の一致で行うので、ここが写らないと毎回全ファイルをコピーし直すことになる。
func TestCopyFile_CopiesLastModWhenEnabled(t *testing.T) {
	setCopyLastMod(t, true)

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	target := filepath.Join(dir, "target.txt")

	if err := os.WriteFile(src, []byte("mtime確認用"), 0o600); err != nil {
		t.Fatalf("コピー元の作成に失敗: %v", err)
	}
	// 「今」から十分離れた決定的な時刻にする。コピーした時刻のままでも偶然一致しないように。
	srcModTime := time.Date(2020, 1, 2, 3, 4, 5, 0, time.Local)
	if err := os.Chtimes(src, srcModTime, srcModTime); err != nil {
		t.Fatalf("コピー元のmtime設定に失敗: %v", err)
	}

	if err := copyFile(src, target); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	targetInfo, err := os.Stat(target)
	if err != nil {
		t.Fatalf("コピー先のstatに失敗: %v", err)
	}
	// ファイルシステムの時刻解像度差（FATは2秒刻み等）に左右されないよう、1秒精度へ丸めて比較する。
	got := targetInfo.ModTime().Truncate(time.Second)
	want := srcModTime.Truncate(time.Second)
	if !got.Equal(want) {
		t.Errorf("コピー先のmtimeがコピー元と一致しない: got %v, want %v", got, want)
	}
}
