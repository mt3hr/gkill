package mcp

import (
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// スキルのツール（ADR-0634）。gkill_server の /api/*skill* へ何を送り、AI へ何を返すかを固定する。
// 名前・パスの規則そのものは gkill_server の1実装（dao/skills）が検査するので、ここでは送り方と形だけを見る。

func skillFileResponse(file *jsonobj.Object) *jsonobj.Object {
	return obj("skill", nil, "file", file, "errors", arr(), "messages", arr())
}

func TestSkillReadTools(t *testing.T) {
	setup := func() (*mockClient, *Server) {
		client := createReadWriteMockClient()
		return client, NewReadServer(client, nil)
	}

	t.Run("gkill_get_skill_list は一覧をそのまま返し、要約に名前を並べる", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("skills", arr(
			obj("name", "weekly", "description", "d", "file_count", 2, "invalid_reason", ""),
			obj("name", "broken", "description", "", "file_count", 1, "invalid_reason", "SKILL.md is missing"),
		)))
		result, err := serverToolCall(t, server, "gkill_get_skill_list", obj())
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_skill_list")
		expectEqual(t, len(arrAt(t, result, "skills")), 2)
		summary, _ := summarizeSkillPayload("gkill_get_skill_list", result)
		expectEqual(t, summary, "2 skill(s): weekly, broken.")
	})

	t.Run("path を省くと SKILL.md とファイル一覧（max_bytes は送らない）", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("skill", obj("name", "weekly", "content", "---\nname: weekly\n---\n", "revision", "r1", "files", arr(obj("path", "SKILL.md"))), "file", nil))
		result, err := serverToolCall(t, server, "gkill_get_skill", obj("name", " weekly "))
		expectNoError(t, err)
		body := client.calls[0].Body
		expectEqual(t, client.calls[0].Pathname, "/api/get_skill")
		expectEqual(t, body.Value("name"), "weekly")
		expectTrue(t, !body.Defined("path") && !body.Defined("max_bytes"), "path/max_bytes must not be sent: %s", jsonobj.MarshalString(body))
		expectEqual(t, result.Value("revision"), "r1")
		expectEqual(t, len(arrAt(t, result, "files")), 1)
	})

	t.Run("path があればそのファイルを返し、AI へ渡す量は max_file_bytes で抑える", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(skillFileResponse(obj("path", "references/a.md", "size", 3, "is_text", true, "revision", "r2", "content", "abc", "content_base64", "", "content_omitted", false)))
		result, err := serverToolCall(t, server, "gkill_get_skill", obj("name", "weekly", "path", "references/a.md"))
		expectNoError(t, err)
		body := client.calls[0].Body
		expectEqual(t, body.Value("path"), "references/a.md")
		expectEqual(t, body.Value("max_bytes"), MaxIDFFileBytes)
		expectEqual(t, result.Value("content"), "abc")
		expectTrue(t, !result.Defined("file_content_base64") && !result.Defined("content_omitted"), "text file carries binary keys: %s", jsonobj.MarshalString(result))
	})

	t.Run("画像は gkill_get_idf_file と同じく image ブロックで届き、テキスト表現に base64 を載せない", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(skillFileResponse(obj("path", "assets/logo.png", "size", 8, "is_text", false, "revision", "r3", "content", "", "content_base64", "iVBORw0KGgo=", "content_omitted", false)))
		payload, err := serverToolCall(t, server, "gkill_get_skill", obj("name", "weekly", "path", "assets/logo.png"))
		expectNoError(t, err)
		expectEqual(t, payload.Value("mime_type"), "image/png")
		expectEqual(t, payload.Value("is_image"), true)
		result := server.BuildToolResult("gkill_get_skill", payload, false, nil)
		content := arrAt(t, result, "content")
		expectEqual(t, len(content), 2)
		image := objAt(t, content[1])
		expectEqual(t, image.Value("type"), "image")
		expectEqual(t, image.Value("mimeType"), "image/png")
		expectTrue(t, !strings.Contains(firstText(t, result), "iVBORw0KGgo="), "base64 leaked into the text part")
	})

	t.Run("上限を超えたファイルは中身なしで warnings に理由", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(skillFileResponse(obj("path", "assets/big.bin", "size", 20000000, "is_text", false, "revision", "r4", "content", "", "content_base64", "", "content_omitted", true)))
		result, err := serverToolCall(t, server, "gkill_get_skill", obj("name", "weekly", "path", "assets/big.bin"))
		expectNoError(t, err)
		expectEqual(t, result.Value("content_omitted"), true)
		expectTrue(t, !result.Defined("file_content_base64") && !result.Defined("content"), "omitted file carries content: %s", jsonobj.MarshalString(result))
		warnings := arrAt(t, result, "warnings")
		expectTrue(t, len(warnings) == 1 && strings.Contains(warnings[0].(string), "settings screen"), "warnings = %v", warnings)
	})

	t.Run("name は必須、未知の引数は弾く", func(t *testing.T) {
		_, server := setup()
		_, err := serverToolCall(t, server, "gkill_get_skill", obj())
		expectThrowsField(t, nil, err, "name")
		_, err = serverToolCall(t, server, "gkill_get_skill", obj("name", "weekly", "max_bytes", 1))
		expectErrorContains(t, err, "is not supported")
	})
}

func TestSkillWriteTools(t *testing.T) {
	setup := func() (*mockClient, *Server) {
		client := createReadWriteMockClient()
		client.mockResolvedValue(obj("path", "SKILL.md", "revision", "newrev", "errors", arr(), "messages", arr()))
		return client, NewReadWriteServer(client, nil)
	}

	t.Run("gkill_add_skill は frontmatter つきの SKILL.md を revision なし（新規だけ）で書く", func(t *testing.T) {
		client, server := setup()
		result, err := serverToolCall(t, server, "gkill_add_skill", obj("name", "weekly", "description", `週次: "まとめ" # 夜`, "body", "# 手順"))
		expectNoError(t, err)
		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/write_skill_file")
		expectEqual(t, call.Body.Value("path"), "SKILL.md")
		expectTrue(t, !call.Body.Defined("revision"), "add must not send a revision")
		// 説明文の ":" '"' "#" で YAML が壊れないよう、値は JSON の文字列（YAML の二重引用符スカラー）で書く
		expectEqual(t, call.Body.Value("content"), "---\nname: \"weekly\"\ndescription: \"週次: \\\"まとめ\\\" # 夜\"\n---\n# 手順\n")
		expectEqual(t, result.Value("revision"), "newrev")
		summary, _ := summarizeSkillPayload("gkill_add_skill", result)
		expectEqual(t, summary, "Created skill: weekly (SKILL.md revision newrev).")
	})

	t.Run("gkill_update_skill は path 省略で SKILL.md、revision と中身はそのまま送る", func(t *testing.T) {
		client, server := setup()
		result, err := serverToolCall(t, server, "gkill_update_skill", obj("name", "weekly", "content", "  keep leading spaces\n", "revision", "r1"))
		expectNoError(t, err)
		body := client.calls[0].Body
		expectEqual(t, body.Value("path"), "SKILL.md")
		expectEqual(t, body.Value("revision"), "r1")
		expectEqual(t, body.Value("content"), "  keep leading spaces\n")
		expectEqual(t, result.Value("path"), "SKILL.md")

		_, err = serverToolCall(t, server, "gkill_update_skill", obj("name", "weekly", "path", "scripts/run.py", "content", ""))
		expectNoError(t, err)
		body = client.calls[1].Body
		expectEqual(t, body.Value("path"), "scripts/run.py")
		expectTrue(t, !body.Defined("revision"), "revision must be omitted when not given (create only)")
	})

	t.Run("gkill の 409（revision の食い違い）はそのまま AI へ返る", func(t *testing.T) {
		client, server := setup()
		client.mockRejectedValueOnce(NewGkillApiError("API error at /api/write_skill_file: ERR000437: stale (the current revision is abc)", nil))
		_, err := serverToolCall(t, server, "gkill_update_skill", obj("name", "weekly", "content", "x", "revision", "old"))
		expectErrorContains(t, err, "the current revision is abc")
	})

	t.Run("read サーバでは書けない", func(t *testing.T) {
		client := createReadWriteMockClient()
		_, err := serverToolCall(t, NewReadServer(client, nil), "gkill_update_skill", obj("name", "weekly", "content", "x"))
		expectErrorContains(t, err, "Unknown tool")
		expectEqual(t, len(client.calls), 0)
	})
}

// gkill_delete_skill は実装だけして公開しない（skill_delete_tool.go の冒頭）。
func TestSkillDeleteToolIsNotPublished(t *testing.T) {
	t.Run("どのサーバの一覧にも載らない", func(t *testing.T) {
		for _, server := range []*Server{
			NewReadServer(createReadWriteMockClient(), nil),
			NewWriteServer(createReadWriteMockClient(), nil),
			NewReadWriteServer(createReadWriteMockClient(), nil),
		} {
			for _, tool := range server.Tools {
				expectTrue(t, strAt(t, tool, "name") != "gkill_delete_skill", "%s server lists gkill_delete_skill", server.ServerKind)
			}
		}
		expectTrue(t, !IsWriteToolName("gkill_delete_skill"), "gkill_delete_skill is a write tool name")
	})

	t.Run("呼んでも Unknown tool で、gkill へは何も送らない", func(t *testing.T) {
		for _, newServer := range []func(GkillAPI, *Logger) *Server{NewWriteServer, NewReadWriteServer} {
			client := createReadWriteMockClient()
			_, err := serverToolCall(t, newServer(client, nil), "gkill_delete_skill", obj("name", "weekly", "path", "a.md", "revision", "r"))
			expectErrorContains(t, err, "Unknown tool")
			expectEqual(t, len(client.calls), 0)
		}
	})

	t.Run("本体はファイル単位で、path と revision を必ず送る（丸ごと削除にはならない）", func(t *testing.T) {
		client := createReadWriteMockClient()
		ctx := &CallContext{Client: client, SID: "sid"}
		_, err := handleDeleteSkill(ctx, obj("name", "weekly"))
		expectThrowsField(t, nil, err, "path")
		_, err = handleDeleteSkill(ctx, obj("name", "weekly", "path", "a.md"))
		expectThrowsField(t, nil, err, "revision")
		expectEqual(t, len(client.calls), 0)

		result, err := handleDeleteSkill(ctx, obj("name", "weekly", "path", "references/a.md", "revision", "r1"))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/delete_skill")
		expectEqual(t, client.calls[0].Body.Value("path"), "references/a.md")
		expectEqual(t, client.calls[0].Body.Value("revision"), "r1")
		expectEqual(t, result.Value("deleted_path"), "references/a.md")
		expectEqual(t, strAt(t, deleteSkillTool, "name"), "gkill_delete_skill")
	})
}

func TestStatusCarriesSkills(t *testing.T) {
	t.Run("スキルの名前と説明が載り、要約に件数が出る", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockImplementation(func(pathname string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			if pathname == "/api/get_skill_list" {
				return obj("skills", arr(obj("name", "weekly", "description", "d", "file_count", 3, "updated_time", "x", "invalid_reason", ""))), nil
			}
			return obj("application_config", obj("user_id", "testuser", "device", "pc")), nil
		})
		server := NewReadServer(client, nil)
		payload, err := serverToolCall(t, server, "gkill_status", obj())
		expectNoError(t, err)
		expectEqual(t, jsonobj.MarshalString(payload.Value("skills")), `[{"name":"weekly","description":"d"}]`)
		summary, _ := summarizeReadToolPayloadBody("gkill_status", payload)
		expectTrue(t, strings.Contains(summary, "1 skill(s) stored"), "summary = %q", summary)
	})

	t.Run("一覧が取れなくても status は成功し、skills_error を付ける", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockImplementation(func(pathname string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			if pathname == "/api/get_skill_list" {
				return nil, NewGkillApiError("HTTP 503 from /api/get_skill_list.", obj("status", 503))
			}
			return obj("application_config", obj("user_id", "testuser", "device", "pc")), nil
		})
		payload, err := serverToolCall(t, NewReadServer(client, nil), "gkill_status", obj())
		expectNoError(t, err)
		expectEqual(t, payload.Value("gkill_reachable"), true)
		expectTrue(t, payload.Defined("skills_error") && !payload.Defined("skills"), "payload = %s", jsonobj.MarshalString(payload))
		summary, _ := summarizeReadToolPayloadBody("gkill_status", payload)
		expectTrue(t, !strings.Contains(summary, "skill"), "summary mentions skills: %q", summary)
	})
}

// バイナリの mime_type / is_image は拡張子の固定表（skillMimeType）で決める。mime.TypeByExtension は Windows で
// レジストリを読むので端末ごとに結果が変わる。表の引き方（大小無視・.pdf・未知の拡張子）を固定する。
func TestSkillMimeTypeTable(t *testing.T) {
	cases := []struct {
		path     string
		mimeType string
		isImage  bool
	}{
		{"assets/LOGO.PNG", "image/png", true},
		{"assets/photo.jpeg", "image/jpeg", true},
		{"references/spec.pdf", "application/pdf", false},
		{"assets/blob.bin", "application/octet-stream", false},
	}
	for _, c := range cases {
		t.Run(c.path+" は "+c.mimeType, func(t *testing.T) {
			client := createReadWriteMockClient()
			client.mockResolvedValue(skillFileResponse(obj("path", c.path, "size", 4, "is_text", false, "revision", "r", "content", "", "content_base64", "AAAA", "content_omitted", false)))
			payload, err := serverToolCall(t, NewReadServer(client, nil), "gkill_get_skill", obj("name", "weekly", "path", c.path))
			expectNoError(t, err)
			expectEqual(t, payload.Value("mime_type"), c.mimeType)
			expectEqual(t, payload.Value("is_image"), c.isImage)
			expectEqual(t, payload.Value("file_content_base64"), "AAAA")
			expectTrue(t, !payload.Defined("content"), "binary file carries text content: %s", jsonobj.MarshalString(payload))
		})
	}
}

// 画像でないバイナリ（PDF 等）は image ブロックを作れないので、structuredContent の file_content_base64 が唯一の
// バイト列の渡し口になる。画像と同じ扱いで落とすと、エラーも出ないまま中身の無い応答になる。
func TestSkillNonImageBinaryKeepsBase64InStructuredContent(t *testing.T) {
	t.Run("PDF は image ブロックを作らず、structuredContent に file_content_base64 を残す", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockResolvedValue(skillFileResponse(obj("path", "references/spec.pdf", "size", 4, "is_text", false, "revision", "r5", "content", "", "content_base64", "JVBERi0=", "content_omitted", false)))
		server := NewReadServer(client, nil)
		payload, err := serverToolCall(t, server, "gkill_get_skill", obj("name", "weekly", "path", "references/spec.pdf"))
		expectNoError(t, err)
		result := server.BuildToolResult("gkill_get_skill", payload, false, nil)
		content := arrAt(t, result, "content")
		expectEqual(t, len(content), 1)
		expectEqual(t, objAt(t, content[0]).Value("type"), "text")
		expectTrue(t, !strings.Contains(firstText(t, result), "JVBERi0="), "base64 leaked into the text part")
		structured := objAt(t, result, "structuredContent")
		expectEqual(t, structured.Value("file_content_base64"), "JVBERi0=")
		expectEqual(t, structured.Value("mime_type"), "application/pdf")
		expectTrue(t, !structured.Defined("image_content_attached"), "non-image must not claim an image block: %s", jsonobj.MarshalString(structured))
	})

	t.Run("画像は structuredContent からも base64 を落とし、image_content_attached の印を残す", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockResolvedValue(skillFileResponse(obj("path", "assets/logo.png", "size", 8, "is_text", false, "revision", "r3", "content", "", "content_base64", "iVBORw0KGgo=", "content_omitted", false)))
		server := NewReadServer(client, nil)
		payload, err := serverToolCall(t, server, "gkill_get_skill", obj("name", "weekly", "path", "assets/logo.png"))
		expectNoError(t, err)
		result := server.BuildToolResult("gkill_get_skill", payload, false, nil)
		structured := objAt(t, result, "structuredContent")
		expectTrue(t, !structured.Defined("file_content_base64"), "image duplicates base64 in structuredContent")
		expectEqual(t, structured.Value("image_content_attached"), true)
	})
}

// 一覧が空のときの要約。名前を並べる経路と同じ関数なので、空の分岐を消しても "0 skill(s): ." が黙って返るだけになる。
func TestSkillListEmptySummary(t *testing.T) {
	t.Run("skills が [] なら「無い」と言う", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockResolvedValue(obj("skills", arr()))
		result, err := serverToolCall(t, NewReadServer(client, nil), "gkill_get_skill_list", obj())
		expectNoError(t, err)
		expectEqual(t, len(arrAt(t, result, "skills")), 0)
		summary, ok := summarizeSkillPayload("gkill_get_skill_list", result)
		expectTrue(t, ok, "summary not produced")
		expectEqual(t, summary, "No skills are stored for this account.")
	})

	t.Run("skills が無い応答でも [] に揃えて同じ要約", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockResolvedValue(obj("errors", arr(), "messages", arr()))
		result, err := serverToolCall(t, NewReadServer(client, nil), "gkill_get_skill_list", obj())
		expectNoError(t, err)
		expectEqual(t, len(arrAt(t, result, "skills")), 0)
		summary, _ := summarizeSkillPayload("gkill_get_skill_list", result)
		expectEqual(t, summary, "No skills are stored for this account.")
	})
}

// gkill_add_skill の引数の trim の規則。body は Markdown の中身なので trim しない（先頭の空白・インデントも内容。
// normalizeSkillArgs の raw 指定）。raw から外すと AssertTrimmedString が黙って先頭と末尾の空白を落とし、
// 空の body は「必須」として拒まれる。name / description は trim し、trim して空の description は送る前に拒む。
func TestAddSkillArgumentTrimming(t *testing.T) {
	t.Run("先頭の空白と末尾の改行を残し、name / description だけ trim する", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockResolvedValue(obj("path", "SKILL.md", "revision", "newrev", "errors", arr(), "messages", arr()))
		_, err := serverToolCall(t, NewReadWriteServer(client, nil), "gkill_add_skill", obj("name", " weekly ", "description", " d ", "body", "   indented first line\n\n"))
		expectNoError(t, err)
		body := client.calls[0].Body
		expectEqual(t, body.Value("name"), "weekly")
		expectEqual(t, body.Value("content"), "---\nname: \"weekly\"\ndescription: \"d\"\n---\n   indented first line\n\n")
	})

	t.Run("空の body は通り、ヘッダだけの SKILL.md になる", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockResolvedValue(obj("path", "SKILL.md", "revision", "newrev", "errors", arr(), "messages", arr()))
		_, err := serverToolCall(t, NewReadWriteServer(client, nil), "gkill_add_skill", obj("name", "weekly", "description", "d", "body", ""))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Body.Value("content"), "---\nname: \"weekly\"\ndescription: \"d\"\n---\n")
	})

	t.Run("description は空を拒む（gkill へ送らない）", func(t *testing.T) {
		client := createReadWriteMockClient()
		_, err := serverToolCall(t, NewReadWriteServer(client, nil), "gkill_add_skill", obj("name", "weekly", "description", "  ", "body", "x"))
		expectThrowsField(t, nil, err, "description")
		expectEqual(t, len(client.calls), 0)
	})
}
