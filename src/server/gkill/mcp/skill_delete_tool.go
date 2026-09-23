package mcp

// gkill_delete_skill（スキル内のファイルを1つ消す）。**実装だけして公開していない。**
//
// 公開しない理由: スキルは履歴を持たない（記録アプリの本質ではないので gkill に責務を持たせない。ADR-0634）。
// そのため AI が消したファイルは取り消せない。AI の誤操作や、読み込んだ記録・Web ページに紛れ込んだ
// 指示に従った削除を、利用者が後から戻す手段が無い。削除は利用者が gkill の設定画面から行う
// （画面の削除はスキル丸ごと。ファイル単位は zip の置き換えで消える）。
//
// 公開するなら、履歴か「消す前に利用者が確認する」仕組みとセットにし、ADR-0634 を見直すこと。
// 公開の手順: write_tools.go の WriteTools でコメントアウトしてある `deleteSkillTool,` を戻し、
// write_handlers.go の dispatchWriteToolCall でコメントアウトしてある case を戻す。
// 一覧に載せないまま case だけ戻しても呼べない（Server.HandleToolCall が IsWriteToolName で弾く）。
//
// 定義を read_tools.go / write_tools.go に置かないのは、verify_docs がその2ファイルの
// `tool("gkill_…"` を（コメントの中でも）数えて公開ツール数にするため。
// help topic やツールの説明文でこの名前を出さないこと（help_topics_test が落ちるうえ、
// AI に見えないツールを案内することになる）。

import "github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"

// deleteSkillTool は gkill_delete_skill の定義（公開していない。冒頭を参照）。
// 丸ごと削除は画面だけに置くので、path は必須で SKILL.md は消せない（gkill_server が 400 で断る）。
var deleteSkillTool = tool(
	"gkill_delete_skill",
	"Delete one file of a skill (not SKILL.md; deleting a whole skill is done by the user from gkill's settings screen). "+
		"Pass the revision gkill_get_skill returned for the file; a revision that no longer matches is rejected. "+
		"There is no undo: gkill keeps no history of skills, so agree on the deletion with the user first.",
	schema(jsonobj.Obj(
		"name", jsonobj.Obj("type", "string", "description", "Skill name, as listed by gkill_get_skill_list."),
		"path", jsonobj.Obj("type", "string", "description", "File inside the skill, '/'-separated. SKILL.md cannot be deleted."),
		"revision", jsonobj.Obj("type", "string", "description", "Revision of the version you read (from gkill_get_skill)."),
		"locale_name", jsonobj.Obj("type", "string", "description", localeNameDesc),
	), []string{"name", "path", "revision"}),
)

// NormalizeDeleteSkillArgs は gkill_delete_skill の引数。path と revision も必須
// （path を省くと gkill_server はスキルを丸ごと消すので、MCP からは絶対に省かせない）。
func NormalizeDeleteSkillArgs(args any) (*jsonobj.Object, error) {
	return normalizeSkillArgs(args, []string{"name", "path", "revision", "locale_name"}, []string{"name", "path", "revision"}, nil)
}

// handleDeleteSkill は gkill_delete_skill の本体（公開していない。冒頭を参照）。
func handleDeleteSkill(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeDeleteSkillArgs(args)
	if err != nil {
		return nil, err
	}
	name := jsString(normalized.Value("name"))
	path := jsString(normalized.Value("path"))
	if _, err := ctx.callApi("/api/delete_skill", localeBody(normalized,
		"name", name,
		"path", path,
		"revision", normalized.Value("revision"),
	)); err != nil {
		return nil, err
	}
	return jsonobj.Obj("name", name, "deleted_path", path), nil
}
