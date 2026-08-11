package main

import (
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// defaultSourcePattern は設定が空のときに使う既定のデータソース。
const defaultSourcePattern = "~/Kyou/GoogleTakeout_*"

// parseSourcePatterns は設定値をパターンのリストにする。
// 展開の実体はSDKにある（fitbitプラグインと共通）。
func parseSourcePatterns(value any) []string {
	return sdk.ParseSourcePatterns(value, defaultSourcePattern)
}

// openSources は取り込み元のZIPを開き、位置情報でありうるエントリだけを列挙する。
//
// ZIPしか読まない。展開済みのフォルダは対象外。
//
// 形式の判定はここではしない（中身を読む必要があるため）。
// ここでやるのは拡張子による足切りだけで、実際の判定は detectFormat が行う。
// Takeout のルートを指定すれば、その下の タイムライン/ や
// Google Health/Physical Activity_GoogleData/ を再帰走査で自然に見つけられる。
//
// 返した *sdk.SourceSet は必ず Close すること。
func openSources(patterns []string) (*sdk.SourceSet, error) {
	return sdk.OpenSources(patterns, func(entryName string) bool {
		lowerName := strings.ToLower(pathBase(entryName))
		if strings.HasSuffix(lowerName, ".json") {
			return true
		}
		// CSVはワークアウトのトラックだけ。Takeoutには無関係なCSVが1,900本ある
		return strings.HasSuffix(lowerName, ".csv") && strings.HasPrefix(lowerName, "gps_location")
	})
}

// pathBase はZIP内のパス（区切りは常に "/"）からベース名を取り出す。
func pathBase(entryName string) string {
	if index := strings.LastIndexByte(entryName, '/'); index >= 0 {
		return entryName[index+1:]
	}
	return entryName
}

// scannedFileOf は取り込み元のエントリを差分判定用の形に写す。
// 形式は呼び出し側が入れる。
func scannedFileOf(entry sdk.SourceEntry, formatID string) scannedFile {
	return scannedFile{
		Path:      entry.Path,
		MtimeUnix: entry.MtimeUnix,
		Size:      entry.Size,
		CRC32:     entry.CRC32,
		Format:    formatID,
	}
}
