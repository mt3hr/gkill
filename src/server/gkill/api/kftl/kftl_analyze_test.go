package kftl

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// ─── Analyze（書かない入口。ADR-0507）─────────────────────────────────────────
//
// Web のメモ帳は「おかしな行」のピンク表示と未知タグ・未知板名の確認を Analyze に頼る。
// 守るのは「Analyze で通った入力が送信で弾かれない・その逆も無い」で、
// 2つの入口が同じ prepareRequests を通っていることをここで固定する。

func helperAnalyze(t *testing.T, text string) *KFTLAnalysis {
	t.Helper()
	stmt := &KFTLStatement{StatementText: text}
	analysis, err := stmt.Analyze(context.Background(), &user_config.ApplicationConfig{}, "test-user", "test-device", "ja")
	if err != nil {
		t.Fatalf("Analyze がサーバ障害扱いのエラーを返した: %v", err)
	}
	return analysis
}

// 入力エラーの行番号の集合が、送信（GenerateAndExecuteRequests）と一致する。
func TestAnalyze_ReportsTheSameInvalidLinesAsExecute(t *testing.T) {
	cases := []string{
		"/mood",                          // 値の行が無い
		"/mood 8",                        // 同じ行に引数
		"memo\n/mood\n99\n、\n/mood\nabc", // 範囲外・数値でない（2件）
		"/mi\n\n",                        // 値の行が空
		"repeat memo\n？？\n毎日\n？？",        // 繰り返しの回数行が無い
		"？2026-13-45\nmemo",              // 日時の解釈失敗
		// ── 保存マーカー「！」の穴と、書く前の内容検査（ADR-0508）──
		"/timeis\n!\n",           // マーカーは値の行ではない（2026-09-15 まで 200・0件だった）
		"ーら\n！\n",                // 同上（気分値0が書かれていた）
		"/num\n!\n",              // 同上（500 になっていた）
		"",                       // 空メモ
		"\n！\n",                  // 空行だけで保存
		"#tag\n\n!\n",            // タグの後ろに空行だけ
		"memo\n,\n\n!\n",         // 区切りの後ろが空行（先頭のメモも書かれない）
		"/num\ntitle",            // 数値の行が無い
		"/expense\nshop",         // 店名だけ
		"#tag\n!\n",              // 付け先の無いタグ
		"?2026-09-15 10:00",      // 付け先の無い関連時刻
		"memo\n,\n#tag",          // 区切りの後ろにタグだけ
		"/mi\ntitle\nboard\nabc", // 予定日時欄が読めない
		"/end",                   // 打刻終了の題名が無い（DoRequest から移した）
		"/endt\n",                // 打刻終了のタグが無い（同上）
		"~~\nboard\n\n\n\n~~",    // リポストタスクの対象が無い（同上）
	}
	for _, text := range cases {
		t.Run(strings.ReplaceAll(text, "\n", "|"), func(t *testing.T) {
			analysis := helperAnalyze(t, text)
			if len(analysis.InputErrors) == 0 {
				t.Fatalf("Analyze が不正行を返さなかった: %q", text)
			}
			executeErrors := helperSubmitExpectingInputErrors(t, text)
			if len(executeErrors) != len(analysis.InputErrors) {
				t.Fatalf("件数が入口で違う: Analyze=%d Execute=%d", len(analysis.InputErrors), len(executeErrors))
			}
			for i := range executeErrors {
				a, e := analysis.InputErrors[i], executeErrors[i]
				if a.LineNumber != e.LineNumber || a.LineText != e.LineText || a.MessageID != e.MessageID {
					t.Errorf("[%d] Analyze=(line %d %q %s) Execute=(line %d %q %s)",
						i, a.LineNumber, a.LineText, a.MessageID, e.LineNumber, e.LineText, e.MessageID)
				}
			}
		})
	}
}

// 正しい本文ではタグ・板名・件数が列挙され、不正行は空。
func TestAnalyze_ListsTagsBoardsAndCount(t *testing.T) {
	// Mi のブロックは「/mi・タイトル・板名・見積開始・見積終了・期限」の6行固定
	text := strings.Join([]string{
		"memo",
		"。tagA",
		"。tagB",
		"、",
		"/mi",
		"mi title",
		"boardX",
		"", "", "",
		"。tagA",
		"、",
		"/mi",
		"mi title 2",
		"",
	}, "\n")
	analysis := helperAnalyze(t, text)
	if len(analysis.InputErrors) != 0 {
		t.Fatalf("不正行が無いのに InputErrors = %+v", analysis.InputErrors)
	}
	if got, want := strings.Join(analysis.Tags, ","), "tagA,tagB"; got != want {
		t.Errorf("Tags = %q, want %q（重複なし・出現順）", got, want)
	}
	// 記録ごとの組。タグの無い Mi（3件目）は入らない
	if got, want := fmt.Sprint(analysis.TagGroups), "[[tagA tagB] [tagA]]"; got != want {
		t.Errorf("TagGroups = %s, want %s（記録ごと・タグの無い記録は入れない）", got, want)
	}
	// 板名は書いたとおり。書かなかった Mi は既定板へ解決せず、列挙もしない
	if got, want := strings.Join(analysis.MiBoardNames, ","), "boardX"; got != want {
		t.Errorf("MiBoardNames = %q, want %q", got, want)
	}
	if analysis.RecordCount != 3 {
		t.Errorf("RecordCount = %d, want 3（kmemo + Mi 2件）", analysis.RecordCount)
	}
}

// ブロックの中のタグ・板名も列挙される（Web の確認ダイアログは Analyze の結果しか見ない）。
// 2026-09-15 まで TS 側の kftl-submit-emits.test.ts が持っていた
// 「ブロックの中の知らないタグでも送信前に確認を出す」「まだ無い板名なら確認を出す」の対。
func TestAnalyze_ListsTagsAndBoardsInsideBlocks(t *testing.T) {
	t.Run("リポストタスクのブロックの中のタグと板名", func(t *testing.T) {
		analysis := helperAnalyze(t, "メモ\n～～\nまだ無い板\n\n\n\n。ブロックの中のタグ\n～～\n。閉じたあとのタグ")
		if len(analysis.InputErrors) != 0 {
			t.Fatalf("InputErrors = %+v", analysis.InputErrors)
		}
		// 順序はリクエストの登録順（メモ→リポストタスク）で決まるので、集合で見る
		tags := append([]string(nil), analysis.Tags...)
		slices.Sort(tags)
		if got, want := strings.Join(tags, ","), "ブロックの中のタグ,閉じたあとのタグ"; got != want {
			t.Errorf("Tags = %q, want %q", got, want)
		}
		if got, want := strings.Join(analysis.MiBoardNames, ","), "まだ無い板"; got != want {
			t.Errorf("MiBoardNames = %q, want %q", got, want)
		}
	})
	t.Run("支出のブロックの中のタグ", func(t *testing.T) {
		analysis := helperAnalyze(t, "ーん\nコンビニ\nおにぎり\n150\n。食費\nお茶\n120\n。飲み物")
		if len(analysis.InputErrors) != 0 {
			t.Fatalf("InputErrors = %+v", analysis.InputErrors)
		}
		if got, want := strings.Join(analysis.Tags, ","), "食費,飲み物"; got != want {
			t.Errorf("Tags = %q, want %q", got, want)
		}
		// 支払いごとに別の記録なので、組も支払いごと
		if got, want := fmt.Sprint(analysis.TagGroups), "[[食費] [飲み物]]"; got != want {
			t.Errorf("TagGroups = %s, want %s", got, want)
		}
		if analysis.RecordCount != 2 {
			t.Errorf("RecordCount = %d, want 2（支払いごとに1件）", analysis.RecordCount)
		}
	})
}

// 繰り返しは Analyze でも展開されるので、件数と上限がそのまま分かる。
func TestAnalyze_ExpandsRepeatsWithoutRepositories(t *testing.T) {
	analysis := helperAnalyze(t, "repeat memo\n？？\n毎日\n5\n？？")
	if len(analysis.InputErrors) != 0 {
		t.Fatalf("InputErrors = %+v", analysis.InputErrors)
	}
	if analysis.RecordCount != 5 {
		t.Errorf("RecordCount = %d, want 5（毎日×5回）", analysis.RecordCount)
	}
}

// 値の行が空（次の行はあるが空白だけ）でも、ゼロ値を書かずに行エラーになる。
//
// Web（TS）は「空のタイトル/URL/本文は送信エラー」を独自に持っていたが、Go 側の
// requireNextLineText が「次の行が空白だけ」も弾くので、Web を Go に寄せても
// 「/mi の次の行が空」が黙って0件になることはない（ADR-0507）。
func TestStatement_PrefixFollowedByBlankValueLineIsInputError(t *testing.T) {
	for _, prefix := range []string{"/mood", "/num", "/expense", "/url", "/mi", "/start", "/timeis"} {
		for _, blank := range []string{"", " ", "　"} {
			text := prefix + "\n" + blank + "\nmemo"
			inputErrors := helperSubmitExpectingInputErrors(t, text)
			if inputErrors[0].LineNumber != 1 || inputErrors[0].MessageID != "KFTL_REQUIRE_VALUE_LINE_MESSAGE_TITLE" {
				t.Errorf("%q: [0] = line %d %s, want line 1 KFTL_REQUIRE_VALUE_LINE_MESSAGE_TITLE",
					strings.ReplaceAll(text, "\n", "|"), inputErrors[0].LineNumber, inputErrors[0].MessageID)
			}
		}
	}
}

// 本文があれば、後ろに続く空行はエラーにしない（空行は区切りとして普通に書かれる）。
// 本文が空白だけのメモがエラーになるのは TestAnalyze_BlankRecordsAreInputErrors。
func TestAnalyze_BlankKmemoIsNotAnError(t *testing.T) {
	analysis := helperAnalyze(t, "memo\n\n\n")
	if len(analysis.InputErrors) != 0 {
		t.Errorf("本文のある kmemo の後ろの空行がエラーになった: %+v", analysis.InputErrors)
	}
}

// ─── 書く前の内容検査（validateRequestContents。ADR-0508）─────────────────────
//
// 2026-09-15 まで DoRequest が「内容が空なら何も書かずに nil」で済ませていて、
// `ーち` だけ書いて「！」で保存すると 200「保存しました」でタブが閉じ、何も残らなかった
// （旧 Web の TS は ERR9000xx で送信を止めていた）。Analyze で捕まる＝打鍵中にピンクになる。

// 内容が空の記録は、型ごとの文言（旧 Web と同じキー）で、その記録を作った行の番号つきのエラーになる。
func TestAnalyze_BlankRecordsAreInputErrors(t *testing.T) {
	cases := []struct {
		name      string
		text      string
		line      int
		messageID string
	}{
		{"空のテキスト", "", 1, "KFTL_KMEMO_BLANK_SKIP_SAVE_MESSAGE_TITLE"},
		{"空行だけで保存", "\n！\n", 1, "KFTL_KMEMO_BLANK_SKIP_SAVE_MESSAGE_TITLE"},
		{"空白だけの本文", "  \n　\n", 1, "KFTL_KMEMO_BLANK_SKIP_SAVE_MESSAGE_TITLE"},
		{"改行だけの本文は本文 \\n として書かれない", "\n\n！\n", 1, "KFTL_KMEMO_BLANK_SKIP_SAVE_MESSAGE_TITLE"},
		{"タグの後ろに空行だけ", "。タグ\n\n！\n", 2, "KFTL_KMEMO_BLANK_SKIP_SAVE_MESSAGE_TITLE"},
		{"区切りの後ろが空行", "メモ\n、\n\n！\n", 3, "KFTL_KMEMO_BLANK_SKIP_SAVE_MESSAGE_TITLE"},
		{"付け先の無いタグ", "。タグ\n！\n", 1, "KFTL_META_INFO_NO_TARGET_MESSAGE_TITLE"},
		{"付け先の無い関連時刻", "？2026-09-15 10:00", 1, "KFTL_META_INFO_NO_TARGET_MESSAGE_TITLE"},
		{"付け先の無いテキスト開始", "ーー\n！\n", 1, "KFTL_META_INFO_NO_TARGET_MESSAGE_TITLE"},
		{"区切りの後ろにタグだけ", "メモ\n、\n。タグ", 3, "KFTL_META_INFO_NO_TARGET_MESSAGE_TITLE"},
		{"打刻終了の題名が無い", "ーえ", 1, "KFTL_TIMEIS_END_REQUIRE_END_TITLE_MESSAGE_TITLE"},
		{"打刻終了(if exist)の題名が無い", "ーいえ\n！\n", 1, "KFTL_TIMEIS_END_REQUIRE_END_TITLE_MESSAGE_TITLE"},
		{"タグ打刻終了のタグが無い", "ーたえ\n", 1, "KFTL_TIMEIS_END_REQUIRE_END_TAG_MESSAGE_TITLE"},
		{"リポストタスクの対象が無い", "～～\n板\n\n\n\n～～", 1, "NOT_FOUND_MI_REKYOU_TARGET_ERROR_MESSAGE"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			analysis := helperAnalyze(t, c.text)
			if len(analysis.InputErrors) == 0 {
				t.Fatalf("%q がエラーにならなかった（黙って0件になる）", c.text)
			}
			got := analysis.InputErrors[0]
			if got.LineNumber != c.line || got.MessageID != c.messageID {
				t.Errorf("%q: [0] = line %d %s, want line %d %s", c.text, got.LineNumber, got.MessageID, c.line, c.messageID)
			}
		})
	}
}

// 保存マーカー「！」の行は値の行に数えない。
// 2026-09-15 まで `ーち`+「！」は requireNextLineText を素通りし、タイトル空のまま DoRequest に届いて
// 200・0件（`ーら` は気分値0を1件、`ーか` は 500）だった。マーカー無し（保存ボタン）と同じ結果にする。
func TestAnalyze_SaveMarkerLineIsNotAValueLine(t *testing.T) {
	for _, prefix := range []string{"ーち", "ーた", "ーみ", "ーう", "ーん", "ーら", "ーか", "/timeis", "/start", "/mi", "/url", "/expense", "/mood", "/num"} {
		for _, marker := range []string{"！", "!"} {
			text := prefix + "\n" + marker + "\n"
			t.Run(strings.ReplaceAll(text, "\n", "|"), func(t *testing.T) {
				analysis := helperAnalyze(t, text)
				if len(analysis.InputErrors) != 1 {
					t.Fatalf("InputErrors = %+v, want 1件", analysis.InputErrors)
				}
				got := analysis.InputErrors[0]
				if got.LineNumber != 1 || got.MessageID != "KFTL_REQUIRE_VALUE_LINE_MESSAGE_TITLE" {
					t.Errorf("[0] = line %d %s, want line 1 KFTL_REQUIRE_VALUE_LINE_MESSAGE_TITLE", got.LineNumber, got.MessageID)
				}
				if analysis.RecordCount != 0 {
					t.Errorf("RecordCount = %d, want 0", analysis.RecordCount)
				}
			})
		}
	}
	// 値があれば今までどおり通る（マーカーの切り詰めで正常系を壊していない）
	for _, text := range []string{"ーち\nタイトル\n！\n", "メモ\n！\n", "ーら\n5\n！\n", "ーか\nタイトル\n1\n!\n", "！"} {
		t.Run("ok:"+strings.ReplaceAll(text, "\n", "|"), func(t *testing.T) {
			analysis := helperAnalyze(t, text)
			if len(analysis.InputErrors) != 0 {
				t.Errorf("InputErrors = %+v, want 空", analysis.InputErrors)
			}
			if analysis.RecordCount != 1 {
				t.Errorf("RecordCount = %d, want 1", analysis.RecordCount)
			}
		})
	}
}

// 「？時刻」の直後の `ーん` は関連時刻をブロックへ取り込む。取り込んだプロトタイプが map に残って
// 「付け先の無い関連時刻」に誤爆しないこと（KFTLRequestMap.Delete）。支払いは今までどおり数える。
func TestAnalyze_RelatedTimeBeforeExpenseBlockIsNotAnOrphan(t *testing.T) {
	analysis := helperAnalyze(t, "？2026-09-15 10:00\nーん\n店\n品\n100\nお茶\n120")
	if len(analysis.InputErrors) != 0 {
		t.Fatalf("InputErrors = %+v, want 空", analysis.InputErrors)
	}
	if analysis.RecordCount != 2 {
		t.Errorf("RecordCount = %d, want 2（支払い2件。プロトタイプは数えない）", analysis.RecordCount)
	}
}

// 店名だけで支出ブロックが終わると入力エラー。支払いの後ろの空行は今までどおり許す。
func TestAnalyze_ExpenseShopNameNeedsAnItemLine(t *testing.T) {
	for _, text := range []string{"ーん\n店", "ーん\n店\n！\n", "ーん\n店\n\n！\n"} {
		t.Run(strings.ReplaceAll(text, "\n", "|"), func(t *testing.T) {
			analysis := helperAnalyze(t, text)
			if len(analysis.InputErrors) != 1 {
				t.Fatalf("InputErrors = %+v, want 1件", analysis.InputErrors)
			}
			if got := analysis.InputErrors[0]; got.LineNumber != 2 || got.MessageID != "KFTL_REQUIRE_VALUE_LINE_MESSAGE_TITLE" {
				t.Errorf("[0] = line %d %s, want line 2 KFTL_REQUIRE_VALUE_LINE_MESSAGE_TITLE", got.LineNumber, got.MessageID)
			}
		})
	}
	analysis := helperAnalyze(t, "ーん\n店\n品\n100\n\n！\n")
	if len(analysis.InputErrors) != 0 {
		t.Errorf("支払いの後ろの空行がエラーになった: %+v", analysis.InputErrors)
	}
}

// 予定日時欄（見積開始・見積終了・期限）の、空でないのに読めない行は入力エラー。
// ADR-0505 が据え置いた「未設定として握り潰す」を ADR-0508 でやめた。空行は今までどおり未設定。
func TestAnalyze_UnparsableScheduleFieldIsInputError(t *testing.T) {
	cases := []struct {
		name string
		text string
		line int
	}{
		{"Mi の見積開始", "ーみ\nタイトル\n板\nabc", 4},
		{"Mi の期限", "ーみ\nタイトル\n板\n\n\nabc", 6},
		{"6行を埋めずに区切りへ", "ーみ\nタイトル\n板\n、\nメモ", 4},
		{"MiReKyou の見積終了", "メモ\n～～\n板\n\nabc\n\n～～", 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			analysis := helperAnalyze(t, c.text)
			if len(analysis.InputErrors) == 0 {
				t.Fatalf("%q がエラーにならなかった（日付だけ入らず黙って保存される）", c.text)
			}
			got := analysis.InputErrors[0]
			if got.LineNumber != c.line || got.MessageID != "KFTL_TIMEIS_INVALID_PARSE_TIME_ERROR_MESSAGE_TITLE" {
				t.Errorf("[0] = line %d %s, want line %d KFTL_TIMEIS_INVALID_PARSE_TIME_ERROR_MESSAGE_TITLE", got.LineNumber, got.MessageID, c.line)
			}
		})
	}
	analysis := helperAnalyze(t, "ーみ\nタイトル\n板\n\n\n\n。タグ")
	if len(analysis.InputErrors) != 0 {
		t.Errorf("空行で欄を飛ばす書き方がエラーになった: %+v", analysis.InputErrors)
	}
}

// ─── 打刻終了の対象検索に設定の playing 条件を写す ───────────────────────────

func rawJSON(s string) *json.RawMessage {
	raw := json.RawMessage(s)
	return &raw
}

func TestPlayingTimeIsQueryFromConfig(t *testing.T) {
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC)

	t.Run("設定が無ければ実行中の打刻すべて", func(t *testing.T) {
		for _, config := range []*user_config.ApplicationConfig{nil, {}, {PlayingTimeIsJSONData: rawJSON(`{"playing_timeis_find_kyou_query": null}`)}} {
			query := playingTimeIsQueryFromConfig(config, now)
			if query.PlayingTime == nil || !query.PlayingTime.Equal(now) || !query.OnlyLatestData {
				t.Errorf("PlayingTime/OnlyLatestData が入っていない: %+v", query)
			}
			if query.Words != nil || query.NotWords != nil || query.Tags != nil || len(query.HideTags) != 0 {
				t.Errorf("条件が無いのに絞り込みが入っている: %+v", query)
			}
		}
	})

	t.Run("保存された条件の語・タグ・非表示タグを写す（rep 名では絞らない）", func(t *testing.T) {
		config := &user_config.ApplicationConfig{
			PlayingTimeIsJSONData: rawJSON(`{"playing_timeis_find_kyou_query": {
				"keywords": "work -lunch", "words": ["work"], "words_and": true, "not_words": ["lunch"],
				"tags": ["job"], "tags_and": false, "reps": ["should_be_ignored"], "playing_time": "2000-01-01T00:00:00Z"}}`),
			TagStruct: rawJSON(`{"tag_name": "root", "is_force_hide": false, "children": [
				{"tag_name": "secret", "is_force_hide": true, "children": null},
				{"tag_name": "job", "is_force_hide": false, "children": [{"tag_name": "hidden_child", "is_force_hide": true, "children": null}]}]}`),
		}
		query := playingTimeIsQueryFromConfig(config, now)
		if got := strings.Join(query.Words, ","); got != "work" || !query.WordsAnd {
			t.Errorf("Words = %q WordsAnd = %v, want work / true", got, query.WordsAnd)
		}
		if got := strings.Join(query.NotWords, ","); got != "lunch" {
			t.Errorf("NotWords = %q, want lunch", got)
		}
		if got := strings.Join(query.Tags, ","); got != "job" || query.TagsAnd {
			t.Errorf("Tags = %q TagsAnd = %v, want job / false", got, query.TagsAnd)
		}
		if got := strings.Join(query.HideTags, ","); got != "secret,hidden_child" {
			t.Errorf("HideTags = %q, want secret,hidden_child（is_force_hide を深さ優先で集める）", got)
		}
		if query.Reps != nil {
			t.Errorf("Reps = %v, want nil（rep 名で絞らない。TS も reps = null）", query.Reps)
		}
		// 基準時刻は保存値ではなく呼び出し側の「今」
		if query.PlayingTime == nil || !query.PlayingTime.Equal(now) {
			t.Errorf("PlayingTime = %v, want %v", query.PlayingTime, now)
		}
	})

	t.Run("旧形式（use_*）の保存値も null 意味論へ揃えてから写す", func(t *testing.T) {
		config := &user_config.ApplicationConfig{
			PlayingTimeIsJSONData: rawJSON(`{"playing_timeis_find_kyou_query": {
				"use_words": false, "words": ["stale"], "use_tags": true, "tags": ["job"]}}`),
		}
		query := playingTimeIsQueryFromConfig(config, now)
		if query.Words != nil {
			t.Errorf("use_words=false なのに Words = %v（語で絞ってはいけない）", query.Words)
		}
		if got := strings.Join(query.Tags, ","); got != "job" {
			t.Errorf("Tags = %q, want job", got)
		}
	})

	t.Run("壊れた JSON は従来どおり全件", func(t *testing.T) {
		config := &user_config.ApplicationConfig{PlayingTimeIsJSONData: rawJSON(`{not json`)}
		query := playingTimeIsQueryFromConfig(config, now)
		if query.Words != nil || query.Tags != nil {
			t.Errorf("壊れた設定から条件が出た: %+v", query)
		}
	})
}

// タグ条件があるときは FindKyous（Kyou 検索の全経路）の結果と突き合わせ、無いときは rep の結果をそのまま返す。
func TestFindPlayingTimeIsEntries_TagFilterGoesThroughFindKyous(t *testing.T) {
	var findKyousCalls int
	base := &KFTLRequestBase{Ctx: &KFTLStatementLineContext{
		ApplicationConfig: &user_config.ApplicationConfig{
			PlayingTimeIsJSONData: rawJSON(`{"playing_timeis_find_kyou_query": {"tags": ["job"]}}`),
		},
		Repositories: &reps.GkillRepositories{},
		FindKyous: func(_ context.Context, query *find.FindQuery) ([]reps.Kyou, error) {
			findKyousCalls++
			if got := strings.Join(query.Tags, ","); got != "job" {
				t.Errorf("FindKyous に渡った Tags = %q, want job", got)
			}
			return nil, nil
		},
	}}
	// TimeIsReps が空でも FindTimeIs は空を返す。タグ条件があるので閉包が1回呼ばれる。
	entries, err := findPlayingTimeIsEntries(context.Background(), base)
	if err != nil {
		t.Fatalf("findPlayingTimeIsEntries error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %v, want 空", entries)
	}
	if findKyousCalls != 1 {
		t.Errorf("FindKyous の呼び出し回数 = %d, want 1（タグ条件があるときは Kyou 検索の層を通す）", findKyousCalls)
	}

	// タグ条件が無ければ閉包は呼ばれない
	findKyousCalls = 0
	base.Ctx.ApplicationConfig = &user_config.ApplicationConfig{}
	if _, err := findPlayingTimeIsEntries(context.Background(), base); err != nil {
		t.Fatalf("findPlayingTimeIsEntries error: %v", err)
	}
	if findKyousCalls != 0 {
		t.Errorf("タグ条件が無いのに FindKyous が %d 回呼ばれた", findKyousCalls)
	}
}

// 終了対象の候補は**開始時刻の新しい順**で、**削除済みを含まない**。
//
// 呼び出し側（ーえ / ーたえ 系）は先頭から一致した1件だけを終えるので、この並びがそのまま「どれを終えるか」になる。
// TimeIsReps.FindTimeIs は map 由来で順序を保証せず削除済みも落とさない（2026-09-16 まで Go はそれをそのまま先頭から
// 取っていて、同じタグの終え忘れが N 件あると 1/N でしか当たらず、削除済みの打刻に終了を書くこともあった。ADR-0509）。
// 並びの検査は修正前だと 1/24（4件の順列）でしか通らず、削除済みの検査は修正前は必ず落ちる。
func TestFindPlayingTimeIsEntries_NewestFirstAndSkipsDeleted(t *testing.T) {
	ctx := context.Background()
	timeIsRep, err := reps.NewTimeIsRepositorySQLite3Impl(ctx, filepath.Join(t.TempDir(), "timeis.db"), true)
	if err != nil {
		t.Fatalf("NewTimeIsRepositorySQLite3Impl: %v", err)
	}
	t.Cleanup(func() { _ = timeIsRep.Close(ctx) })

	base := time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
	running := func(id string, start time.Time, isDeleted bool) reps.TimeIs {
		return reps.TimeIs{
			ID: id, Title: "playing", StartTime: start, IsDeleted: isDeleted,
			CreateTime: start, UpdateTime: start,
			CreateApp: "test", CreateDevice: "test", CreateUser: "test",
			UpdateApp: "test", UpdateDevice: "test", UpdateUser: "test",
		}
	}
	// 追加順はわざと新旧を混ぜる（rep の返す順に依存しないことを見る）
	for _, timeis := range []reps.TimeIs{
		running("second-oldest", base.AddDate(0, 1, 0), false),
		running("newest", base.AddDate(0, 3, 0), false),
		running("oldest", base, false),
		running("deleted-running", base.AddDate(0, 4, 0), true),
		running("third", base.AddDate(0, 2, 0), false),
	} {
		if err := timeIsRep.AddTimeIsInfo(ctx, timeis); err != nil {
			t.Fatalf("AddTimeIsInfo(%s): %v", timeis.ID, err)
		}
	}

	req := &KFTLRequestBase{Ctx: &KFTLStatementLineContext{
		Repositories: &reps.GkillRepositories{TimeIsReps: reps.TimeIsRepositories{timeIsRep}},
	}}
	entries, err := findPlayingTimeIsEntries(ctx, req)
	if err != nil {
		t.Fatalf("findPlayingTimeIsEntries: %v", err)
	}
	var ids []string
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	if got, want := strings.Join(ids, ","), "newest,third,second-oldest,oldest"; got != want {
		t.Errorf("候補の並び = %q, want %q（開始時刻の新しい順・削除済み除外）", got, want)
	}
}

// タグの組は記録ごと（Web がタグ履歴へ積む単位）。1記録の中の重複は落とし、
// タグの無い記録は入れず、繰り返しの展開で同じ組が続いても1つにまとめる。
func TestAnalyze_TagGroupsPerRecord(t *testing.T) {
	t.Run("記録ごと・組の中は重複なし", func(t *testing.T) {
		analysis := helperAnalyze(t, "memo1\n。a、b\n。a\n、\nmemo2\n、\nmemo3\n。c")
		if len(analysis.InputErrors) != 0 {
			t.Fatalf("InputErrors = %+v", analysis.InputErrors)
		}
		if got, want := fmt.Sprint(analysis.TagGroups), "[[a b] [c]]"; got != want {
			t.Errorf("TagGroups = %s, want %s", got, want)
		}
	})
	t.Run("繰り返しの複製は1つの組にまとめる", func(t *testing.T) {
		analysis := helperAnalyze(t, "repeat memo\n。r\n？？\n毎日\n3\n？？")
		if len(analysis.InputErrors) != 0 {
			t.Fatalf("InputErrors = %+v", analysis.InputErrors)
		}
		if analysis.RecordCount != 3 {
			t.Fatalf("RecordCount = %d, want 3", analysis.RecordCount)
		}
		if got, want := fmt.Sprint(analysis.TagGroups), "[[r]]"; got != want {
			t.Errorf("TagGroups = %s, want %s", got, want)
		}
	})
	t.Run("タグが1つも無ければ空", func(t *testing.T) {
		analysis := helperAnalyze(t, "plain memo")
		if len(analysis.TagGroups) != 0 {
			t.Errorf("TagGroups = %v, want 空", analysis.TagGroups)
		}
	})
}
