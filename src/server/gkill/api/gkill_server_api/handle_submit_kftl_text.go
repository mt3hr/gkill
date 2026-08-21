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

// HandleSubmitKFTLText はKFTL形式のテキストを解釈し、生成されたKyouを記録します。
//
// POST /api/submit_kftl_text（wrapAuthRepos）
// req_res.SubmitKFTLTextRequest / req_res.SubmitKFTLTextResponse
//
// 解釈にはテンプレート等を含むApplicationConfigが要ります。未登録の利用者・端末に対しては
// 既定値を登録してから読み直すので、初回リクエストでもエラーにはしません。
// 記録は書き込み用repへ直接行います。リクエストにTXIDはなく、
// commit_tx/discard_txの未確定状態は経由しません。
// 生成されるKyouのCreateApp/UpdateAppは "gkill_kftl" 固定です。
func (g *GkillServerAPI) HandleSubmitKFTLText(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.SubmitKFTLTextRequest{}
	response := &req_res.SubmitKFTLTextResponse{}

	defer func() {
		err := r.Body.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()
	defer func() {
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			err = fmt.Errorf("error at parse submit kftl text response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidSubmitKFTLTextRequestDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SUBMIT_KFTL_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse submit kftl text request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidSubmitKFTLTextRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SUBMIT_KFTL_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// ミドルウェアで設定された認証情報を取得
	auth := AuthFromContext(r.Context())
	userID := auth.UserID
	device := auth.Device
	repositories := auth.Repositories

	// 冪等キー付きの再送で、既に成功済みなら再実行せず成功で返す（二重登録防止）。
	// キーは利用者ごとに名前空間を切る。
	idempotencyKey := ""
	if request.IdempotencyKey != "" {
		idempotencyKey = userID + ":" + request.IdempotencyKey
		if kftlIdempotencyStore.alreadyDone(idempotencyKey) {
			response.Messages = append(response.Messages, &message.GkillMessage{
				MessageCode: message.SubmitKFTLTextSuccessMessage,
				Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_SUBMIT_KFTL_TEXT_MESSAGE"}),
			})
			return
		}
	}

	applicationConfig, err := g.GkillDAOManager.ConfigDAOs.ApplicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
	if err != nil || applicationConfig == nil {
		defaultApplicationConfig := user_config.GetDefaultApplicationConfig(userID, device)
		_, err = g.GkillDAOManager.ConfigDAOs.ApplicationConfigDAO.AddApplicationConfig(r.Context(), defaultApplicationConfig)
		if err != nil {
			slog.Log(r.Context(), gkill_log.Debug, "error at add default application config", "error", fmt.Sprintf("%q", err))
		}
		applicationConfig, err = g.GkillDAOManager.ConfigDAOs.ApplicationConfigDAO.GetApplicationConfig(r.Context(), userID, device)
		if err != nil || applicationConfig == nil {
			if err != nil {
				err = fmt.Errorf("error at get application config user id = %s device = %s: %w", userID, device, err)
			} else {
				err = fmt.Errorf("error at get application config user id = %s device = %s: application config is nil", userID, device)
			}
			slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.GetApplicationConfigError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SUBMIT_KFTL_TEXT_MESSAGE"}),
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	statement := &kftl.KFTLStatement{StatementText: request.KFTLText}
	err = statement.GenerateAndExecuteRequests(
		r.Context(),
		repositories,
		applicationConfig,
		userID,
		device,
		"gkill_kftl",
		request.LocaleName,
	)
	if err != nil {
		err = fmt.Errorf("error at submit kftl text user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.SubmitKFTLTextError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SUBMIT_KFTL_TEXT_MESSAGE"}),
		}
		response.Errors = append(response.Errors, gkillError)
		return
	}

	// 成功したときだけ記録する。以降この利用者の同じキーの再送は再実行されずに畳まれる。
	// 意図的な再送は別メッセージ＝別キーなので畳まれない（監査 S3-wear）。
	if idempotencyKey != "" {
		kftlIdempotencyStore.markDone(idempotencyKey)
	}

	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.SubmitKFTLTextSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_SUBMIT_KFTL_TEXT_MESSAGE"}),
	})
}
