package gkill_server_api

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
