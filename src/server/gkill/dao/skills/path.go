package skills

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ManifestFileName はスキルの本体（frontmatter + 手順本文）のファイル名。
const ManifestFileName = "SKILL.md"

// skillNamePattern はスキル名の規則。Agent Skills と同じく英小文字・数字・ハイフンで、
// 先頭と末尾は英数字。先頭が英数字なので、予約名（_global や作業用の _tmp-*）や
// ドットで始まるディレクトリ（利用者が置く .git など）とは構造的に衝突しない。
var skillNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

// pathSegmentPattern はスキル内パスの1要素の規則。先頭が英数字なので、
// ドットで始まるファイル・".."・空要素は構造的に作れない。ASCII に限るのは、
// Windows で作った zip の日本語名（Shift_JIS）の文字化けや端末間の正規化の差を避けるため。
var pathSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// windowsReservedNames は Windows でファイル名に使えない名前（拡張子の有無を問わない）。
var windowsReservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true, "COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true, "LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// ValidateSkillName はスキル名が規則に合うかを返す。
func ValidateSkillName(name string) error {
	if !skillNamePattern.MatchString(name) {
		return detailError(ErrInvalidName, fmt.Sprintf("%q (use lowercase letters, digits and hyphens; 1-64 chars; start and end with a letter or digit)", name))
	}
	return nil
}

// NormalizeFilePath はスキル内のパス（区切りは "/"）を検査して返す。
// フォルダは実体を持たずパスの一部として扱うので、ここで通るのはファイルのパスだけ。
func NormalizeFilePath(path string) (string, error) {
	if path == "" {
		return "", detailError(ErrInvalidPath, "empty path")
	}
	if strings.Contains(path, `\`) {
		return "", detailError(ErrInvalidPath, `use "/" as the separator`, path)
	}
	segments := strings.Split(path, "/")
	for _, segment := range segments {
		if !pathSegmentPattern.MatchString(segment) {
			return "", detailError(ErrInvalidPath, "each path segment must start with a letter or digit and contain only letters, digits, '.', '_' and '-'", path)
		}
		if strings.HasSuffix(segment, ".") {
			return "", detailError(ErrInvalidPath, "a path segment must not end with '.'", path)
		}
		base := strings.ToUpper(segment)
		if dot := strings.IndexByte(base, '.'); dot >= 0 {
			base = base[:dot]
		}
		if windowsReservedNames[base] {
			return "", detailError(ErrInvalidPath, "reserved file name on Windows", path)
		}
	}
	// ルートの SKILL.md は綴りを固定する（skill.md 等を許すと Windows で同じファイルになる）
	if len(segments) == 1 && strings.EqualFold(path, ManifestFileName) && path != ManifestFileName {
		return "", detailError(ErrInvalidPath, "the manifest must be spelled "+ManifestFileName, path)
	}
	return path, nil
}

// isSingleSafePathElement は値を単一のパス要素として使ってよいか検証する。
// 区切り文字・親ディレクトリ参照・空文字を含むものを拒否する（dao/reps/local_rep_cache_path.go と同じ規則）。
// 加えて ":" と NUL も拒否する（Windows ではドライブ指定や代替データストリームの意味になる。
// 利用者IDの形式はアカウント作成時にしか検査していないので、既存アカウントには何でも入りうる）。
func isSingleSafePathElement(element string) bool {
	if element == "" || element == "." || element == ".." {
		return false
	}
	if strings.ContainsAny(element, "/\\:\x00") {
		return false
	}
	return filepath.Clean(element) == element
}

// joinWithin は root の下へ rel（"/" 区切り）を繋げ、root の外へ出ないことを確かめて返す。
func joinWithin(root string, rel string) (string, error) {
	native := filepath.FromSlash(rel)
	// CodeQL の path-injection / zipslip は filepath.IsLocal の真分岐をバリアとして認識する（Rel の結果の検査は認識しない）。
	// NormalizeFilePath と zip の検証を通ったパスは常にここを通るので、実行時の防御は下の Rel 検査が担う。
	// ".." を ReplaceAll で除く形（dao/reps/local_rep_cache_path.go）はここでは使えない。"a..b.txt" は正しいファイル名で、黙って別名に書かれる。
	if !filepath.IsLocal(native) {
		return "", detailError(ErrInvalidPath, "path escapes the skill directory", rel)
	}
	joined := filepath.Join(root, native)
	relFromRoot, err := filepath.Rel(root, joined)
	if err != nil || relFromRoot == "." || relFromRoot == ".." || strings.HasPrefix(relFromRoot, ".."+string(filepath.Separator)) || filepath.IsAbs(relFromRoot) {
		return "", detailError(ErrInvalidPath, "path escapes the skill directory", rel)
	}
	return joined, nil
}

// findCaseConflict は paths の中に、path と大文字小文字だけが違うものがあれば返す。
// Windows では同じファイルになるので、同じスキルの中では重複として扱う。
func findCaseConflict(paths []string, path string) string {
	for _, existing := range paths {
		if existing != path && strings.EqualFold(existing, path) {
			return existing
		}
	}
	return ""
}

// findFileDirConflict は、path とファイル/フォルダの関係でぶつかるものがあれば返す。
// 例: "a" がファイルなのに "a/b" を置こうとした、またはその逆。
func findFileDirConflict(paths []string, path string) string {
	lowerPath := strings.ToLower(path)
	for _, existing := range paths {
		lowerExisting := strings.ToLower(existing)
		if strings.HasPrefix(lowerPath, lowerExisting+"/") || strings.HasPrefix(lowerExisting, lowerPath+"/") {
			return existing
		}
	}
	return ""
}
