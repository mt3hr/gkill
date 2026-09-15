package gkill_server_api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/kftl"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
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
// 記録はサーバ内で発行した TXID で temp rep に積み、最後に reps.CommitTx が1つの SQLite
// トランザクションで書き込み用repへ確定します。途中で失敗したら DiscardTx して何も残しません
// （2026-09-15 まで実 rep へ直書きで、失敗した行より前の記録が残る設計でした）。
// 生成されるKyouのCreateApp/UpdateAppはリクエストの create_app で、無指定なら "gkill_kftl" です
// （Wear companion は "gkill_wear" を送る。MCP は送らない）。
func (g *GkillServerAPI) HandleSubmitKFTLText(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.SubmitKFTLTextRequest{}
	response := &req_res.SubmitKFTLTextResponse{}

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
			err = fmt.Errorf("error at parse submit kftl text response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse submit kftl text response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.InvalidSubmitKFTLTextRequestDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SUBMIT_KFTL_TEXT_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse submit kftl text request from json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse submit kftl text request from json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.InvalidSubmitKFTLTextRequestDataError,
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
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}

	// create_app / update_app に載せるアプリ名。無指定はメモ帳と同じ "gkill_kftl"。
	// Wear companion は "gkill_wear" を送り、手打ちのメモ帳と区別できるようにしている（2026-09-11）。
	createApp := strings.TrimSpace(request.CreateApp)
	if createApp == "" {
		createApp = "gkill_kftl"
	}

	statement := &kftl.KFTLStatement{
		StatementText: request.KFTLText,
		FindKyous:     g.kftlFindKyousFunc(userID, device),
	}
	createdRecords, err := statement.GenerateAndExecuteRequests(
		r.Context(),
		repositories,
		applicationConfig,
		userID,
		device,
		createApp,
		request.LocaleName,
	)
	// 確定は1つのトランザクションなので、失敗したときは何も残らず created は空になる。
	response.Created = toSubmitKFTLTextCreated(createdRecords)

	if err != nil {
		err = fmt.Errorf("error at submit kftl text user id = %s device = %s: %w", userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at submit kftl text user id", "error", fmt.Sprintf("%q", err))

		// 利用者の書き間違いはサーバ障害と分けて 400 で返し、行番号と原因を載せる。
		// 2026-08-24 まではどちらも ERR000351(500) + 定型文1本に畳まれていて、
		// 何行目の何が悪いのか応答からは一切分からなかった（Wear もこの1文しか読めない）。
		inputErrors := kftl.CollectKFTLInputErrors(err)
		if len(inputErrors) != 0 {
			localizer := api.GetLocalizer(request.LocaleName)
			for _, inputError := range inputErrors {
				response.Errors = append(response.Errors, &message.GkillError{
					ErrorCode:    message.SubmitKFTLTextInvalidInputError,
					ErrorMessage: formatKFTLInputErrorMessage(localizer, inputError),
					Cause:        err,
				})
			}
			return
		}

		gkillError := &message.GkillError{
			ErrorCode:    message.SubmitKFTLTextError,
			ErrorMessage: formatKFTLExecutionErrorMessage(api.GetLocalizer(request.LocaleName), err),
			Cause:        err,
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

// formatKFTLExecutionErrorMessage は実行フェーズの失敗に行番号を添える。
//
// 原因そのもの(DBのエラー等)は端末固有の情報を含みうるので載せない。載せるのは
// 「何行目で止まったか」だけ（行テキストは利用者自身が書いたもの）。確定は1つの
// トランザクションなので何も残らないが、直すべき行は分からないと困る。
func formatKFTLExecutionErrorMessage(localizer *i18n.Localizer, err error) string {
	base := localizer.MustLocalizeMessage(&i18n.Message{ID: "FAILED_SUBMIT_KFTL_TEXT_MESSAGE"})
	var executionError *kftl.KFTLExecutionError
	if errors.As(err, &executionError) && executionError.LineNumber > 0 {
		return fmt.Sprintf("%s (line %d: %q)", base, executionError.LineNumber, executionError.LineText)
	}
	return base
}

// formatKFTLInputErrorMessage は入力エラー1件を利用者向けの1文にする。
//
// 多言語メッセージIDがあればそれを引き、無ければ原因の文面をそのまま添える
// (原因は英語だが、何も出さないよりは追える)。行番号と行テキストは言語に依らない
// 形で挟む —— 利用者が直すのに要るのは「何行目か」で、そこは翻訳しても価値が無い。
func formatKFTLInputErrorMessage(localizer *i18n.Localizer, inputError *kftl.KFTLInputError) string {
	detail := ""
	if inputError.MessageID != "" {
		detail = localizer.MustLocalizeMessage(&i18n.Message{ID: inputError.MessageID})
	} else if inputError.Cause != nil {
		detail = inputError.Cause.Error()
	}
	header := localizer.MustLocalizeMessage(&i18n.Message{ID: "KFTL_FOUND_INVALID_LINE_MESSAGE"})
	if inputError.LineNumber > 0 {
		return fmt.Sprintf("%s (line %d: %q): %s", header, inputError.LineNumber, inputError.LineText, detail)
	}
	return fmt.Sprintf("%s: %s", header, detail)
}

// toSubmitKFTLTextCreated は kftl パッケージの記録を応答DTOへ写す。
func toSubmitKFTLTextCreated(records []kftl.KFTLCreatedRecord) []*req_res.SubmitKFTLTextCreated {
	if len(records) == 0 {
		return nil
	}
	created := make([]*req_res.SubmitKFTLTextCreated, 0, len(records))
	for _, record := range records {
		created = append(created, &req_res.SubmitKFTLTextCreated{
			ID:          record.ID,
			DataType:    record.DataType,
			Updated:     record.Updated,
			RelatedTime: record.RelatedTime,
		})
	}
	return created
}

// kftlFindKyousFunc は KFTL の打刻終了（`/end` 系）が対象を探すときの Kyou 検索。
//
// 設定の playing 検索条件（`playing_timeis_json_data`）のタグ・非表示タグは Kyou 検索の層
// （`api.FindFilter`）でしか効かないので、kftl パッケージには閉包で渡す（kftl → api の import を作らない）。
// 2026-09-15 まで Web だけが設定条件を適用していて、Wear / MCP 経由の `/end` は全 rep から探していた。
func (g *GkillServerAPI) kftlFindKyousFunc(userID, device string) kftl.FindKyousFunc {
	return func(ctx context.Context, query *find.FindQuery) ([]reps.Kyou, error) {
		findFilter := &api.FindFilter{}
		kyous, _, err := findFilter.FindKyous(ctx, userID, device, g.GkillDAOManager, query)
		if err != nil {
			return nil, fmt.Errorf("error at find kyous for kftl user id = %s device = %s: %w", userID, device, err)
		}
		return kyous, nil
	}
}
