package mcp

// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md
//
// スキル（利用者が AI 向けに書いた手順書。$GKILL_HOME/skills/<user_id>/<name>/ の SKILL.md と付属ファイル）の
// ツール。ファイルを触るのは gkill_server だけで、ここは /api/get_skill_list・/api/get_skill・
// /api/write_skill_file を呼ぶ HTTP クライアント（ADR-0634）。定義は read_tools.go / write_tools.go
// （verify_docs がツール数をそこから数える）、非公開の削除ツールは skill_delete_tool.go。

import (
	"encoding/json"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// skillManifestPath は path を省いたときに書く SKILL.md。
const skillManifestPath = "SKILL.md"

// skillImageMimeTypes はバイナリの拡張子から image ブロックの mimeType を決める表。
// mime.TypeByExtension は Windows ではレジストリを読むので端末ごとに結果が変わる（golden が揺れる）。
var skillImageMimeTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".gif":  "image/gif",
	".webp": "image/webp",
}

func skillMimeType(path string) string {
	lower := strings.ToLower(path)
	for ext, mimeType := range skillImageMimeTypes {
		if strings.HasSuffix(lower, ext) {
			return mimeType
		}
	}
	if strings.HasSuffix(lower, ".pdf") {
		return "application/pdf"
	}
	return "application/octet-stream"
}

// optionalTrimmedString は source[key] があれば空でない文字列として取り出す。
func optionalTrimmedString(source *jsonobj.Object, key string) (string, bool, error) {
	if !source.Defined(key) {
		return "", false, nil
	}
	s, err := AssertTrimmedString(source.Value(key), key)
	if err != nil {
		return "", false, err
	}
	return s, true, nil
}

// rawString は source[key] を文字列として取り出す（trim せず、空文字も許す。ファイルの中身用）。
func rawString(source *jsonobj.Object, key string) (string, error) {
	s, ok := source.Value(key).(string)
	if !ok {
		return "", InvalidArgument(key, "must be a string", source.Value(key))
	}
	return s, nil
}

// normalizeSkillArgs はスキルのツールの引数を検査する。required は必須のキー、trimmed は前後空白を落とすキー。
// 名前・パスの規則そのものは gkill_server が検査する（1実装。違反は 400 で理由とパスが返る）。
func normalizeSkillArgs(args any, allowed []string, required []string, raw []string) (*jsonobj.Object, error) {
	source, err := argsObject(args)
	if err != nil {
		return nil, err
	}
	if err := AssertKnownKeys(source, NewStringSet(allowed...), "arguments", noHiddenKeys); err != nil {
		return nil, err
	}
	normalized := jsonobj.New()
	for _, key := range allowed {
		isRequired := containsStringItem(required, key)
		if !source.Defined(key) {
			if isRequired {
				return nil, InvalidArgument(key, "is required", source.Value(key))
			}
			continue
		}
		if containsStringItem(raw, key) {
			s, err := rawString(source, key)
			if err != nil {
				return nil, err
			}
			normalized.Set(key, s)
			continue
		}
		s, _, err := optionalTrimmedString(source, key)
		if err != nil {
			return nil, err
		}
		normalized.Set(key, s)
	}
	return normalized, nil
}

// NormalizeGetSkillArgs は gkill_get_skill の引数。
func NormalizeGetSkillArgs(args any) (*jsonobj.Object, error) {
	return normalizeSkillArgs(args, []string{"name", "path", "locale_name"}, []string{"name"}, nil)
}

// NormalizeAddSkillArgs は gkill_add_skill の引数。body は trim しない（Markdown の先頭の空白も中身）。
func NormalizeAddSkillArgs(args any) (*jsonobj.Object, error) {
	return normalizeSkillArgs(args, []string{"name", "description", "body", "locale_name"}, []string{"name", "description", "body"}, []string{"body"})
}

// NormalizeUpdateSkillArgs は gkill_update_skill の引数。content は trim しない。
func NormalizeUpdateSkillArgs(args any) (*jsonobj.Object, error) {
	return normalizeSkillArgs(args, []string{"name", "path", "content", "revision", "locale_name"}, []string{"name", "content"}, []string{"content"})
}

// localeBody は gkill へ送る本文に locale_name を（あれば）載せる。
func localeBody(normalized *jsonobj.Object, kv ...any) *jsonobj.Object {
	body := jsonobj.Obj(kv...)
	if normalized.Defined("locale_name") {
		body.Set("locale_name", normalized.Value("locale_name"))
	}
	return body
}

// fetchSkillList は gkill のスキル一覧（[{name, description, updated_time, file_count, invalid_reason}]）を返す。
func fetchSkillList(ctx *CallContext, normalized *jsonobj.Object) ([]any, error) {
	response, err := ctx.callApi("/api/get_skill_list", localeOnlyBody(normalized))
	if err != nil {
		return nil, err
	}
	return arrayOrEmpty(response.Value("skills")), nil
}

func handleGetSkillList(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeLocaleOnlyArgs(args)
	if err != nil {
		return nil, err
	}
	skills, err := fetchSkillList(ctx, normalized)
	if err != nil {
		return nil, err
	}
	return jsonobj.Obj("skills", skills), nil
}

func handleGetSkill(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeGetSkillArgs(args)
	if err != nil {
		return nil, err
	}
	name := jsString(normalized.Value("name"))
	body := localeBody(normalized, "name", name)
	if normalized.Defined("path") {
		body.Set("path", normalized.Value("path"))
		// 保存量に上限は無いが、AI へ返す量は IDF と同じ max_file_bytes で抑える（ADR-0634）
		body.Set("max_bytes", MaxIDFFileBytes)
	}
	response, err := ctx.callApi("/api/get_skill", body)
	if err != nil {
		return nil, err
	}
	if !normalized.Defined("path") {
		skill, ok := response.Object("skill")
		if !ok || skill == nil {
			return nil, NewGkillApiError("gkill returned no skill for "+name+".", nil)
		}
		return skill.Clone(), nil
	}

	file, ok := response.Object("file")
	if !ok || file == nil {
		return nil, NewGkillApiError("gkill returned no file for "+name+"/"+jsString(normalized.Value("path"))+".", nil)
	}
	path := jsString(file.Value("path"))
	payload := jsonobj.Obj(
		"name", name,
		"path", path,
		"size", file.Value("size"),
		"is_text", jsTruthy(file.Value("is_text")),
		"revision", file.Value("revision"),
		"updated_time", file.Value("updated_time"),
	)
	switch {
	case jsTruthy(file.Value("content_omitted")):
		payload.Set("content_omitted", true)
		payload.Set("warnings", jsonobj.Arr(
			"The file is "+jsString(file.Value("size"))+" bytes, larger than this server's max_file_bytes ("+jsString(MaxIDFFileBytes)+
				"), so its content was not returned. Ask the user to download the skill as a zip from gkill's settings screen if you need it.",
		))
	case jsTruthy(file.Value("is_text")):
		payload.Set("content", nullish(file.Value("content"), ""))
	default:
		mimeType := skillMimeType(path)
		// file_content_base64 / mime_type / is_image は gkill_get_idf_file と同じキー。
		// BuildToolResult がテキスト表現から base64 を外し、画像は image ブロックで届ける。
		payload.Set("mime_type", mimeType)
		payload.Set("is_image", strings.HasPrefix(mimeType, "image/"))
		payload.Set("file_content_base64", nullish(file.Value("content_base64"), ""))
	}
	return payload, nil
}

// buildSkillManifest は gkill_add_skill の SKILL.md を組み立てる。
// frontmatter の値は JSON の文字列で書く（YAML の二重引用符スカラーとしてそのまま読める。
// 説明文に ":" や "#" が入っても壊れない）。gkill_server が name / description を検査する。
func buildSkillManifest(name string, description string, body string) (string, error) {
	quotedName, err := json.Marshal(name)
	if err != nil {
		return "", err
	}
	quotedDescription, err := json.Marshal(description)
	if err != nil {
		return "", err
	}
	manifest := "---\nname: " + string(quotedName) + "\ndescription: " + string(quotedDescription) + "\n---\n" + body
	if body != "" && !strings.HasSuffix(body, "\n") {
		manifest += "\n"
	}
	return manifest, nil
}

func handleAddSkill(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeAddSkillArgs(args)
	if err != nil {
		return nil, err
	}
	name := jsString(normalized.Value("name"))
	manifest, err := buildSkillManifest(name, jsString(normalized.Value("description")), jsString(normalized.Value("body")))
	if err != nil {
		return nil, err
	}
	// revision を渡さない = 新規作成だけ。既にあれば gkill が 409（ERR000436）で断る
	response, err := ctx.callApi("/api/write_skill_file", localeBody(normalized,
		"name", name,
		"path", skillManifestPath,
		"content", manifest,
	))
	if err != nil {
		return nil, err
	}
	return jsonobj.Obj("name", name, "path", skillManifestPath, "revision", response.Value("revision")), nil
}

func handleUpdateSkill(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeUpdateSkillArgs(args)
	if err != nil {
		return nil, err
	}
	name := jsString(normalized.Value("name"))
	path := skillManifestPath
	if normalized.Defined("path") {
		path = jsString(normalized.Value("path"))
	}
	body := localeBody(normalized, "name", name, "path", path, "content", normalized.Value("content"))
	if normalized.Defined("revision") {
		body.Set("revision", normalized.Value("revision"))
	}
	response, err := ctx.callApi("/api/write_skill_file", body)
	if err != nil {
		return nil, err
	}
	return jsonobj.Obj("name", name, "path", path, "revision", response.Value("revision")), nil
}

// summarizeSkillPayload はスキルのツールの1行要約。対象外は false。
func summarizeSkillPayload(name string, payload *jsonobj.Object) (string, bool) {
	switch name {
	case "gkill_get_skill_list":
		skills := arrayOrEmpty(payload.Value("skills"))
		names := []string{}
		for _, item := range skills {
			if skill, ok := item.(*jsonobj.Object); ok && skill != nil {
				names = append(names, jsString(skill.Value("name")))
			}
		}
		if len(names) == 0 {
			return "No skills are stored for this account.", true
		}
		return itoa(len(names)) + " skill(s): " + strings.Join(names, ", ") + ".", true
	case "gkill_get_skill":
		skillName := jsString(payload.Value("name"))
		if !payload.Defined("path") {
			return "Skill " + skillName + ": SKILL.md + " + itoa(lengthOfArray(payload.Value("files"))) + " file(s) listed.", true
		}
		file := skillName + "/" + jsString(payload.Value("path"))
		if jsTruthy(payload.Value("content_omitted")) {
			return "Skill file " + file + ": " + jsString(payload.Value("size")) + " bytes — content NOT returned (larger than max_file_bytes).", true
		}
		kind := "binary"
		if jsTruthy(payload.Value("is_text")) {
			kind = "text"
		}
		return "Skill file " + file + " (" + jsString(payload.Value("size")) + " bytes, " + kind + ").", true
	case "gkill_add_skill":
		return "Created skill: " + jsString(payload.Value("name")) + " (SKILL.md revision " + jsString(payload.Value("revision")) + ").", true
	case "gkill_update_skill":
		return "Wrote skill file: " + jsString(payload.Value("name")) + "/" + jsString(payload.Value("path")) + " (revision " + jsString(payload.Value("revision")) + ").", true
	}
	return "", false
}
