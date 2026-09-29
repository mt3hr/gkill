package gkill_server_api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/skills"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// skillSentinelExpectation は dao/skills の番兵1つと、skillStoreGkillError が写す先の期待値。
type skillSentinelExpectation struct {
	name      string
	sentinel  error
	code      string
	status    int
	kind      string
	messageID string
}

// skillSentinelExpectations は skill_errors.go の switch の正本に当たる対応表（9番兵）。
//
// switch の case を1行消しても build も vet も通り、その番兵だけが操作ごとの 500 へ落ちて
// 「無いファイル」が 404 ではなく 500（kind=server）で返るようになる。壊れ方が静かなので 1 つずつ固定する。
var skillSentinelExpectations = []skillSentinelExpectation{
	{"ErrSkillNotFound", skills.ErrSkillNotFound, message.SkillNotFoundError, http.StatusNotFound, message.ErrorKindNotFound, "SKILL_NOT_FOUND_MESSAGE"},
	{"ErrFileNotFound", skills.ErrFileNotFound, message.SkillFileNotFoundError, http.StatusNotFound, message.ErrorKindNotFound, "SKILL_FILE_NOT_FOUND_MESSAGE"},
	{"ErrInvalidName", skills.ErrInvalidName, message.InvalidSkillNameError, http.StatusBadRequest, message.ErrorKindInput, "INVALID_SKILL_NAME_MESSAGE"},
	{"ErrInvalidPath", skills.ErrInvalidPath, message.InvalidSkillFilePathError, http.StatusBadRequest, message.ErrorKindInput, "INVALID_SKILL_FILE_PATH_MESSAGE"},
	{"ErrInvalidFrontmatter", skills.ErrInvalidFrontmatter, message.InvalidSkillManifestError, http.StatusBadRequest, message.ErrorKindInput, "INVALID_SKILL_MANIFEST_MESSAGE"},
	{"ErrInvalidZip", skills.ErrInvalidZip, message.InvalidSkillZipError, http.StatusBadRequest, message.ErrorKindInput, "INVALID_SKILL_ZIP_MESSAGE"},
	{"ErrAlreadyExists", skills.ErrAlreadyExists, message.SkillFileAlreadyExistsError, http.StatusConflict, message.ErrorKindConflict, "SKILL_FILE_ALREADY_EXISTS_MESSAGE"},
	{"ErrRevisionConflict", skills.ErrRevisionConflict, message.SkillRevisionConflictError, http.StatusConflict, message.ErrorKindConflict, "SKILL_REVISION_CONFLICT_MESSAGE"},
	{"ErrManifestDelete", skills.ErrManifestDelete, message.SkillManifestDeleteError, http.StatusBadRequest, message.ErrorKindInput, "SKILL_MANIFEST_DELETE_MESSAGE"},
}

// skillFailedCodeExpectation はスキル API 6 本の「操作ごとの失敗コード」（500 側）。
type skillFailedCodeExpectation struct {
	name      string
	code      string
	messageID string
}

var skillFailedCodeExpectations = []skillFailedCodeExpectation{
	{"get_skill_list", message.GetSkillListError, "FAILED_GET_SKILL_LIST_MESSAGE"},
	{"get_skill", message.GetSkillError, "FAILED_GET_SKILL_MESSAGE"},
	{"download_skill", message.DownloadSkillError, "FAILED_DOWNLOAD_SKILL_MESSAGE"},
	{"upload_skill", message.UploadSkillError, "FAILED_UPLOAD_SKILL_MESSAGE"},
	{"write_skill_file", message.WriteSkillFileError, "FAILED_WRITE_SKILL_FILE_MESSAGE"},
	{"delete_skill", message.DeleteSkillError, "FAILED_DELETE_SKILL_MESSAGE"},
}

const skillErrorTestLocale = "en"

// localizedSkillMessage は messageID の文言（テストの locale）を返す。
func localizedSkillMessage(t *testing.T, messageID string) string {
	t.Helper()
	text := api.GetLocalizer(skillErrorTestLocale).MustLocalizeMessage(&i18n.Message{ID: messageID})
	if text == "" {
		t.Fatalf("i18n message %q is empty", messageID)
	}
	return text
}

// marshaledSkillError は GkillError をワイヤの形（error_kind / reason 付き）へ通して読み戻す。
func marshaledSkillError(t *testing.T, gkillError *message.GkillError) map[string]string {
	t.Helper()
	raw, err := json.Marshal(gkillError)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	wire := map[string]string{}
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return wire
}

// assertSkillErrorMapped は番兵由来の err が期待のコード・ステータス・種類へ写り、Cause がそのまま残ることを確かめる。
func assertSkillErrorMapped(t *testing.T, err error, tc skillSentinelExpectation, failed skillFailedCodeExpectation) *message.GkillError {
	t.Helper()
	got := skillStoreGkillError(err, skillErrorTestLocale, failed.code, failed.messageID)
	if got == nil {
		t.Fatalf("skillStoreGkillError returned nil")
	}
	if got.ErrorCode != tc.code {
		t.Errorf("error_code = %s, want %s", got.ErrorCode, tc.code)
	}
	if status := message.HTTPStatusOf(got.ErrorCode); status != tc.status {
		t.Errorf("HTTPStatusOf(%s) = %d, want %d", got.ErrorCode, status, tc.status)
	}
	// reason の分類と gkill_error.log の causes の源。落ちても応答は返るので目の前では気付かない
	if got.Cause != err {
		t.Errorf("Cause = %v, want the error passed in (%v)", got.Cause, err)
	}
	wire := marshaledSkillError(t, got)
	if wire["error_kind"] != tc.kind {
		t.Errorf("error_kind = %q, want %q", wire["error_kind"], tc.kind)
	}
	// 番兵は fs.ErrNotExist を包まない約束（dao/skills/errors.go）。包むと 404 に storage_unavailable が付く
	if reason, ok := wire["reason"]; ok {
		t.Errorf("reason = %q, want none (a skill sentinel must not be classified as a storage failure)", reason)
	}
	return got
}

// TestSkillStoreGkillError_Sentinels は 9 番兵それぞれについて、素のまま・DetailError で包んだもの・
// fmt.Errorf("%w") でさらに包んだもの（ハンドラが実際に渡す形）の 3 形が同じ 4xx へ写ることを固定する。
func TestSkillStoreGkillError_Sentinels(t *testing.T) {
	failed := skillFailedCodeExpectations[1] // get_skill
	for _, tc := range skillSentinelExpectations {
		t.Run(tc.name, func(t *testing.T) {
			base := localizedSkillMessage(t, tc.messageID)

			t.Run("素の番兵は補足なし", func(t *testing.T) {
				got := assertSkillErrorMapped(t, tc.sentinel, tc, failed)
				if got.ErrorMessage != base {
					t.Errorf("error_message = %q, want exactly %q (no supplement for a bare sentinel)", got.ErrorMessage, base)
				}
			})

			t.Run("DetailError は補足（どこが悪いか）を文言に添える", func(t *testing.T) {
				err := &skills.DetailError{Kind: tc.sentinel, Detail: "detail-marker", Paths: []string{"references/marker.md"}}
				got := assertSkillErrorMapped(t, err, tc, failed)
				if !strings.HasPrefix(got.ErrorMessage, base) {
					t.Errorf("error_message = %q, want prefix %q", got.ErrorMessage, base)
				}
				if !strings.Contains(got.ErrorMessage, "detail-marker") || !strings.Contains(got.ErrorMessage, "references/marker.md") {
					t.Errorf("error_message = %q, want the detail and the path", got.ErrorMessage)
				}
			})

			t.Run("fmt.Errorf で包んでも同じ（ハンドラが渡す形）", func(t *testing.T) {
				inner := &skills.DetailError{Kind: tc.sentinel, Detail: "detail-marker", Paths: []string{"references/marker.md"}}
				err := fmt.Errorf("error at get skill user id = %s name = %q: %w", "testuser", "weekly", inner)
				got := assertSkillErrorMapped(t, err, tc, failed)
				if !strings.HasPrefix(got.ErrorMessage, base) {
					t.Errorf("error_message = %q, want prefix %q", got.ErrorMessage, base)
				}
				if !strings.Contains(got.ErrorMessage, "detail-marker") || !strings.Contains(got.ErrorMessage, "references/marker.md") {
					t.Errorf("error_message = %q, want the detail and the path", got.ErrorMessage)
				}
				// ハンドラの前置き（利用者ID等）は利用者向けの文言へ漏らさない
				if strings.Contains(got.ErrorMessage, "error at get skill") {
					t.Errorf("error_message = %q leaks the wrapping prefix", got.ErrorMessage)
				}
			})
		})
	}
}

// TestSkillStoreGkillError_UnknownFallsToFailedCode は番兵でないエラーが操作ごとの失敗コード（500）へ落ち、
// 補足を添えないことを 6 本の API ぶん固定する。
func TestSkillStoreGkillError_UnknownFallsToFailedCode(t *testing.T) {
	unknownErrors := []struct {
		name string
		err  error
	}{
		{"errors.New", errors.New("disk failure")},
		// 番兵ではあるが写す先が無い（利用者IDがパスに使えない）。DetailError の補足も添えない
		{"ErrInvalidUserID with detail", &skills.DetailError{Kind: skills.ErrInvalidUserID, Detail: "detail-marker"}},
		// 素の fs.ErrNotExist は番兵ではない（保存層が番兵へ変換する約束）。404 へ写さず 500 のまま
		{"fs.ErrNotExist", fmt.Errorf("open skills dir: %w", fs.ErrNotExist)},
	}
	for _, failed := range skillFailedCodeExpectations {
		t.Run(failed.name, func(t *testing.T) {
			base := localizedSkillMessage(t, failed.messageID)
			for _, unknown := range unknownErrors {
				t.Run(unknown.name, func(t *testing.T) {
					got := skillStoreGkillError(unknown.err, skillErrorTestLocale, failed.code, failed.messageID)
					if got.ErrorCode != failed.code {
						t.Errorf("error_code = %s, want %s", got.ErrorCode, failed.code)
					}
					if status := message.HTTPStatusOf(got.ErrorCode); status != http.StatusInternalServerError {
						t.Errorf("HTTPStatusOf(%s) = %d, want 500", got.ErrorCode, status)
					}
					if got.ErrorMessage != base {
						t.Errorf("error_message = %q, want exactly %q (no supplement on the failed code)", got.ErrorMessage, base)
					}
					if got.Cause != unknown.err {
						t.Errorf("Cause = %v, want %v", got.Cause, unknown.err)
					}
					if wire := marshaledSkillError(t, got); wire["error_kind"] != message.ErrorKindServer {
						t.Errorf("error_kind = %q, want %q", wire["error_kind"], message.ErrorKindServer)
					}
				})
			}
		})
	}
}
