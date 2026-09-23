package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/kftl"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleParseKFTLText はKFTL形式のテキストを解析だけして、おかしな行・付くタグ・板名を返します。何も書きません。
//
// POST /api/parse_kftl_text（wrapAuth）
// req_res.ParseKFTLTextRequest / req_res.ParseKFTLTextResponse
//
// Web のメモ帳が打鍵のたびに呼んで「おかしな行」を塗り、保存の直前に呼んで未知タグ・未知板名の
// 確認に使います（ADR-0507）。解析は submit_kftl_text と同じ kftl.KFTLStatement.prepareRequests を
// 通るので、ここで通った入力が送信で弾かれることはありません。DB は読まないので repositories は
// 要りません（wrapAuth）。書き間違いは errors ではなく invalid_lines に載せ、HTTP 200 で返します。
// 解釈には ApplicationConfig が要り、未登録の利用者・端末には既定値を登録してから読み直します。
func (g *GkillServerAPI) HandleParseKFTLText(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.ParseKFTLTextRequest{}
	response := &req_res.ParseKFTLTextResponse{
		InvalidLines: []*req_res.ParseKFTLTextInvalidLine{},
		Tags:         []string{},
		TagGroups:    [][]string{},
		MiBoardNames: []string{},
	}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close request body", "error", fmt.Sprintf("%q", err))
		}
	}()
	defer func() {
		writeErrorStatus(r.Context(), w, response.Errors)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse parse kftl text response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse parse kftl text response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidParseKFTLTextRequestDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SUBMIT_KFTL_TEXT_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse parse kftl text request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse parse kftl text request from json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidParseKFTLTextRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SUBMIT_KFTL_TEXT_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device

	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.ApplicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil || applicationConfig == nil {
		defaultApplicationConfig := user_config.GetDefaultApplicationConfig(userID, device)
		_, err = g.GkillDAOManager.ConfigDAOs.ApplicationConfigDAO.AddApplicationConfig(r.Context(), defaultApplicationConfig)
		if err != nil {
			slog.Log(r.Context(), gkill_log.Warn, "error at add default application config", "error", fmt.Sprintf("%q", err))
		}
		applicationConfig, err = g.GkillDAOManager.ConfigDAOs.ApplicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
		if err != nil || applicationConfig == nil {
			if err != nil {
				err = fmt.Errorf("error at get application config user id = %s device = %s: %w", userID, device, err)
			} else {
				err = fmt.Errorf("error at get application config user id = %s device = %s: application config is nil", userID, device)
			}
			slog.Log(r.Context(), gkill_log.Debug, "error at errorf", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetApplicationConfigError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SUBMIT_KFTL_TEXT_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	statement := &kftl.KFTLStatement{StatementText: request.KFTLText}
	analysis, err := statement.Analyze(r.Context(), applicationConfig, userID, device, request.LocaleName)
	if err != nil {
		err = fmt.Errorf("error at parse kftl text user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse kftl text", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.ParseKFTLTextError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SUBMIT_KFTL_TEXT_MESSAGE"}),
			Cause:        err,
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 書き間違いの文面は submit_kftl_text の errors[].error_message と同じ関数で作る。
	// 打鍵中に見える文と、保存で弾かれたときに見える文が食い違わないようにするため。
	localizer := api.GetLocalizer(request.LocaleName)
	for _, inputError := range analysis.InputErrors {
		response.InvalidLines = append(response.InvalidLines, &req_res.ParseKFTLTextInvalidLine{
			LineNumber: inputError.LineNumber,
			LineText:   inputError.LineText,
			Message:    formatKFTLInputErrorMessage(localizer, inputError),
		})
	}
	if analysis.Tags != nil {
		response.Tags = analysis.Tags
	}
	if analysis.TagGroups != nil {
		response.TagGroups = analysis.TagGroups
	}
	if analysis.MiBoardNames != nil {
		response.MiBoardNames = analysis.MiBoardNames
	}
	response.RecordCount = analysis.RecordCount
}
