package reps

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// localRepCacheNeedsCopy は元DBファイルをローカルキャッシュへコピーし直す必要があるかを返します。
//
// 判定は mtime + サイズ です。
// どちらかを stat できないとき（初回・元ファイル消失）は、安全側に倒して「要コピー」にします。
//
// この判定は *_local_cached.go の UpdateCache で
// 「閉じる → 消す → コピーし直す → 開き直す」を丸ごと飛ばすために使います。
// os.Remove のあとに呼んではいけません。
// 消してから stat すると cacheStatErr が必ず立つので常に「要コピー」になり、
// 変更のないrepまで LastUpdateCacheChanged() が true を返し、
// 上位のキャッシュrepが毎回フルリビルドします。
// 実データ（rep約940・約83万行・外付けUSB上のDB 818本 1.3GB）では
// これで update_cache 1回が2分を超えていました。
func localRepCacheNeedsCopy(originalDBFileName string, localCacheDBFileName string) bool {
	cacheStat, cacheStatErr := os.Stat(localCacheDBFileName)
	originalStat, originalStatErr := os.Stat(originalDBFileName)
	if originalStatErr != nil || cacheStatErr != nil {
		return true
	}
	return !originalStat.ModTime().Equal(cacheStat.ModTime()) || originalStat.Size() != cacheStat.Size()
}

// copyLocalRepCacheDB は元DBファイルをローカルキャッシュへコピーします。
//
// コピー後に mtime を元ファイルへ合わせます。
// localRepCacheNeedsCopy はこの mtime を基準に比較するので、
// 合わせ忘れると次回以降も毎回コピーし直すことになります。
func copyLocalRepCacheDB(originalDBFileName string, localCacheDBFileName string) error {
	originalStat, err := os.Stat(originalDBFileName)
	if err != nil {
		return fmt.Errorf("error at stat file %s: %w", originalDBFileName, err)
	}

	originalDBFile, err := os.Open(originalDBFileName)
	if err != nil {
		return fmt.Errorf("error at open file %s: %w", originalDBFileName, err)
	}
	defer func() {
		err := originalDBFile.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	cacheDBFile, err := os.Create(localCacheDBFileName)
	if err != nil {
		return fmt.Errorf("error at open file %s: %w", localCacheDBFileName, err)
	}

	_, copyErr := io.Copy(cacheDBFile, originalDBFile)
	// Chtimesより先に閉じる。開いたままだと、このあとのCloseで書き込み時刻が上書きされ、
	// 次回の localRepCacheNeedsCopy が必ず「要コピー」になる。
	closeErr := cacheDBFile.Close()
	if copyErr != nil {
		return fmt.Errorf("error at copy local cache db %s to %s: %w", originalDBFileName, localCacheDBFileName, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("error at close local cache db %s: %w", localCacheDBFileName, closeErr)
	}

	os.Chtimes(localCacheDBFileName, originalStat.ModTime(), originalStat.ModTime())
	return nil
}
