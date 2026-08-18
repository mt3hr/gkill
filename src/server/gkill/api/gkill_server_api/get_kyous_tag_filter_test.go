package gkill_server_api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// addTestTagTo は既存のKyouへタグを1つ付けます。
func addTestTagTo(t *testing.T, tsURL string, sessionID string, targetID string, tagName string) {
	t.Helper()
	now := time.Now().Truncate(time.Second)
	addTagReq := &req_res.AddTagRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		Tag: reps.Tag{
			ID:          GenerateNewID(),
			TargetID:    targetID,
			Tag:         tagName,
			RelatedTime: now,
			CreateTime:  now,
			CreateApp:   "test",
			CreateUser:  "admin",
			UpdateTime:  now,
			UpdateApp:   "test",
			UpdateUser:  "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_tag", addTagReq)
	defer resp.Body.Close()

	var addResp req_res.AddTagResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add tag response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add tag errors: %+v", addResp.Errors)
	}
}

// タグ絞り込みの回帰テスト。
//
// getAllTags(全repの全タグ走査でRelatedTagIDs=「タグが1つでも付いているIDの集合」を作る)は
// **「タグ無し」仮想タグ(api.NoTags)を使う検索のときだけ**走らせている。
// RelatedTagIDs の読み手が NoTags 分岐しか無いので、それ以外では結果に影響しないため。
//
// 起動条件を間違えると壊れ方が静かなので、両方向を固定する。
//   - NoTags を使う検索で走らせ忘れる → RelatedTagIDs が空 = 全件が「タグなし」扱いになり、
//     タグの付いた記録まで返る
//   - NoTags を使わない検索で結果が変わる → タグ絞り込みそのものが壊れる
func TestHandleGetKyous_TagFilterAndNoTags(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	taggedID := addTestKmemo(t, tsURL, sessionID, "タグの付いたメモ")
	untaggedID := addTestKmemo(t, tsURL, sessionID, "タグの無いメモ")
	addTestTagTo(t, tsURL, sessionID, taggedID, "重要")

	collectIDs := func(res req_res.GetKyousResponse) map[string]bool {
		ids := map[string]bool{}
		for _, kyou := range res.Kyous {
			ids[kyou.ID] = true
		}
		return ids
	}

	t.Run("タグ名で絞ると、そのタグの記録だけが返る", func(t *testing.T) {
		query := &find.FindQuery{Tags: []string{"重要"}}
		ids := collectIDs(getKyousWithQuery(t, tsURL, sessionID, query))
		if !ids[taggedID] {
			t.Error("タグの付いた記録が返っていない")
		}
		if ids[untaggedID] {
			t.Error("タグの無い記録まで返っている")
		}
	})

	t.Run("タグ無しで絞ると、タグの付いていない記録だけが返る", func(t *testing.T) {
		// ここで getAllTags を走らせ損ねると RelatedTagIDs が空になり、
		// タグの付いた記録まで「タグなし」と判定されて返ってしまう
		query := &find.FindQuery{Tags: []string{"no tags"}}
		ids := collectIDs(getKyousWithQuery(t, tsURL, sessionID, query))
		if !ids[untaggedID] {
			t.Error("タグの無い記録が返っていない")
		}
		if ids[taggedID] {
			t.Error("タグの付いた記録まで「タグなし」として返っている")
		}
	})

	t.Run("タグ名とタグ無しの併記はどちらも返る(OR)", func(t *testing.T) {
		query := &find.FindQuery{Tags: []string{"重要", "no tags"}}
		ids := collectIDs(getKyousWithQuery(t, tsURL, sessionID, query))
		if !ids[taggedID] || !ids[untaggedID] {
			t.Errorf("ORの和になっていない tagged=%v untagged=%v", ids[taggedID], ids[untaggedID])
		}
	})

	t.Run("タグ条件が空配列なら0件", func(t *testing.T) {
		// 非nullの空配列は「フィルタ有効かつ0件指定」の意味
		query := &find.FindQuery{Tags: []string{}}
		res := getKyousWithQuery(t, tsURL, sessionID, query)
		if len(res.Kyous) != 0 {
			t.Errorf("空のタグ条件で%d件返っている", len(res.Kyous))
		}
	})
}
