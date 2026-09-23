package gkill_server_api

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// postSkillAPI は path へ body を POST し、ステータスと応答を out へ読み込む。
func postSkillAPI(t *testing.T, url string, body any, out any) int {
	t.Helper()
	resp := postJSON(t, url, body)
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s response: %v", url, err)
	}
	return resp.StatusCode
}

func skillTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		w, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	// 画面は FileReader.readAsDataURL の結果（data URI）をそのまま送る
	return "data:application/zip;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func skillTestManifest(name string, description string) string {
	return "---\nname: " + name + "\ndescription: " + description + "\n---\n# " + name + "\n"
}

func errorCodesOf(errors message.GkillErrors) []string {
	codes := []string{}
	for _, e := range errors {
		codes = append(codes, e.ErrorCode)
	}
	return codes
}

// スキル API（ADR-0634）の一連の流れ。画面の「zip をアップロード → 確認 → 置き換え → 表示 → ダウンロード → 削除」と、
// MCP の「読む → revision 付きで書く」を、ステータスとエラーコードまで固定する。
func TestSkillAPI_Flow(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()
	tsURL := ts.URL
	sessionID := addZipCacheTestUser(t, gkillAPI, "testuser_skill")

	listSkills := func() []*req_res.SkillInfo {
		t.Helper()
		res := &req_res.GetSkillListResponse{}
		status := postSkillAPI(t, tsURL+"/api/get_skill_list", &req_res.GetSkillListRequest{SessionID: sessionID, LocaleName: "en"}, res)
		if status != http.StatusOK || len(res.Errors) != 0 {
			t.Fatalf("get_skill_list: status=%d errors=%+v", status, res.Errors)
		}
		// 空でも null ではなく []
		if res.Skills == nil {
			t.Fatalf("skills is null")
		}
		return res.Skills
	}

	if got := listSkills(); len(got) != 0 {
		t.Fatalf("initial list = %+v", got)
	}

	firstZip := skillTestZip(t, map[string]string{
		"weekly/SKILL.md":            skillTestManifest("weekly", "weekly dashboard"),
		"weekly/references/tags.md":  "# tags\n",
		"weekly/assets/logo.png":     "\x89PNG\x00\x01",
		"weekly/scripts/obsolete.py": "print(1)\n",
	})

	t.Run("dry_run は書き込まずに計画だけ返す", func(t *testing.T) {
		res := &req_res.UploadSkillResponse{}
		status := postSkillAPI(t, tsURL+"/api/upload_skill", &req_res.UploadSkillRequest{SessionID: sessionID, LocaleName: "en", ZipBase64: firstZip, DryRun: true}, res)
		if status != http.StatusOK || len(res.Errors) != 0 {
			t.Fatalf("status=%d errors=%+v", status, res.Errors)
		}
		if res.Applied || res.Plan == nil || !res.Plan.IsNew || res.Plan.Name != "weekly" || len(res.Plan.Added) != 4 {
			t.Fatalf("plan = %+v applied=%v", res.Plan, res.Applied)
		}
		if got := listSkills(); len(got) != 0 {
			t.Fatalf("dry run wrote something: %+v", got)
		}
	})

	t.Run("本番のアップロードでスキルができる", func(t *testing.T) {
		res := &req_res.UploadSkillResponse{}
		status := postSkillAPI(t, tsURL+"/api/upload_skill", &req_res.UploadSkillRequest{SessionID: sessionID, LocaleName: "en", ZipBase64: firstZip}, res)
		if status != http.StatusOK || len(res.Errors) != 0 || !res.Applied {
			t.Fatalf("status=%d errors=%+v applied=%v", status, res.Errors, res.Applied)
		}
		if len(res.Messages) != 1 || res.Messages[0].MessageCode != message.UploadSkillSuccessMessage {
			t.Errorf("messages = %+v", res.Messages)
		}
		skills := listSkills()
		if len(skills) != 1 || skills[0].Name != "weekly" || skills[0].Description != "weekly dashboard" || skills[0].FileCount != 4 {
			t.Fatalf("list = %+v", skills)
		}
	})

	var manifestRevision string
	t.Run("path を省くと SKILL.md とファイル一覧", func(t *testing.T) {
		res := &req_res.GetSkillResponse{}
		status := postSkillAPI(t, tsURL+"/api/get_skill", &req_res.GetSkillRequest{SessionID: sessionID, LocaleName: "en", Name: "weekly"}, res)
		if status != http.StatusOK || res.Skill == nil || res.File != nil {
			t.Fatalf("status=%d res=%+v", status, res)
		}
		if res.Skill.Content != skillTestManifest("weekly", "weekly dashboard") {
			t.Errorf("content = %q", res.Skill.Content)
		}
		manifestRevision = res.Skill.Revision
		paths := []string{}
		for _, file := range res.Skill.Files {
			paths = append(paths, file.Path)
		}
		if !slices.Equal(paths, []string{"SKILL.md", "assets/logo.png", "references/tags.md", "scripts/obsolete.py"}) {
			t.Errorf("paths = %v", paths)
		}
	})

	t.Run("テキストは content、バイナリは content_base64、上限超えは省く", func(t *testing.T) {
		text := &req_res.GetSkillResponse{}
		postSkillAPI(t, tsURL+"/api/get_skill", &req_res.GetSkillRequest{SessionID: sessionID, LocaleName: "en", Name: "weekly", Path: "references/tags.md"}, text)
		if text.File == nil || !text.File.IsText || text.File.Content != "# tags\n" || text.File.ContentBase64 != "" {
			t.Errorf("text file = %+v", text.File)
		}
		binary := &req_res.GetSkillResponse{}
		postSkillAPI(t, tsURL+"/api/get_skill", &req_res.GetSkillRequest{SessionID: sessionID, LocaleName: "en", Name: "weekly", Path: "assets/logo.png"}, binary)
		decoded, _ := base64.StdEncoding.DecodeString(binary.File.ContentBase64)
		if binary.File.IsText || binary.File.Content != "" || string(decoded) != "\x89PNG\x00\x01" {
			t.Errorf("binary file = %+v", binary.File)
		}
		omitted := &req_res.GetSkillResponse{}
		postSkillAPI(t, tsURL+"/api/get_skill", &req_res.GetSkillRequest{SessionID: sessionID, LocaleName: "en", Name: "weekly", Path: "references/tags.md", MaxBytes: 3}, omitted)
		if !omitted.File.ContentOmitted || omitted.File.Content != "" || omitted.File.Size != 7 {
			t.Errorf("omitted file = %+v", omitted.File)
		}
	})

	t.Run("ダウンロードした zip をそのまま上げ直すと変更なし", func(t *testing.T) {
		download := &req_res.DownloadSkillResponse{}
		status := postSkillAPI(t, tsURL+"/api/download_skill", &req_res.DownloadSkillRequest{SessionID: sessionID, LocaleName: "en", Name: "weekly"}, download)
		if status != http.StatusOK || download.FileName != "weekly.zip" || download.ZipBase64 == "" {
			t.Fatalf("status=%d download=%+v", status, download.FileName)
		}
		plan := &req_res.UploadSkillResponse{}
		postSkillAPI(t, tsURL+"/api/upload_skill", &req_res.UploadSkillRequest{SessionID: sessionID, LocaleName: "en", ZipBase64: download.ZipBase64, DryRun: true}, plan)
		if plan.Plan == nil || plan.Plan.IsNew || len(plan.Plan.Added)+len(plan.Plan.Removed)+len(plan.Plan.Changed) != 0 {
			t.Errorf("round trip plan = %+v errors=%+v", plan.Plan, plan.Errors)
		}
	})

	t.Run("write_skill_file は revision で上書きを守る", func(t *testing.T) {
		write := func(path string, content string, revision string) (int, *req_res.WriteSkillFileResponse) {
			res := &req_res.WriteSkillFileResponse{}
			status := postSkillAPI(t, tsURL+"/api/write_skill_file", &req_res.WriteSkillFileRequest{SessionID: sessionID, LocaleName: "en", Name: "weekly", Path: path, Content: content, Revision: revision}, res)
			return status, res
		}
		status, created := write("references/new.md", "new\n", "")
		if status != http.StatusOK || created.Revision == "" || created.Path != "references/new.md" {
			t.Fatalf("create: status=%d res=%+v", status, created)
		}
		status, res := write("references/new.md", "again\n", "")
		if status != http.StatusConflict || !slices.Equal(errorCodesOf(res.Errors), []string{message.SkillFileAlreadyExistsError}) {
			t.Errorf("create over existing: status=%d errors=%+v", status, res.Errors)
		}
		status, res = write("SKILL.md", skillTestManifest("weekly", "v2"), "0000000000000000")
		if status != http.StatusConflict || !slices.Equal(errorCodesOf(res.Errors), []string{message.SkillRevisionConflictError}) {
			t.Fatalf("stale revision: status=%d errors=%+v", status, res.Errors)
		}
		// 食い違ったときは今の revision を文言に添える（AI が読み直さずに済むように）
		if !strings.Contains(res.Errors[0].ErrorMessage, manifestRevision) {
			t.Errorf("conflict message does not carry the current revision: %q", res.Errors[0].ErrorMessage)
		}
		status, res = write("SKILL.md", skillTestManifest("other", "v2"), manifestRevision)
		if status != http.StatusBadRequest || !slices.Equal(errorCodesOf(res.Errors), []string{message.InvalidSkillManifestError}) {
			t.Errorf("mismatched name: status=%d errors=%+v", status, res.Errors)
		}
		status, res = write("SKILL.md", skillTestManifest("weekly", "v2"), manifestRevision)
		if status != http.StatusOK || len(res.Errors) != 0 {
			t.Errorf("update: status=%d errors=%+v", status, res.Errors)
		}
		status, res = write("../escape.md", "x", "")
		if status != http.StatusBadRequest || !slices.Equal(errorCodesOf(res.Errors), []string{message.InvalidSkillFilePathError}) {
			t.Errorf("traversal: status=%d errors=%+v", status, res.Errors)
		}
	})

	t.Run("置き換えの計画に追加・削除・変更が出る", func(t *testing.T) {
		res := &req_res.UploadSkillResponse{}
		postSkillAPI(t, tsURL+"/api/upload_skill", &req_res.UploadSkillRequest{SessionID: sessionID, LocaleName: "en", DryRun: true, ZipBase64: skillTestZip(t, map[string]string{
			"SKILL.md":           skillTestManifest("weekly", "v3"),
			"references/tags.md": "# tags\n",
			"assets/logo.png":    "\x89PNG\x00\x01",
		})}, res)
		if res.Plan == nil {
			t.Fatalf("errors = %+v", res.Errors)
		}
		if !slices.Equal(res.Plan.Removed, []string{"references/new.md", "scripts/obsolete.py"}) || !slices.Equal(res.Plan.Changed, []string{"SKILL.md"}) || len(res.Plan.Added) != 0 {
			t.Errorf("plan = %+v", res.Plan)
		}
	})

	t.Run("不正な zip は理由とパスつきで 400", func(t *testing.T) {
		res := &req_res.UploadSkillResponse{}
		status := postSkillAPI(t, tsURL+"/api/upload_skill", &req_res.UploadSkillRequest{SessionID: sessionID, LocaleName: "en", ZipBase64: skillTestZip(t, map[string]string{
			"SKILL.md":    skillTestManifest("weekly", "v3"),
			"../evil.txt": "x",
		})}, res)
		if status != http.StatusBadRequest || !slices.Equal(errorCodesOf(res.Errors), []string{message.InvalidSkillZipError}) {
			t.Fatalf("status=%d errors=%+v", status, res.Errors)
		}
		if !strings.Contains(res.Errors[0].ErrorMessage, "../evil.txt") {
			t.Errorf("message does not name the bad path: %q", res.Errors[0].ErrorMessage)
		}
		broken := &req_res.UploadSkillResponse{}
		status = postSkillAPI(t, tsURL+"/api/upload_skill", &req_res.UploadSkillRequest{SessionID: sessionID, LocaleName: "en", ZipBase64: "not base64!"}, broken)
		if status != http.StatusBadRequest || !slices.Equal(errorCodesOf(broken.Errors), []string{message.InvalidUploadSkillRequestDataError}) {
			t.Errorf("broken base64: status=%d errors=%+v", status, broken.Errors)
		}
	})

	t.Run("他の利用者からは見えない", func(t *testing.T) {
		otherSession := addZipCacheTestUser(t, gkillAPI, "testuser_skill_other")
		list := &req_res.GetSkillListResponse{}
		postSkillAPI(t, tsURL+"/api/get_skill_list", &req_res.GetSkillListRequest{SessionID: otherSession, LocaleName: "en"}, list)
		if len(list.Skills) != 0 {
			t.Errorf("other user's list = %+v", list.Skills)
		}
		res := &req_res.GetSkillResponse{}
		status := postSkillAPI(t, tsURL+"/api/get_skill", &req_res.GetSkillRequest{SessionID: otherSession, LocaleName: "en", Name: "weekly"}, res)
		if status != http.StatusNotFound || !slices.Equal(errorCodesOf(res.Errors), []string{message.SkillNotFoundError}) {
			t.Errorf("other user's get: status=%d errors=%+v", status, res.Errors)
		}
	})

	t.Run("削除: SKILL.md 単独は不可、ファイル、スキル丸ごと", func(t *testing.T) {
		del := func(path string) (int, *req_res.DeleteSkillResponse) {
			res := &req_res.DeleteSkillResponse{}
			status := postSkillAPI(t, tsURL+"/api/delete_skill", &req_res.DeleteSkillRequest{SessionID: sessionID, LocaleName: "en", Name: "weekly", Path: path}, res)
			return status, res
		}
		status, res := del("SKILL.md")
		if status != http.StatusBadRequest || !slices.Equal(errorCodesOf(res.Errors), []string{message.SkillManifestDeleteError}) {
			t.Errorf("delete manifest: status=%d errors=%+v", status, res.Errors)
		}
		status, res = del("references/new.md")
		if status != http.StatusOK || len(res.Errors) != 0 {
			t.Errorf("delete file: status=%d errors=%+v", status, res.Errors)
		}
		status, res = del("")
		if status != http.StatusOK || len(res.Errors) != 0 || len(res.Messages) != 1 {
			t.Errorf("delete skill: status=%d res=%+v", status, res)
		}
		if got := listSkills(); len(got) != 0 {
			t.Errorf("list after delete = %+v", got)
		}
		status, res = del("")
		if status != http.StatusNotFound || !slices.Equal(errorCodesOf(res.Errors), []string{message.SkillNotFoundError}) {
			t.Errorf("delete missing: status=%d errors=%+v", status, res.Errors)
		}
	})

	t.Run("不正なスキル名は 400", func(t *testing.T) {
		res := &req_res.GetSkillResponse{}
		status := postSkillAPI(t, tsURL+"/api/get_skill", &req_res.GetSkillRequest{SessionID: sessionID, LocaleName: "en", Name: "../weekly"}, res)
		if status != http.StatusBadRequest || !slices.Equal(errorCodesOf(res.Errors), []string{message.InvalidSkillNameError}) {
			t.Errorf("status=%d errors=%+v", status, res.Errors)
		}
	})
}
