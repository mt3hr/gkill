package gkill_server_api

// ワード検索（words / not_words）の型別の照合規則と除外語の適用経路の回帰テスト。
//
// 規則の正本は .claude/skills/gkill-find-query/SKILL.md「ワード検索の照合規則」節と
// documents/adr/0113-word-filter-columns-and-id-prefix.md。
//   - 肯定語は「対象列に含む OR ID が語で始まる」、除外語は「対象列に含まない」（ID は見ない）
//   - Lantana は MOOD、Nlog は AMOUNT、KC は NUM_VALUE、Mi は BOARD_NAME も対象列
//   - 除外語は付随テキスト経由にも効く。語なし・除外語だけなら「素通しした全体から引く」
//   - 空文字・空白だけの語はサーバ入口で捨てる
//
// ファイル名を handle_ で始めていないのは、資料の機械検査（verify_docs）が
// handle_*.go の本数を「ハンドラ数」として数えているため。

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// wordFilterTestNow は本ファイルの記録に共通で使う関連日時。
// 「全件が残る」を確かめる検索が他の記録に引きずられないよう、期間で切れる時刻にしてある。
var wordFilterTestNow = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// wordFilterQuery は本ファイルの記録だけを期間で囲んだ検索条件を組む。
func wordFilterQuery(words []string, notWords []string) *find.FindQuery {
	start := wordFilterTestNow.Add(-time.Hour)
	end := wordFilterTestNow.Add(time.Hour)
	return &find.FindQuery{
		Words:             words,
		NotWords:          notWords,
		CalendarStartDate: &start,
		CalendarEndDate:   &end,
	}
}

func addTestKmemoWithID(t *testing.T, tsURL string, sessionID string, id string, content string) string {
	t.Helper()
	addReq := &req_res.AddKmemoRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		Kmemo: reps.Kmemo{
			ID:          id,
			Content:     content,
			RelatedTime: wordFilterTestNow,
			DataType:    "kmemo",
			CreateTime:  wordFilterTestNow,
			CreateApp:   "test",
			CreateUser:  "admin",
			UpdateTime:  wordFilterTestNow,
			UpdateApp:   "test",
			UpdateUser:  "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_kmemo", addReq)
	defer resp.Body.Close()
	var addResp req_res.AddKmemoResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add kmemo response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add kmemo errors: %+v", addResp.Errors)
	}
	return id
}

func addTestTextTo(t *testing.T, tsURL string, sessionID string, targetID string, text string) {
	t.Helper()
	addReq := &req_res.AddTextRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		Text: reps.Text{
			ID:          GenerateNewID(),
			TargetID:    targetID,
			Text:        text,
			RelatedTime: wordFilterTestNow,
			CreateTime:  wordFilterTestNow,
			CreateApp:   "test",
			CreateUser:  "admin",
			UpdateTime:  wordFilterTestNow,
			UpdateApp:   "test",
			UpdateUser:  "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_text", addReq)
	defer resp.Body.Close()
	var addResp req_res.AddTextResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add text response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add text errors: %+v", addResp.Errors)
	}
}

func addTestLantana(t *testing.T, tsURL string, sessionID string, mood int) string {
	t.Helper()
	id := GenerateNewID()
	addReq := &req_res.AddLantanaRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		Lantana: reps.Lantana{
			ID:          id,
			Mood:        mood,
			RelatedTime: wordFilterTestNow,
			DataType:    "lantana",
			CreateTime:  wordFilterTestNow,
			CreateApp:   "test",
			CreateUser:  "admin",
			UpdateTime:  wordFilterTestNow,
			UpdateApp:   "test",
			UpdateUser:  "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_lantana", addReq)
	defer resp.Body.Close()
	var addResp req_res.AddLantanaResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add lantana response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add lantana errors: %+v", addResp.Errors)
	}
	return id
}

func addTestNlog(t *testing.T, tsURL string, sessionID string, title string, shop string, amount string) string {
	t.Helper()
	id := GenerateNewID()
	addReq := &req_res.AddNlogRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		Nlog: reps.Nlog{
			ID:          id,
			Title:       title,
			Shop:        shop,
			Amount:      json.Number(amount),
			RelatedTime: wordFilterTestNow,
			DataType:    "nlog",
			CreateTime:  wordFilterTestNow,
			CreateApp:   "test",
			CreateUser:  "admin",
			UpdateTime:  wordFilterTestNow,
			UpdateApp:   "test",
			UpdateUser:  "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_nlog", addReq)
	defer resp.Body.Close()
	var addResp req_res.AddNlogResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add nlog response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add nlog errors: %+v", addResp.Errors)
	}
	return id
}

func addTestKC(t *testing.T, tsURL string, sessionID string, title string, numValue string) string {
	t.Helper()
	id := GenerateNewID()
	addReq := &req_res.AddKCRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		KC: reps.KC{
			ID:          id,
			Title:       title,
			NumValue:    json.Number(numValue),
			RelatedTime: wordFilterTestNow,
			DataType:    "kc",
			CreateTime:  wordFilterTestNow,
			CreateApp:   "test",
			CreateUser:  "admin",
			UpdateTime:  wordFilterTestNow,
			UpdateApp:   "test",
			UpdateUser:  "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_kc", addReq)
	defer resp.Body.Close()
	var addResp req_res.AddKCResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add kc response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add kc errors: %+v", addResp.Errors)
	}
	return id
}

func addTestMi(t *testing.T, tsURL string, sessionID string, title string, boardName string) string {
	t.Helper()
	id := GenerateNewID()
	addReq := &req_res.AddMiRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		Mi: reps.Mi{
			ID:         id,
			Title:      title,
			BoardName:  boardName,
			DataType:   "mi",
			CreateTime: wordFilterTestNow,
			CreateApp:  "test",
			CreateUser: "admin",
			UpdateTime: wordFilterTestNow,
			UpdateApp:  "test",
			UpdateUser: "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_mi", addReq)
	defer resp.Body.Close()
	var addResp req_res.AddMiResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add mi response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add mi errors: %+v", addResp.Errors)
	}
	return id
}

func addTestReKyou(t *testing.T, tsURL string, sessionID string, targetID string) string {
	t.Helper()
	id := GenerateNewID()
	addReq := &req_res.AddReKyouRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		ReKyou: reps.ReKyou{
			ID:          id,
			TargetID:    targetID,
			RelatedTime: wordFilterTestNow,
			DataType:    "rekyou",
			CreateTime:  wordFilterTestNow,
			CreateApp:   "test",
			CreateUser:  "admin",
			UpdateTime:  wordFilterTestNow,
			UpdateApp:   "test",
			UpdateUser:  "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_rekyou", addReq)
	defer resp.Body.Close()
	var addResp req_res.AddReKyouResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add rekyou response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add rekyou errors: %+v", addResp.Errors)
	}
	return id
}

// searchWordIDs は検索してヒットした Kyou の ID 集合を返す。エラーがあれば落とす。
func searchWordIDs(t *testing.T, tsURL string, sessionID string, query *find.FindQuery) map[string]bool {
	t.Helper()
	res := getKyousWithQuery(t, tsURL, sessionID, query)
	if len(res.Errors) != 0 {
		t.Fatalf("get kyous errors: %+v", res.Errors)
	}
	return kyouIDSet(res.Kyous)
}

// 型別の対象列: Lantana は MOOD、Nlog は AMOUNT、KC は NUM_VALUE、Mi は BOARD_NAME でも当たる。
func TestHandleGetKyous_WordFilterTypedColumns(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	mood7 := addTestLantana(t, tsURL, sessionID, 7)
	mood3 := addTestLantana(t, tsURL, sessionID, 3)
	nlog := addTestNlog(t, tsURL, sessionID, "昼食", "食堂", "1500")
	kc := addTestKC(t, tsURL, sessionID, "体重", "65.4")
	mi := addTestMi(t, tsURL, sessionID, "買い物", "家事")

	t.Run("Lantana は気分値で当たる", func(t *testing.T) {
		ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{"7"}, []string{}))
		if !ids[mood7] || ids[mood3] {
			t.Errorf("`7` は気分7だけに当たるべき: mood7=%v mood3=%v", ids[mood7], ids[mood3])
		}
	})
	t.Run("Lantana は除外語で消える", func(t *testing.T) {
		ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{}, []string{"7"}))
		if ids[mood7] || !ids[mood3] {
			t.Errorf("`-7` は気分7だけを消すべき: mood7=%v mood3=%v", ids[mood7], ids[mood3])
		}
	})
	t.Run("Nlog は金額で当たる", func(t *testing.T) {
		ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{"1500"}, []string{}))
		if !ids[nlog] {
			t.Errorf("AMOUNT=1500 の支出が `1500` で当たるべき")
		}
	})
	t.Run("KC は数値で当たる", func(t *testing.T) {
		ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{"65.4"}, []string{}))
		if !ids[kc] {
			t.Errorf("NUM_VALUE=65.4 の記録が `65.4` で当たるべき")
		}
	})
	t.Run("Mi は板名で当たる", func(t *testing.T) {
		ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{"家事"}, []string{}))
		if !ids[mi] {
			t.Errorf("BOARD_NAME=家事 のタスクが `家事` で当たるべき")
		}
	})
}

// ID は前方一致だけ。途中の部分一致では当たらず、除外語は ID を見ない。
func TestHandleGetKyous_WordFilterIDPrefix(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	// ID の中身を固定して、語との位置関係を作る
	prefixID := addTestKmemoWithID(t, tsURL, sessionID, "zzprefix-"+GenerateNewID(), "本文には語が無い")
	middleID := addTestKmemoWithID(t, tsURL, sessionID, "mid-zzprefix-"+GenerateNewID(), "本文には語が無い")

	t.Run("ID が語で始まる記録は当たる", func(t *testing.T) {
		ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{"zzprefix"}, []string{}))
		if !ids[prefixID] {
			t.Errorf("ID が語で始まる記録が当たっていない")
		}
		if ids[middleID] {
			t.Errorf("ID の途中に語を含むだけの記録が当たった")
		}
	})
	t.Run("除外語は ID を見ない", func(t *testing.T) {
		ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{"本文"}, []string{"zzprefix"}))
		if !ids[prefixID] || !ids[middleID] {
			t.Errorf("除外語が ID に一致しても消してはいけない: prefix=%v middle=%v", ids[prefixID], ids[middleID])
		}
	})
	t.Run("語なし・除外語だけでも ID は見ない", func(t *testing.T) {
		ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{}, []string{"zzprefix"}))
		if !ids[prefixID] || !ids[middleID] {
			t.Errorf("除外語が ID に一致しても消してはいけない: prefix=%v middle=%v", ids[prefixID], ids[middleID])
		}
	})
}

// 除外語は付随テキスト経由にも効く。
//   - 本体に除外語があり付随テキストが肯定語に当たる記録は出ない（本体側の再検査）
//   - 付随テキストに除外語がある記録は、本体が肯定語に当たっていても出ない
func TestHandleGetKyous_WordFilterNotWordsWithAttachedTexts(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	// 本体 foo / 付随テキストなし → 出る
	plain := addTestKmemo(t, tsURL, sessionID, "foo だけ")
	// 本体 bar / 付随テキスト foo → 出ない（本体に除外語）
	bodyHasNotWord := addTestKmemo(t, tsURL, sessionID, "bar を含む本文")
	addTestTextTo(t, tsURL, sessionID, bodyHasNotWord, "foo の付随テキスト")
	// 本体 foo / 付随テキスト bar → 出ない（付随テキストに除外語）
	textHasNotWord := addTestKmemo(t, tsURL, sessionID, "foo を含む本文")
	addTestTextTo(t, tsURL, sessionID, textHasNotWord, "bar の付随テキスト")
	// 本体なし / 付随テキスト foo → 出る（付随テキスト経由の合流）
	viaText := addTestKmemo(t, tsURL, sessionID, "語の無い本文")
	addTestTextTo(t, tsURL, sessionID, viaText, "foo を持つ付随テキスト")

	ids := searchWordIDs(t, tsURL, sessionID, &find.FindQuery{Words: []string{"foo"}, NotWords: []string{"bar"}})
	if !ids[plain] {
		t.Errorf("本体に foo を含む記録が出ていない")
	}
	if !ids[viaText] {
		t.Errorf("付随テキストに foo を含む記録が出ていない")
	}
	if ids[bodyHasNotWord] {
		t.Errorf("本体に除外語 bar を含む記録が、付随テキストの foo で出てしまった")
	}
	if ids[textHasNotWord] {
		t.Errorf("付随テキストに除外語 bar を含む記録が出てしまった")
	}
}

// 語なし・除外語だけの検索は「素通しした全体から除外語のヒットを引く」。
// 本体に除外語を含む記録・付随テキストに除外語を含む記録・参照先に除外語を含む ReKyou が消え、
// それ以外（Lantana 含む）は全件残る。
func TestHandleGetKyous_WordFilterNotWordsOnlySubtractsFromAll(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	keep := addTestKmemoWithID(t, tsURL, sessionID, GenerateNewID(), "残る本文")
	bodyHasNotWord := addTestKmemoWithID(t, tsURL, sessionID, GenerateNewID(), "bar を含む本文")
	textHasNotWord := addTestKmemoWithID(t, tsURL, sessionID, GenerateNewID(), "残る本文だが付随テキストに除外語")
	addTestTextTo(t, tsURL, sessionID, textHasNotWord, "bar の付随テキスト")
	rekyouOfKeep := addTestReKyou(t, tsURL, sessionID, keep)
	rekyouOfNotWord := addTestReKyou(t, tsURL, sessionID, bodyHasNotWord)
	mood := addTestLantana(t, tsURL, sessionID, 5)

	ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{}, []string{"bar"}))
	for name, id := range map[string]string{"残る本文": keep, "残る本文のReKyou": rekyouOfKeep, "Lantana": mood} {
		if !ids[id] {
			t.Errorf("%s が残っていない（語なし・除外語だけは全体から引くはず）", name)
		}
	}
	for name, id := range map[string]string{"本体に除外語": bodyHasNotWord, "付随テキストに除外語": textHasNotWord, "参照先に除外語があるReKyou": rekyouOfNotWord} {
		if ids[id] {
			t.Errorf("%s が消えていない", name)
		}
	}

	// 語も除外語も無ければ（非nil空）全件そのまま
	all := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{}, []string{}))
	for name, id := range map[string]string{"残る本文": keep, "本体に除外語": bodyHasNotWord, "Lantana": mood, "ReKyou": rekyouOfNotWord} {
		if !all[id] {
			t.Errorf("語なしの検索で %s が出ていない", name)
		}
	}
}

// 空文字・空白だけの語はサーバ入口で捨てられ、素通しになる。
// 以前は words:[""] が全件、not_words:[""] が0件になっていた。
func TestHandleGetKyous_WordFilterBlankWordsAreDropped(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	kmemo := addTestKmemoWithID(t, tsURL, sessionID, GenerateNewID(), "なにか")
	mood := addTestLantana(t, tsURL, sessionID, 5)

	for _, c := range []struct {
		name     string
		words    []string
		notWords []string
	}{
		{name: "空文字の肯定語", words: []string{""}, notWords: []string{}},
		{name: "空白だけの肯定語", words: []string{" ", "　"}, notWords: []string{}},
		{name: "空文字の除外語", words: []string{}, notWords: []string{""}},
		{name: "空白だけの除外語", words: []string{}, notWords: []string{" "}},
	} {
		t.Run(c.name, func(t *testing.T) {
			ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery(c.words, c.notWords))
			if !ids[kmemo] || !ids[mood] {
				t.Errorf("空語は捨てて素通しになるべき: kmemo=%v lantana=%v", ids[kmemo], ids[mood])
			}
		})
	}

	// 前後の空白は落として照合する
	ids := searchWordIDs(t, tsURL, sessionID, wordFilterQuery([]string{" なにか "}, []string{}))
	if !ids[kmemo] || ids[mood] {
		t.Errorf("前後の空白を落とした語で照合すべき: kmemo=%v lantana=%v", ids[kmemo], ids[mood])
	}
}
