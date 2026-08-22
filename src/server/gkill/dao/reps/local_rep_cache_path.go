package reps

// rep名の照合では守れない理由（rep名は利用者間で一意でない）:
// documents/adr/0044-per-user-derived-cache-dir.md

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// isSingleSafePathElement は値を単一のパス要素として使ってよいか検証する。
// 区切り文字・親ディレクトリ参照・空文字を含むものを拒否する。
func isSingleSafePathElement(element string) bool {
	if element == "" || element == "." || element == ".." {
		return false
	}
	if strings.ContainsAny(element, `/\`) {
		return false
	}
	return filepath.Clean(element) == element
}

// noUserCacheDirName は利用者が特定できないときに使うディレクトリ名。
// 実在する利用者IDと衝突しないよう、アカウント作成時の検証（先頭は英数字のみ）が
// 通らない形にしてある。
const noUserCacheDirName = "_nouser"

// derivedCacheDirForUser は派生キャッシュ（thumb / video / zip）のディレクトリを
// caches/{cacheName}/{userID}/{repName} の形で返す。
//
// rep名は filepath.Base(contentDir) で決まるが、これは利用者間で一意ではない。
// 実際に「別のディレクトリを指す同名rep」が複数の利用者に存在するため、
// rep名だけで分けるとキャッシュが混ざる。利用者IDを1階層挟んで分離する。
//
// HTTP配信を伴わない経路（idfサブコマンド等）は userID を持たないので、
// 実在する利用者と衝突しない固定名へ寄せる。
func derivedCacheDirForUser(cacheName string, userID string, contentDir string) string {
	dirName := userID
	if !isSingleSafePathElement(dirName) {
		dirName = noUserCacheDirName
	}
	// ".." 除去。直前の検証で弾いているため実行時には常にno-opだが、
	// CodeQL の path-injection はこの形しか値サニタイザとして認識しない。
	dirName = strings.ReplaceAll(dirName, "..", "")

	repName := filepath.Base(filepath.Clean(os.ExpandEnv(contentDir)))
	repName = strings.ReplaceAll(repName, "..", "")

	return os.ExpandEnv(filepath.Join(gkill_options.CacheDir, cacheName, dirName, repName))
}

// localRepCacheDBFileName は元DBファイルに対応するローカルキャッシュDBのパスを返す。
//
// 複数のユーザが同じファイルをリポジトリとして共有していることがある
// （たとえば user1 と user2 がどちらも datas/user/Kmemo.db を参照している）。
// パスを元DBファイル名だけから決めると、そうした複数のRepositoryが
// まったく同じキャッシュファイルを掴むことになる。
// UpdateCache は「自分のハンドルを閉じる → ファイルを削除 → 再コピー」で動くため、
// 片方が削除しようとした時点でもう片方がまだ開いていると
// Windowsでは共有違反になりUpdateCacheが失敗する。
// これを避けるためユーザ単位でディレクトリを分ける。
//
// userID はアカウント作成時にパス要素としての検証をしていないため、
// originalDBFileName はリポジトリ設定（HTTP経由で更新できる）由来のため、
// どちらもキャッシュルート外へ抜け出す値になりうる。
// ルート配下に閉じ込められない場合はエラーを返す。
func localRepCacheDBFileName(userID string, originalDBFileName string) (string, error) {
	if !isSingleSafePathElement(userID) {
		return "", fmt.Errorf("invalid user id for local rep cache path: %q", userID)
	}
	// ".." 除去。直前の検証で弾いているため実行時には常にno-opだが、
	// CodeQL の path-injection はこの形しか値サニタイザとして認識しない。
	// originalDBFileName 側の同じ処理は実際にアラートを消せている。
	safeUserID := strings.ReplaceAll(userID, "..", "")
	rootDir := filepath.Join(os.ExpandEnv(gkill_options.CacheDir), "local_rep_cache", safeUserID)

	// ":" 除去はWindowsのドライブレター対策（元からの動作）。
	rel := strings.ReplaceAll(originalDBFileName, ":", "")
	// ".." 除去。SecureJoin でも弾いているため実行時には保険だが、
	// CodeQL の path-injection はこの形のみをサニタイザとして認識する。
	rel = strings.ReplaceAll(rel, "..", "")

	localCacheDBFileName, ok := SecureJoin(rootDir, rel)
	if !ok {
		return "", fmt.Errorf("local rep cache path escapes cache root: user id = %q original db file name = %q", userID, originalDBFileName)
	}
	return localCacheDBFileName, nil
}
