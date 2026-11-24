package main

import (
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 取り込み元は Google Takeout の ZIP そのもの。**展開済みのフォルダは読まない。**
// ZIP の走査は SDK の sdk.OpenSources（fitbit / 位置情報 / chatgpt / claudeai /
// archived_git_commit_log と共通）に任せ、ここに残るのは「どのエントリを読むか」
// 「差分の署名」「設定画面に出す問題の選別」だけ。
// ZIP だけを読む理由と却下案: documents/adr/0310-chat-export-plugins-read-zip.md
// Gemini のエントリの選び方: documents/adr/0320-gemini-activity-from-my-activity.md

// parseSourcePatterns は設定値をパターンのリストにする。
// 空なら従来どおりプラグインフォルダ自身を見る。
func parseSourcePatterns(value any, pluginDir string) []string {
	return sdk.ParseSourcePatterns(value, pluginDir)
}

// isGeminiActivityEntry は ZIP 内のパスが Gemini のアクティビティファイルかどうかを判定する。
//
// **ファイル名で選んではいけない。** Takeout はファイル名もフォルダ名もアカウントの言語で訳す
// （日本語では "Takeout/マイ アクティビティ/Gemini アプリ/マイアクティビティ.json"、
// 英語では "Takeout/My Activity/Gemini Apps/MyActivity.json"）。
// "MyActivity.json" を探すと日本語環境で黙って0件になる。
//
// 訳されないのは製品名の "Gemini" だけなので、**親フォルダ名が Gemini で始まる .json / .html**
// という位置で選び、Gemini のアクティビティかどうかは中身で確かめる（loader.go の isGeminiActivityJSON）。
// マイアクティビティには製品ごとに同じ名前のファイルが並ぶので、ここで絞らないと
// Chrome や検索の履歴まで開くことになる。
func isGeminiActivityEntry(entryName string) bool {
	ext := strings.ToLower(path.Ext(entryName))
	if ext != ".json" && ext != ".html" {
		return false
	}
	return isGeminiDirName(path.Base(path.Dir(entryName)))
}

// isGeminiDirName はフォルダ名が Gemini の製品フォルダかどうかを判定する。
// "Gemini アプリ" / "Gemini Apps" / "Gemini" のどれにも当たる。
func isGeminiDirName(name string) bool {
	return strings.HasPrefix(strings.ToLower(name), "gemini")
}

// openSources は取り込み元の ZIP を開き、Gemini のアクティビティファイルのエントリだけを列挙する。
// 返した *sdk.SourceSet は必ず Close すること。
func openSources(patterns []string) (*sdk.SourceSet, error) {
	return sdk.OpenSources(patterns, isGeminiActivityEntry)
}

// htmlFormatMessage は HTML 形式のアクティビティを見つけたときに設定画面へ出す文言。
// 読まずに黙って0件にはしない（ADR-0310 と同じ考え方）。
const htmlFormatMessage = "HTML 形式なので読みません。Takeout の「マイ アクティビティ」の" +
	"「複数の形式」で「アクティビティの記録」を JSON に変えて書き出し直してください。"

// extractedFolderMessage は ZIP ではないものを指定されたときに設定画面へ出す文言。
const extractedFolderMessage = "ZIP ではないので読みません（展開済みのフォルダは読みません）。" +
	"Google Takeout からダウンロードした ZIP を解凍せずに置き、その ZIP かフォルダを指定してください。"

// relevantProblems は設定画面に出す問題だけを残す。
//
//   - nested_zip は落とす。Gemini のアクティビティには添付ファイルが付き、**その添付が ZIP で
//     Takeout の ZIP の中に並ぶ**。実物の書き出しでは1フォルダに百件規模で入っていて、
//     そのまま出すと設定画面が「ZIPの中のZIPは読みません」で埋まって他の警告が読めなくなる。
//     添付の中身は元から読まない方針なので、問題でもない（名前は本文に並べる）
//   - missing_pattern は設定画面が展開結果から直接出しているので二重になる
//   - extracted_folder はプラグインフォルダ自身なら落とす。source_dirs が空だと既定で
//     プラグインフォルダを見るが、そこには manifest.json / config.json が必ずあるので、
//     SDK は「ZIP が無く JSON だけがある」として毎回これを立てる
//   - **mixed_exports は残す。** Takeout は同じフォルダに世代が並ぶことが普通にあり、
//     どの書き出しを見ているかは利用者が知りたい情報（chatgpt / claudeai はここで落としているが、
//     あちらに世代の概念が無いから）
func relevantProblems(problems []sdk.SourceProblem, pluginDir string) []sdk.SourceProblem {
	ownDir := ""
	if pluginDir != "" {
		if abs, err := filepath.Abs(pluginDir); err == nil {
			ownDir = abs
		}
	}
	kept := make([]sdk.SourceProblem, 0, len(problems))
	for _, problem := range problems {
		switch problem.Kind {
		case sdk.ProblemNestedZip, sdk.ProblemMissingPattern:
			continue
		case sdk.ProblemExtractedFolder:
			if ownDir != "" && strings.EqualFold(filepath.Clean(problem.Path), ownDir) {
				continue
			}
			problem.Message = extractedFolderMessage
		}
		kept = append(kept, problem)
	}
	return kept
}

// sourceSignature は全エントリの Path:CRC32:Size を連結した署名を返す。
// どれか1つでも変わればキャッシュを作り直す。
//
// 更新時刻は使わない。Takeout は書き出し時刻を全エントリに同じ値で入れるので、
// 中身が変わっても動かない（ADR-0303）。
// CRC32 と展開後サイズは中央ディレクトリにあるので、署名を作るのに伸長は要らない。
//
// 署名に入れるのはアクティビティファイルのエントリだけ。添付ファイル（.wav / .zip）は
// 入れない —— accept で弾いているので、そもそもここには来ない。
func sourceSignature(entries []sdk.SourceEntry) string {
	if len(entries) == 0 {
		return ""
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, entry.Path+":"+
			strconv.FormatUint(uint64(entry.CRC32), 16)+":"+
			strconv.FormatInt(entry.Size, 10))
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n")
}

// exportRankOf は世代（ExportID）ごとの順位を返す。0 が最新。
//
// 同じ往復が複数の書き出しに入っていることがあるので、新しい書き出しのほうを採るために使う。
// sdk の Exports() は新しい順に並んでいる。
func exportRankOf(set *sdk.SourceSet) map[string]int {
	ranks := map[string]int{}
	if set == nil {
		return ranks
	}
	for rank, export := range set.Exports() {
		ranks[export.ExportID] = rank
	}
	return ranks
}
