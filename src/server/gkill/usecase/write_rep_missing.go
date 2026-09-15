package usecase

import (
	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

// writeRepMissingError は「その種別の書き込み先 rep が未設定」の GkillError を作る。
//
// 2026-09-15 まで、tx を使わない Add*/Update* は WriteXxxRep が nil のまま AddXxxInfo を呼んで
// nil ポインタ参照で panic し、利用者には「内部エラーが発生しました」しか出なかった。
// 設定→保存先で直せる不備なので、error_kind は config、reason は write_rep_missing で返す
// （どちらもコード WriteRepMissingError から message パッケージの表で決まる）。
// 文言は操作単位の既存 ID（FAILED_ADD_KMEMO_MESSAGE 等）のまま —— 「何をすれば直るか」は
// reason から消費者が引く。
func writeRepMissingError(localeName string, messageID string, dataType string) *message.GkillError {
	return &message.GkillError{
		ErrorCode:    message.WriteRepMissingError,
		ErrorMessage: api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: messageID}),
		Cause:        &message.ReasonError{Msg: "write repository for " + dataType + " is not configured", Reason: message.ReasonWriteRepMissing},
	}
}
