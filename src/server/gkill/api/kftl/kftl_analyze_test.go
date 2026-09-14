package kftl

import (
	"context"
	"encoding/json"
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

// kmemo の空本文（空行だけ）はエラーにしない。空行は区切りとして普通に書かれる。
func TestAnalyze_BlankKmemoIsNotAnError(t *testing.T) {
	analysis := helperAnalyze(t, "memo\n\n\n")
	if len(analysis.InputErrors) != 0 {
		t.Errorf("空行だけの kmemo がエラーになった: %+v", analysis.InputErrors)
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
