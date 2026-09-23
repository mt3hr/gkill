package req_res

import "time"

// SkillInfo はスキル一覧の1行（/api/get_skill_list）。
type SkillInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UpdatedTime time.Time `json:"updated_time"`
	FileCount   int       `json:"file_count"`
	// InvalidReason は SKILL.md が無い・frontmatter が壊れている等の理由。正常なら空。
	InvalidReason string `json:"invalid_reason"`
}

// SkillFileInfo はスキル内の1ファイルの情報。Revision は中身の SHA-256 の hex 先頭16桁で、
// /api/write_skill_file・/api/delete_skill の楽観ロックに渡す。
type SkillFileInfo struct {
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	IsText      bool      `json:"is_text"`
	Revision    string    `json:"revision"`
	UpdatedTime time.Time `json:"updated_time"`
}

// SkillDetail は1つのスキルの中身（SKILL.md の全文とファイル一覧）。
type SkillDetail struct {
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	InvalidReason string    `json:"invalid_reason"`
	UpdatedTime   time.Time `json:"updated_time"`
	// Content は SKILL.md の全文（frontmatter を含む）。SKILL.md が無ければ空。
	Content string `json:"content"`
	// Revision は SKILL.md の revision。
	Revision string           `json:"revision"`
	Files    []*SkillFileInfo `json:"files"`
}

// SkillFileContent はスキル内の1ファイルの中身。テキストは Content、バイナリは ContentBase64 に入る。
// max_bytes を超えたときはどちらも空で ContentOmitted が true。
type SkillFileContent struct {
	SkillFileInfo
	Content        string `json:"content"`
	ContentBase64  string `json:"content_base64"`
	ContentOmitted bool   `json:"content_omitted"`
}

// SkillReplacePlan は zip で置き換えたときに何が起きるか（/api/upload_skill）。
type SkillReplacePlan struct {
	Name    string   `json:"name"`
	IsNew   bool     `json:"is_new"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Changed []string `json:"changed"`
	Ignored []string `json:"ignored"`
}
