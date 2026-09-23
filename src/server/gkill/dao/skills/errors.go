package skills

import (
	"errors"
	"strings"
)

// 保存層が返す番兵エラー。ハンドラは errors.Is でこれらを見分けてエラーコードへ写す。
//
// 「見つからない」系は fs.ErrNotExist を包まない。包むと応答の reason が
// storage_unavailable（ストレージが使えない）になり、404 に誤った理由が付くため。
var (
	// ErrSkillNotFound はスキルのディレクトリが無い。
	ErrSkillNotFound = errors.New("skill not found")
	// ErrFileNotFound はスキルの中に指定のファイルが無い。
	ErrFileNotFound = errors.New("skill file not found")
	// ErrInvalidUserID は利用者IDをディレクトリ名として使えない。
	ErrInvalidUserID = errors.New("invalid user id for skills directory")
	// ErrInvalidName はスキル名が規則（英小文字・数字・ハイフン）に合わない。
	ErrInvalidName = errors.New("invalid skill name")
	// ErrInvalidPath はスキル内のパスが規則に合わない。
	ErrInvalidPath = errors.New("invalid skill file path")
	// ErrInvalidFrontmatter は SKILL.md の frontmatter が読めない・name や description が不正。
	ErrInvalidFrontmatter = errors.New("invalid SKILL.md frontmatter")
	// ErrInvalidZip はアップロードされた zip がスキルとして取り込めない。
	ErrInvalidZip = errors.New("invalid skill zip")
	// ErrAlreadyExists は新規作成（revision 省略）のつもりで、既にファイルがある。
	ErrAlreadyExists = errors.New("skill file already exists")
	// ErrRevisionConflict は渡された revision が今の中身と食い違う（他で書き換えられた）。
	ErrRevisionConflict = errors.New("skill file revision conflict")
	// ErrManifestDelete は SKILL.md だけを消そうとした。消すならスキルごと消す。
	ErrManifestDelete = errors.New("SKILL.md cannot be deleted alone")
)

// DetailError は番兵エラーに「どこが悪いか」を添える。
// errors.Is は Kind に委ねるので、ハンドラは番兵で分岐しつつ Detail を利用者へ見せられる。
type DetailError struct {
	Kind   error
	Detail string
	// Paths は問題のあったパス（zip の中身の検証で使う）。
	Paths []string
}

func (e *DetailError) Error() string {
	msg := e.Kind.Error()
	if e.Detail != "" {
		msg += ": " + e.Detail
	}
	if len(e.Paths) != 0 {
		msg += " (" + strings.Join(e.Paths, ", ") + ")"
	}
	return msg
}

func (e *DetailError) Unwrap() error { return e.Kind }

func detailError(kind error, detail string, paths ...string) error {
	return &DetailError{Kind: kind, Detail: detail, Paths: paths}
}

// DescribeError は利用者に見せる補足（Detail とパス）を返す。番兵そのものなら空文字。
func DescribeError(err error) string {
	var detailErr *DetailError
	if !errors.As(err, &detailErr) {
		return ""
	}
	msg := detailErr.Detail
	if len(detailErr.Paths) != 0 {
		if msg != "" {
			msg += ": "
		}
		msg += strings.Join(detailErr.Paths, ", ")
	}
	return msg
}
