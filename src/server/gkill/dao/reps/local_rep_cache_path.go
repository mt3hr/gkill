package reps

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// localRepCacheDBFileName は元DBファイルに対応するローカルキャッシュDBのパスを返す。
//
// 複数のユーザが同じファイルをリポジトリとして共有していることがある
// （たとえば testuser と testuser_all がどちらも datas/testuser/Kmemo.db を参照している）。
// パスを元DBファイル名だけから決めると、そうした複数のRepositoryが
// まったく同じキャッシュファイルを掴むことになる。
// UpdateCache は「自分のハンドルを閉じる → ファイルを削除 → 再コピー」で動くため、
// 片方が削除しようとした時点でもう片方がまだ開いていると
// Windowsでは共有違反になりUpdateCacheが失敗する。
// これを避けるためユーザ単位でディレクトリを分ける。
func localRepCacheDBFileName(userID string, originalDBFileName string) string {
	return filepath.Join(
		os.ExpandEnv(gkill_options.CacheDir),
		"local_rep_cache",
		userID,
		strings.ReplaceAll(originalDBFileName, ":", ""),
	)
}
