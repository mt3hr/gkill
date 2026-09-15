package gkill_server_api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// 失敗した応答の1行は writeErrorStatus が出す。
//
// 深部のエラーログは Debug に置いてあり、既定のログレベル（error）では1行も出ない。
// そのままだと「500 が返るが理由がどこにも出ない」に戻るので、ここが唯一の防御線になる。
// レベルはステータスから機械的に決まる。**呼び出し側の善意で決めない。**

// levelCapturingHandler は流れてきたレコードのレベルとメッセージを控えるだけの slog.Handler。
type levelCapturingHandler struct {
	records []slog.Record
}

func (h *levelCapturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *levelCapturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r.Clone())
	return nil
}
func (h *levelCapturingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *levelCapturingHandler) WithGroup(_ string) slog.Handler      { return h }

func captureWriteErrorStatusLog(t *testing.T, errs []*message.GkillError) *levelCapturingHandler {
	t.Helper()
	captured := &levelCapturingHandler{}
	original := slog.Default()
	slog.SetDefault(slog.New(captured))
	t.Cleanup(func() { slog.SetDefault(original) })

	w := httptest.NewRecorder()
	writeErrorStatus(context.Background(), w, errs)
	return captured
}

// TestWriteErrorStatusLogLevel はステータスとログレベルの対応を固定する。
//
// **落ちたら、失敗した応答のレベルが内容と合っていない。**
// 5xx を Warn 以下にすると `--log`（既定 error）で障害の痕跡が残らず、
// 401/403 を Error にすると未ログインのアクセスで gkill_error.log が埋まる。
func TestWriteErrorStatusLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		errorCode string
		want      slog.Level
	}{
		{"5xxはError", message.RepositoriesGetError, gkill_log.Error},
		{"401はWarn", message.AccountSessionNotFoundError, gkill_log.Warn},
		{"403はWarn", message.AccountIsNotEnableError, gkill_log.Warn},
		{"429はWarn", message.LoginRateLimitError, gkill_log.Warn},
		{"その他の4xxはDebug", message.AccountInvalidLoginRequestDataError, gkill_log.Debug},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured := captureWriteErrorStatusLog(t, []*message.GkillError{{ErrorCode: tt.errorCode}})
			if len(captured.records) != 1 {
				t.Fatalf("ログが %d 行。1行だけ出ること", len(captured.records))
			}
			if got := captured.records[0].Level; got != tt.want {
				t.Errorf("error_code=%s のレベルが %v。%v であること（ステータス %d）",
					tt.errorCode, got, tt.want, message.HTTPStatusOf(tt.errorCode))
			}
		})
	}
}

// TestWriteErrorStatusDoesNotLogOnSuccess は成功時に1行も出さないことを確認する。
//
// 成功でも1行出ると、全リクエストぶんの行が積む（アクセスログは別に取ってある）。
func TestWriteErrorStatusDoesNotLogOnSuccess(t *testing.T) {
	captured := captureWriteErrorStatusLog(t, nil)
	if len(captured.records) != 0 {
		t.Errorf("成功時に %d 行出ている。0行であること", len(captured.records))
	}
}

// TestWriteErrorStatusLogCarriesErrorCodes は、出す1行にエラーコードが載ることを確認する。
//
// これが「どのAPIがどの理由で落ちたか」を示す唯一の行になるので、
// コードが載っていないと 500 だったことしか分からない。
func TestWriteErrorStatusLogCarriesErrorCodes(t *testing.T) {
	captured := captureWriteErrorStatusLog(t, []*message.GkillError{{ErrorCode: message.RepositoriesGetError}})
	if len(captured.records) != 1 {
		t.Fatalf("ログが %d 行。1行だけ出ること", len(captured.records))
	}

	foundStatus := false
	foundCodes := false
	captured.records[0].Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "status":
			foundStatus = true
		case "error_codes":
			foundCodes = true
		}
		return true
	})
	if !foundStatus {
		t.Error("status 属性が無い")
	}
	if !foundCodes {
		t.Error("error_codes 属性が無い")
	}
}

// TestWriteErrorStatusLogCarriesRequestInfo は、アクセスログの文脈があるときに
// メソッド・パス・ユーザIDが載ることを確認する。
func TestWriteErrorStatusLogCarriesRequestInfo(t *testing.T) {
	captured := &levelCapturingHandler{}
	original := slog.Default()
	slog.SetDefault(slog.New(captured))
	t.Cleanup(func() { slog.SetDefault(original) })

	ctx, info := newAccessLogContext(context.Background(), http.MethodPost, "/api/get_kyous")
	info.UserID = "testuser"

	w := httptest.NewRecorder()
	writeErrorStatus(ctx, w, []*message.GkillError{{ErrorCode: message.RepositoriesGetError}})

	if len(captured.records) != 1 {
		t.Fatalf("ログが %d 行。1行だけ出ること", len(captured.records))
	}
	got := map[string]string{}
	captured.records[0].Attrs(func(a slog.Attr) bool {
		got[a.Key] = a.Value.String()
		return true
	})
	for _, key := range []string{"method", "path", "user_id"} {
		if _, ok := got[key]; !ok {
			t.Errorf("%s 属性が無い。どのAPIで誰が落ちたかが分からない", key)
		}
	}
}

// TestWriteErrorStatusLogCarriesReasonAndCause は、GkillError.Cause があるときに
// 分類（reasons）と原因の文面（causes）が同じ1行に載ることを確認する。
//
// 2026-09-15 までこの行はコードしか持たず、原因は Debug ログ（既定では出ない）にしか無かった。
// **落ちたら、「500 が返るが理由がどこにも出ない」に戻っている。**
func TestWriteErrorStatusLogCarriesReasonAndCause(t *testing.T) {
	cause := &message.ReasonError{Msg: "write repository for kmemo is not configured", Reason: message.ReasonWriteRepMissing}
	captured := captureWriteErrorStatusLog(t, []*message.GkillError{{ErrorCode: message.AddKmemoError, Cause: cause}})
	if len(captured.records) != 1 {
		t.Fatalf("ログが %d 行。1行だけ出ること", len(captured.records))
	}

	attrs := map[string]string{}
	captured.records[0].Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.String()
		return true
	})
	if got := attrs["reasons"]; got == "" || !strings.Contains(got, message.ReasonWriteRepMissing) {
		t.Errorf("reasons に分類が無い: %q", got)
	}
	if got := attrs["causes"]; got == "" || !strings.Contains(got, "not configured") {
		t.Errorf("causes に原因の文面が無い: %q", got)
	}

	// Cause が無ければ属性自体を出さない（値の無いキーでログを太らせない）
	captured = captureWriteErrorStatusLog(t, []*message.GkillError{{ErrorCode: message.AddKmemoError}})
	captured.records[0].Attrs(func(a slog.Attr) bool {
		if a.Key == "reasons" || a.Key == "causes" {
			t.Errorf("Cause が無いのに %s 属性が出ている", a.Key)
		}
		return true
	})
}

// TestWriteErrorStatusLogCanceledIsDebug は、呼び出し側の中断（reason=canceled）だけの 500 を
// Error ではなく Debug で出すことを固定する。検索欄の打ち直しのたびに gkill_error.log を埋めないため。
// 中断以外が1件でも混ざれば従来どおり Error。
func TestWriteErrorStatusLogCanceledIsDebug(t *testing.T) {
	canceled := &message.GkillError{ErrorCode: message.FindKyousError, Cause: context.Canceled}
	captured := captureWriteErrorStatusLog(t, []*message.GkillError{canceled})
	if len(captured.records) != 1 || captured.records[0].Level != gkill_log.Debug {
		t.Fatalf("中断だけの 500 は Debug のはず: %+v", captured.records)
	}

	mixed := []*message.GkillError{canceled, {ErrorCode: message.FindKyousError, Cause: context.DeadlineExceeded}}
	captured = captureWriteErrorStatusLog(t, mixed)
	if len(captured.records) != 1 || captured.records[0].Level != gkill_log.Error {
		t.Fatalf("中断以外が混ざれば Error のはず: %+v", captured.records)
	}
}
