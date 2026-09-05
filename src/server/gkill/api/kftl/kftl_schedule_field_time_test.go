package kftl

import (
	"context"
	"testing"
	"time"
)

// helperApplyReturningError は行を適用し、最初に返った失敗をそのまま返す。
// helperApplyToRequestMap は失敗で t.Fatalf するので、失敗そのものを検査したいときはこちら。
func helperApplyReturningError(t *testing.T, text string) error {
	t.Helper()
	lines := helperGenerateLines(t, text)
	requestMap := NewKFTLRequestMap()
	for _, line := range lines {
		if err := line.ApplyThisLineToRequestMap(context.Background(), requestMap); err != nil {
			return err
		}
	}
	return nil
}

// TestParseScheduleFieldTime は Mi / MiReKyou の予定日時欄1行の読み方を固定する。
// クライアント側 src/client/__tests__/unit/kftl/kftl-schedule-field-time.test.ts と対。
func TestParseScheduleFieldTime(t *testing.T) {
	base := time.Date(2026, 8, 20, 21, 30, 0, 0, time.Local)

	t.Run("完全な日時はそのまま読む", func(t *testing.T) {
		got, ok, err := parseScheduleFieldTime("2026-03-15 10:20", base)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("設定されていること")
		}
		want := time.Date(2026, 3, 15, 10, 20, 0, 0, time.Local)
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("時刻のみは base の年月日に載る", func(t *testing.T) {
		got, ok, err := parseScheduleFieldTime("18:00", base)
		if err != nil || !ok {
			t.Fatalf("err=%v ok=%v", err, ok)
		}
		want := time.Date(2026, 8, 20, 18, 0, 0, 0, time.Local)
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("空行と空白のみは未設定でエラーにしない", func(t *testing.T) {
		for _, in := range []string{"", "   ", "\t"} {
			_, ok, err := parseScheduleFieldTime(in, base)
			if err != nil {
				t.Errorf("%q: unexpected error: %v", in, err)
			}
			if ok {
				t.Errorf("%q: 未設定であること", in)
			}
		}
	})

	// 日時として読めない行を一律に行エラーへ倒すと既存の書き方が広範に壊れるので、
	// ここは従来どおり未設定のまま。変えるのは「？」だけ。
	t.Run("読めない行は未設定のままエラーにしない", func(t *testing.T) {
		_, ok, err := parseScheduleFieldTime("not a time", base)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if ok {
			t.Error("未設定であること")
		}
	})

	// 以前は接頭辞として剥がしたうえ、残りのパース失敗を未設定として握り潰していた。
	// 「？？」(繰り返しブロック)の書き損じが無音で消えるのはこの経路。
	t.Run("関連時刻の接頭辞は入力エラーにする", func(t *testing.T) {
		for _, in := range []string{"？18:00", "?18:00", "？", "?", "？？"} {
			_, _, err := parseScheduleFieldTime(in, base)
			if err == nil {
				t.Errorf("%q: エラーになること", in)
				continue
			}
			inputErrors := CollectKFTLInputErrors(err)
			if len(inputErrors) != 1 {
				t.Errorf("%q: 入力エラー1件であること, got %d", in, len(inputErrors))
				continue
			}
			if inputErrors[0].MessageID != "KFTL_SCHEDULE_TIME_PREFIX_NOT_ALLOWED_MESSAGE_TITLE" {
				t.Errorf("%q: MessageID = %q", in, inputErrors[0].MessageID)
			}
		}
	})
}

// Mi の予定日時3欄すべてで「？」が行エラーになること。
// 欄ごとに実装が分かれているので、1欄だけ直した取りこぼしをここで捕まえる。
func TestStatement_MiScheduleFieldsRejectRelatedTimePrefix(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"見積開始", "ーみ\nタスク\n仕事\n？18:00"},
		{"見積終了", "ーみ\nタスク\n仕事\n\n？18:00"},
		{"期限", "ーみ\nタスク\n仕事\n\n\n？18:00"},
		{"ASCII接頭辞", "ーみ\nタスク\n仕事\n?18:00"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := helperApplyReturningError(t, c.text)
			if err == nil {
				t.Fatal("エラーになること")
			}
			inputErrors := CollectKFTLInputErrors(err)
			if len(inputErrors) == 0 {
				t.Fatalf("入力エラーであること: %v", err)
			}
			if inputErrors[0].MessageID != "KFTL_SCHEDULE_TIME_PREFIX_NOT_ALLOWED_MESSAGE_TITLE" {
				t.Errorf("MessageID = %q", inputErrors[0].MessageID)
			}
		})
	}
}

// MiReKyou の予定日時3欄は Go 側では1つの関数を共有しているが、
// 共有をやめたときに気づけるよう3欄とも検査する。
func TestStatement_MiReKyouScheduleFieldsRejectRelatedTimePrefix(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"見積開始", "メモ\n～～\n仕事\n？18:00\n～～"},
		{"見積終了", "メモ\n～～\n仕事\n\n？18:00\n～～"},
		{"期限", "メモ\n～～\n仕事\n\n\n？18:00\n～～"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := helperApplyReturningError(t, c.text)
			if err == nil {
				t.Fatal("エラーになること")
			}
			inputErrors := CollectKFTLInputErrors(err)
			if len(inputErrors) == 0 {
				t.Fatalf("入力エラーであること: %v", err)
			}
			if inputErrors[0].MessageID != "KFTL_SCHEDULE_TIME_PREFIX_NOT_ALLOWED_MESSAGE_TITLE" {
				t.Errorf("MessageID = %q", inputErrors[0].MessageID)
			}
		})
	}
}

// 接頭辞なしの書き方は今までどおり通ること(禁止のとばっちりで壊れていないこと)。
func TestStatement_MiScheduleFieldsAcceptPlainDateTime(t *testing.T) {
	text := "ーみ\nタスク\n仕事\n2026-03-20 09:00\n2026-03-20 10:00\n2026-03-21 18:00"
	requestMap := helperApplyToRequestMap(t, text)
	var found *kftlMiRequest
	for _, req := range requestMap.All() {
		if mi, ok := req.(*kftlMiRequest); ok {
			found = mi
		}
	}
	if found == nil {
		t.Fatal("Miリクエストがあること")
	}
	if found.estimateStartTime == nil || found.estimateEndTime == nil || found.limitTime == nil {
		t.Fatalf("3欄とも設定されていること: start=%v end=%v limit=%v",
			found.estimateStartTime, found.estimateEndTime, found.limitTime)
	}
}
