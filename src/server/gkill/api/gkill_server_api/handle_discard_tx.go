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

// HandleDiscardTX は指定TXIDの未確定データを全temp repから破棄します。
//
// POST /api/discard_tx（wrapAuthRepos）
// req_res.DiscardTxRequest / req_res.DiscardTxResponse
//
// 削除対象は (txID, userID, device) に一致する行だけなので、他人・他端末の未確定データは消しません。
// 1種別で失敗しても止めずに全種別を試し、最初に失敗した種別のコードで返します。
// 成功してもMessagesには何も積まないため、呼び出し側はErrorsが空かどうかで判断します。
// commit が成功したときは HandleCommitTx が temp rep の行を消すので、呼ぶのは失敗したときだけです。
func (g *GkillServerAPI) HandleDiscardTX(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	request := &req_res.DiscardTxRequest{}
	response := &req_res.DiscardTxResponse{}

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
			err = fmt.Errorf("error at parse discart tx response to json: %w", err)
			slog.Log(r.Context(), gkill_log.Debug, "error at parse discart tx response to json", "error", fmt.Sprintf("%q", err))
			gkillError := &message.GkillError{
				ErrorCode:    message.AccountInvalidDiscardTxResponseDataError,
				ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DISCARD_TRANSACTION_MESSAGE"}),
				Cause:        err,
			}
			response.Errors = append(response.Errors, gkillError)
			return
		}
	}()

	err := json.NewDecoder(r.Body).Decode(request)
	if err != nil {
		err = fmt.Errorf("error at parse discard tx request to json: %w", err)
		slog.Log(r.Context(), gkill_log.Debug, "error at parse discard tx request to json", "error", fmt.Sprintf("%q", err))
		gkillError := &message.GkillError{
			ErrorCode:    message.AccountInvalidDiscardTxRequestDataError,
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "FAILED_DISCARD_TRANSACTION_MESSAGE"}),
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

	// 1種別が失敗しても止めずに全種別を試す（止めると残りの種別の未確定データが残り続ける）。
	// 応答のコードは従来どおり種別別（ERR000334〜345・401）で、最初に失敗した種別のものを載せる。
	err = repositories.DiscardTx(r.Context(), txID, userID, device)
	if err != nil {
		err = fmt.Errorf("error at discard tx id %s user id = %s device = %s: %w", txID, userID, device, err)
		slog.Log(r.Context(), gkill_log.Debug, "error at discard tx", "error", fmt.Sprintf("%q", err))
		response.Errors = append(response.Errors, &message.GkillError{
			ErrorCode:    discardTxErrorCode(err),
			ErrorMessage: api.GetLocalizer(request.LocaleName).MustLocalizeMessage(&i18n.Message{ID: "INTERNAL_SERVER_ERROR_MESSAGE"}),
			Cause:        err,
		})
		return
	}
}

// discardTxErrorCodes は temp rep の破棄に失敗した種別ごとのエラーコード。
var discardTxErrorCodes = map[string]string{
	"idf_kyou":     message.CommitTxDeleteIDFKyouError,
	"kc":           message.CommitTxDeleteKCError,
	"kmemo":        message.CommitTxDeleteKmemoError,
	"lantana":      message.CommitTxDeleteLantanaError,
	"mi":           message.CommitTxDeleteMiError,
	"nlog":         message.CommitTxDeleteNlogError,
	"notification": message.CommitTxDeleteNotificationError,
	"rekyou":       message.CommitTxDeleteReKyouError,
	"mirekyou":     message.CommitTxGetMiReKyouError,
	"tag":          message.CommitTxDeleteTagError,
	"text":         message.CommitTxDeleteTextError,
	"timeis":       message.CommitTxDeleteTimeIsError,
	"urlog":        message.CommitTxDeleteURLogError,
}

// discardTxErrorCode は reps.DiscardTx の束ねたエラーから、最初に失敗した種別のコードを引く。
func discardTxErrorCode(err error) string {
	var discardErr *reps.DiscardTxError
	if errors.As(err, &discardErr) {
		if code, ok := discardTxErrorCodes[discardErr.DataType]; ok {
			return code
		}
	}
	return message.CommitTxDeleteIDFKyouError
}
