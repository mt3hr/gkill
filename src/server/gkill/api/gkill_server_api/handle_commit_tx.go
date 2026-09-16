package gkill_server_api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// HandleCommitTx は指定TXIDの未確定データをtemp repから読み出し、書き込み用repへ確定させます。
//
// POST /api/commit_tx（wrapAuthRepos）
// req_res.CommitTxRequest / req_res.CommitTxResponse
//
// temp repは (txID, userID, device) でスコープされるので、他人・他端末の未確定データは拾いません。
// 確定は reps.CommitTx が行い、関係する書き込み rep のファイルを1接続に ATTACH した
// **1つの SQLite トランザクション**で全種別を追記します。途中で失敗すると ROLLBACK され、
// 実 rep には何も残りません（ERR000419。temp rep の行は残るので discard_tx で捨てるか再 commit できます）。
// 成功したら temp rep の行は消えます（commit は tx を消費します。以前は discard_tx が担当でした）。
// MiReKyouの未確定データがあるのにWriteMiReKyouRepが未設定の場合は書く前にエラーで返します。
// 2026-09-15 までは種別ごとの逐次追記で、途中の種別で失敗すると書けた種別だけが残っていました（部分確定）。
// documents/adr/0219-commit-tx-is-one-sqlite-transaction.md
func (g *GkillServerAPI) HandleCommitTx(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	request := &req_res.CommitTxRequest{}
	response := &req_res.CommitTxResponse{}

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
			err = fmt.Errorf("error at parse commit tx response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse commit tx response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidCommitTxResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SAVE_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse commit tx request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse commit tx request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidCommitTxRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SAVE_MESSAGE"}),
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

	txID := request.TXID

	committed, err := repositories.CommitTx(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at commit tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at commit tx", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, commitTxGkillError(err, request.LocaleName))
		return
	}

	response.Committed = make([]*req_res.CommittedRecord, 0, len(committed))
	for _, record := range committed {
		response.Committed = append(response.Committed, &req_res.CommittedRecord{
			ID:       record.ID,
			DataType: record.DataType,
			Updated:  record.Updated,
		})
	}
	response.Messages = append(response.Messages, &message.GkillMessage{
		MessageCode: message.CommitTxSuccessMessage,
		Message:     api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "SUCCESS_ADD_URLOG_ADDED_GET_MESSAGE"}),
	})
}

// commitTxStageReadErrorCodes は temp rep の読み出しに失敗した種別ごとのエラーコード。
// この段ではまだ何も書いていない。
var commitTxStageReadErrorCodes = map[string]string{
	"idf_kyou":     message.CommitTxGetIDFKyouError,
	"kc":           message.CommitTxGetKCError,
	"kmemo":        message.CommitTxGetKmemoError,
	"lantana":      message.CommitTxGetLantanaError,
	"mi":           message.CommitTxGetMiError,
	"nlog":         message.CommitTxGetNlogError,
	"notification": message.CommitTxGetNotificationError,
	"rekyou":       message.CommitTxGetReKyouError,
	"mirekyou":     message.CommitTxGetMiReKyouError,
	"tag":          message.CommitTxGetTagError,
	"text":         message.CommitTxGetTextError,
	"timeis":       message.CommitTxGetTimeIsError,
	"urlog":        message.CommitTxGetURLogError,
}

// commitTxGkillError は reps.CommitTx の失敗を応答のエラーへ写す。
//
// temp rep の読み出し失敗は従来どおり種別別のコード（ERR000320〜331・401）、それ以外は
// 「確定に失敗し、何も書かれていない」の1コード（ERR000419）に畳む。書き込み rep の未設定も
// 書く前に返るので同じ扱い。
func commitTxGkillError(err error, localeName string) *message.GkillError {
	var stageReadErr *reps.CommitTxStageReadError
	if errors.As(err, &stageReadErr) {
		if code, ok := commitTxStageReadErrorCodes[stageReadErr.DataType]; ok {
			return &message.GkillError{
				ErrorCode:    code,
				ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
				Cause:        err,
			}
		}
	}
	return &message.GkillError{
		ErrorCode:    message.CommitTxRolledBackError,
		ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_SAVE_NOTHING_SAVED_MESSAGE"}),
		Cause:        err,
	}
}
