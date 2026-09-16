package gkill_server_api

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// discardTxErrorCode は reps.DiscardTx が errors.Join で束ねた失敗から「最初に失敗した種別」の
// コードを引く。ハンドラの正常系（handle_commit_tx_atomic_test.go）は discard が成功するので
// この分岐を一度も通らない。表の抜けは静かに ERR000334（idf_kyou 扱い）へ落ちるので、
// DiscardTx が報告しうる全種別が表に載っていることも合わせて固定する。
func TestDiscardTxErrorCode(t *testing.T) {
	t.Run("束の先頭の種別のコードを返す", func(t *testing.T) {
		joined := errors.Join(
			&reps.DiscardTxError{DataType: "kmemo", Err: errors.New("locked")},
			&reps.DiscardTxError{DataType: "tag", Err: errors.New("locked")},
		)
		// ハンドラは fmt.Errorf("...: %w", err) で1段包んでから渡す
		wrapped := fmt.Errorf("error at discard tx id = x: %w", joined)
		if got := discardTxErrorCode(wrapped); got != message.CommitTxDeleteKmemoError {
			t.Errorf("code = %s, want %s (kmemo)", got, message.CommitTxDeleteKmemoError)
		}
	})

	t.Run("DiscardTxError でない失敗と未知の種別は idf_kyou のコードへ落ちる", func(t *testing.T) {
		if got := discardTxErrorCode(errors.New("plain")); got != message.CommitTxDeleteIDFKyouError {
			t.Errorf("plain error の code = %s, want %s", got, message.CommitTxDeleteIDFKyouError)
		}
		unknown := &reps.DiscardTxError{DataType: "no_such_type", Err: errors.New("x")}
		if got := discardTxErrorCode(unknown); got != message.CommitTxDeleteIDFKyouError {
			t.Errorf("未知の種別の code = %s, want %s", got, message.CommitTxDeleteIDFKyouError)
		}
	})

	// reps.DiscardTx の steps と同じ並び。ここが増えたら discardTxErrorCodes にも1行足す。
	t.Run("DiscardTx が報告しうる全種別が表にある", func(t *testing.T) {
		for _, dataType := range []string{
			"idf_kyou", "kc", "kmemo", "lantana", "mi", "nlog", "notification",
			"rekyou", "mirekyou", "tag", "text", "timeis", "urlog",
		} {
			code, ok := discardTxErrorCodes[dataType]
			if !ok {
				t.Errorf("%s が discardTxErrorCodes に無い（この種別の破棄失敗が idf_kyou のコードで返る）", dataType)
				continue
			}
			if got := discardTxErrorCode(&reps.DiscardTxError{DataType: dataType, Err: errors.New("x")}); got != code {
				t.Errorf("%s の code = %s, want %s", dataType, got, code)
			}
		}
	})
}
