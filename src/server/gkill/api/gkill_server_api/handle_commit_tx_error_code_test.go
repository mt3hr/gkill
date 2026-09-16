package gkill_server_api

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// commitTxGkillError は reps.CommitTx の失敗を応答のエラーへ写す。ハンドラの経路テスト
// （handle_commit_tx_atomic_test.go）が通るのは ROLLBACK の1コード（ERR000419）だけで、
// temp rep の読み出し失敗を種別別のコードへ写す表（commitTxStageReadErrorCodes）は
// 一度も実行されていなかった。表の抜けは静かに ERR000419 へ畳まれ、Cause が落ちると
// 応答の reason と gkill_error.log の cause が消える（どちらも目の前ではエラーにならない）ので、
// handle_discard_tx_test.go と対称にここで固定する。
func TestCommitTxGkillError(t *testing.T) {
	t.Run("temp rep の読み出し失敗は種別別のコードで返り、Cause に元の error を持つ", func(t *testing.T) {
		for dataType, wantCode := range commitTxStageReadErrorCodes {
			cause := errors.New("database is locked")
			// reps.CommitTx は fmt.Errorf("...: %w", err) で包んで返すことがあるので1段包む
			err := fmt.Errorf("commit tx id = x: %w", &reps.CommitTxStageReadError{DataType: dataType, Err: cause})
			got := commitTxGkillError(err, "en")
			if got.ErrorCode != wantCode {
				t.Errorf("%s: code = %s, want %s", dataType, got.ErrorCode, wantCode)
			}
			if !errors.Is(got.Cause, cause) {
				t.Errorf("%s: Cause が元の error を持っていない（reason とログの cause が消える）: %v", dataType, got.Cause)
			}
		}
	})

	// reps.CommitTx の readStagedTx が報告しうる種別と同じ並び。ここが増えたら表にも1行足す。
	t.Run("読み出し段が報告しうる全種別が表にある", func(t *testing.T) {
		for _, dataType := range []string{
			"idf_kyou", "kc", "kmemo", "lantana", "mi", "nlog", "notification",
			"rekyou", "mirekyou", "tag", "text", "timeis", "urlog",
		} {
			if _, ok := commitTxStageReadErrorCodes[dataType]; !ok {
				t.Errorf("%s が commitTxStageReadErrorCodes に無い（この種別の読み出し失敗が ERR000419 に畳まれる）", dataType)
			}
		}
		if len(commitTxStageReadErrorCodes) != 13 {
			t.Errorf("表の件数 = %d, want 13", len(commitTxStageReadErrorCodes))
		}
	})

	t.Run("読み出し以外の失敗と未知の種別は ERR000419 に畳み、Cause を持つ", func(t *testing.T) {
		for name, err := range map[string]error{
			"書き込み段の失敗":       errors.New("SQLITE_FULL"),
			"未知の種別の読み出し失敗":   &reps.CommitTxStageReadError{DataType: "no_such_type", Err: errors.New("x")},
			"書き込み先 rep の未設定": &reps.CommitTxWriteRepMissingError{DataType: "mirekyou"},
		} {
			got := commitTxGkillError(err, "en")
			if got.ErrorCode != message.CommitTxRolledBackError {
				t.Errorf("%s: code = %s, want %s", name, got.ErrorCode, message.CommitTxRolledBackError)
			}
			if !errors.Is(got.Cause, err) {
				t.Errorf("%s: Cause = %v, want %v", name, got.Cause, err)
			}
		}
		// 書き込み先の未設定は reason で「設定→保存先で直せる」と伝わる（CommitTxWriteRepMissingError が Reasoner）
		got := commitTxGkillError(&reps.CommitTxWriteRepMissingError{DataType: "mirekyou"}, "en")
		if reason := message.ReasonOf(got.Cause); reason != message.ReasonWriteRepMissing {
			t.Errorf("reason = %q, want %q", reason, message.ReasonWriteRepMissing)
		}
	})
}
