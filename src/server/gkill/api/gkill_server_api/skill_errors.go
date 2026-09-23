package gkill_server_api

import (
	"errors"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/skills"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// skillStoreGkillError は dao/skills のエラーを GkillError へ写す（スキル API の6ハンドラで共通）。
//
// 利用者・AI が直せるもの（名前・パス・frontmatter・zip・revision・存在しない）は 4xx の専用コードへ写し、
// 文言に dao/skills の補足（どのパスが悪いか・今の revision 等）を括弧で添える。
// それ以外（ディスクの失敗・利用者IDがパスに使えない等）は操作ごとの 500（failedCode）で、補足は添えない。
func skillStoreGkillError(err error, localeName string, failedCode string, failedMessageID string) *message.GkillError {
	code := failedCode
	messageID := failedMessageID
	switch {
	case errors.Is(err, skills.ErrSkillNotFound):
		code, messageID = message.SkillNotFoundError, "SKILL_NOT_FOUND_MESSAGE"
	case errors.Is(err, skills.ErrFileNotFound):
		code, messageID = message.SkillFileNotFoundError, "SKILL_FILE_NOT_FOUND_MESSAGE"
	case errors.Is(err, skills.ErrInvalidName):
		code, messageID = message.InvalidSkillNameError, "INVALID_SKILL_NAME_MESSAGE"
	case errors.Is(err, skills.ErrInvalidPath):
		code, messageID = message.InvalidSkillFilePathError, "INVALID_SKILL_FILE_PATH_MESSAGE"
	case errors.Is(err, skills.ErrInvalidFrontmatter):
		code, messageID = message.InvalidSkillManifestError, "INVALID_SKILL_MANIFEST_MESSAGE"
	case errors.Is(err, skills.ErrInvalidZip):
		code, messageID = message.InvalidSkillZipError, "INVALID_SKILL_ZIP_MESSAGE"
	case errors.Is(err, skills.ErrAlreadyExists):
		code, messageID = message.SkillFileAlreadyExistsError, "SKILL_FILE_ALREADY_EXISTS_MESSAGE"
	case errors.Is(err, skills.ErrRevisionConflict):
		code, messageID = message.SkillRevisionConflictError, "SKILL_REVISION_CONFLICT_MESSAGE"
	case errors.Is(err, skills.ErrManifestDelete):
		code, messageID = message.SkillManifestDeleteError, "SKILL_MANIFEST_DELETE_MESSAGE"
	}
	errorMessage := api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: messageID})
	if code != failedCode {
		if detail := skills.DescribeError(err); detail != "" {
			errorMessage += " (" + detail + ")"
		}
	}
	return &message.GkillError{ErrorCode: code, ErrorMessage: errorMessage, Cause: err}
}

// toSkillFileInfo は dao/skills のファイル情報を応答の形にする。
func toSkillFileInfo(file *skills.FileInfo) *req_res.SkillFileInfo {
	return &req_res.SkillFileInfo{
		Path:        file.Path,
		Size:        file.Size,
		IsText:      file.IsText,
		Revision:    file.Revision,
		UpdatedTime: file.UpdatedTime,
	}
}
