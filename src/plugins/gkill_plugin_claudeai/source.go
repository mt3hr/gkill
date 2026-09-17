package main

import (
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// 取り込み元は Claude.ai からダウンロードした ZIP そのもの。**展開済みの JSON は読まない。**
// ZIP の走査は SDK の sdk.OpenSources（fitbit / 位置情報 / archived_git_commit_log と共通）に
// 任せ、ここに残るのは「どのエントリを読むか」「差分の署名」「設定画面に出す問題の選別」だけ。
// ZIP だけを読む理由と却下案: documents/adr/0310-chat-export-plugins-read-zip.md

// parseSourcePatterns は設定値をパターンのリストにする。
// 空なら従来どおりプラグインフォルダ自身を見る。
func parseSourcePatterns(value any, pluginDir string) []string {
	return sdk.ParseSourcePatterns(value, pluginDir)
}

// isConversationEntry は ZIP 内のパスが会話ファイルかどうかを判定する。
// ベース名が conversations.json（実物。conversations-000.zip の中にこの1本）か
// conversations-NNN.json（将来 ZIP の中でも分割されたときのため）。
func isConversationEntry(entryName string) bool {
	name := pathBase(entryName)
	return isNumberedConversationFile(name) || isSingleConversationFile(name)
}

// openSources は取り込み元の ZIP を開き、会話ファイルのエントリだけを列挙する。
// 返した *sdk.SourceSet は必ず Close すること。
func openSources(patterns []string) (*sdk.SourceSet, error) {
	return sdk.OpenSources(patterns, isConversationEntry)
}

// pathBase は ZIP 内のパス（区切りは常に "/"）からベース名を取り出す。
func pathBase(entryName string) string {
	if index := strings.LastIndexByte(entryName, '/'); index >= 0 {
		return entryName[index+1:]
	}
	return entryName
}

// extractedFolderMessage は ZIP ではないものを指定されたときに設定画面へ出す文言。
// SDK の文言は Takeout の話なので言い換える。
const extractedFolderMessage = "ZIP ではないので読みません（展開済みの JSON は読みません）。" +
	"Claude.ai からエクスポートした ZIP（conversations-000.zip など）を解凍せずに置き、その ZIP かフォルダを指定してください。"

// relevantProblems は設定画面に出す問題だけを残す。
//
//   - nested_zip / mixed_exports は Takeout 固有の話（Claude.ai の ZIP に世代の概念は無い）
//   - missing_pattern は設定画面が展開結果から直接出しているので二重になる
//   - extracted_folder はプラグインフォルダ自身なら落とす。source_dirs が空だと既定で
//     プラグインフォルダを見るが、そこには manifest.json / config.json が必ずあるので、
//     SDK は「ZIP が無く JSON だけがある」として毎回これを立てる
//   - それ以外の extracted_folder（展開済み JSON のフォルダ・conversations*.json の直接指定）は
//     文言を Claude.ai 向けに言い換えて残す。旧配置のまま黙って0件にしないための唯一の出口
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
		case sdk.ProblemNestedZip, sdk.ProblemMixedExports, sdk.ProblemMissingPattern:
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
// 更新時刻は使わない。Claude.ai の ZIP はエントリの更新時刻が 1980-01-01 固定で、
// 中身が変わっても動かない（ADR-0303）。
// CRC32 と展開後サイズは中央ディレクトリにあるので、署名を作るのに伸長は要らない。
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
