package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

func TestAutoTagCmdNotNil(t *testing.T) {
	if AutoTagCmd == nil {
		t.Fatal("AutoTagCmd should not be nil")
	}
	if AutoTagCmd.Use != "auto_tag" {
		t.Errorf("AutoTagCmd.Use = %q, want %q", AutoTagCmd.Use, "auto_tag")
	}
	for _, flagName := range []string{"tag_by_rep_prefix", "tag_by_rep_name", "dry_run"} {
		if AutoTagCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("AutoTagCmd should have --%s", flagName)
		}
	}
}

func TestParseAutoTagRules(t *testing.T) {
	prefixRules, repTypeRules, err := parseAutoTagRules(
		[]string{"AutoScreenshot_=autolog_screenshot", "Box_=box"},
		[]string{"git_commit_log"},
	)
	if err != nil {
		t.Fatalf("parseAutoTagRules: %v", err)
	}
	if len(prefixRules) != 2 {
		t.Fatalf("prefixRules length: got %d, want 2", len(prefixRules))
	}
	if prefixRules[0].Prefix != "AutoScreenshot_" || prefixRules[0].Tag != "autolog_screenshot" {
		t.Errorf("prefixRules[0]: got %#v", prefixRules[0])
	}
	if len(repTypeRules) != 1 || repTypeRules[0] != "git_commit_log" {
		t.Errorf("repTypeRules: got %#v", repTypeRules)
	}
}

func TestParseAutoTagRulesRejectsBrokenPrefixRule(t *testing.T) {
	for _, arg := range []string{"AutoScreenshot_", "=autolog_screenshot", "AutoScreenshot_="} {
		if _, _, err := parseAutoTagRules([]string{arg}, nil); err == nil {
			t.Errorf("parseAutoTagRules(%q) should fail", arg)
		}
	}
	if _, _, err := parseAutoTagRules(nil, []string{""}); err == nil {
		t.Error("empty rep type should fail")
	}
}

func TestAddAutoTagTargetDoesNotDuplicateSameTag(t *testing.T) {
	targets := map[string]*autoTagTarget{}
	kyou := reps.Kyou{ID: "kyou1", RepName: "gkill"}

	addAutoTagTarget(targets, kyou, "gkill")
	addAutoTagTarget(targets, kyou, "gkill")
	addAutoTagTarget(targets, kyou, "autolog_screenshot")

	if len(targets) != 1 {
		t.Fatalf("targets length: got %d, want 1", len(targets))
	}
	if got := targets["kyou1"].Tags; len(got) != 2 {
		t.Errorf("tags: got %#v, want 2 entries", got)
	}
}

func TestAutoTagIDIsStableForSameTargetAndTag(t *testing.T) {
	// IDが変わると過去に付与したぶんと食い違い、全件が付け直しになる。
	// 冪等性はこのIDとサーバ側のAlreadyExistTagErrorだけで担保している
	first := autoTagID("kyou1", "gkill")
	if first != autoTagID("kyou1", "gkill") {
		t.Error("tag id should be stable")
	}
	if first == autoTagID("kyou1", "gkill_autolog") {
		t.Error("different tag names should get different ids")
	}
	if first == autoTagID("kyou2", "gkill") {
		t.Error("different targets should get different ids")
	}
	if _, err := uuid.Parse(first); err != nil {
		t.Errorf("tag id should be a uuid: %v", err)
	}
	// 区切りが無いと "ab"+"c" と "a"+"bc" が同じIDになってしまう
	if autoTagID("ab", "c") == autoTagID("a", "bc") {
		t.Error("tag id should not collide across the target/tag boundary")
	}
}

func TestAutoTagIDMatchesPreviouslyIssuedID(t *testing.T) {
	// 独立バイナリだった頃に付与したタグと同じIDになること。
	// 名前空間の文字列を変えると全件が付け直しになるので、値で固定しておく
	want := uuid.NewSHA1(
		uuid.NewSHA1(uuid.NameSpaceOID, []byte("github.com/mt3hr/gkill/gkill_auto_tag")),
		[]byte("kyou1\x00gkill"),
	).String()
	if got := autoTagID("kyou1", "gkill"); got != want {
		t.Errorf("autoTagID = %q, want %q", got, want)
	}
}

func TestShouldRefreshAutoTagSession(t *testing.T) {
	// 500件ごと(進捗印字と同じ区切り)でだけ true。0件では延長しない。
	cases := []struct {
		added int
		want  bool
	}{
		{0, false},
		{1, false},
		{499, false},
		{500, true},
		{501, false},
		{999, false},
		{1000, true},
		{1500, true},
	}
	for _, c := range cases {
		if got := shouldRefreshAutoTagSession(c.added); got != c.want {
			t.Errorf("shouldRefreshAutoTagSession(%d) = %v, want %v", c.added, got, c.want)
		}
	}
}

func TestFindTaggedKyouIDsQueryUsesTagsAnd(t *testing.T) {
	// 単一タグの「付いているものだけ」をANDで表現していることを固定する
	// (現在のfind_filterはOR/ANDとも完全一致照合なので結果は同じだが、意図の直接表現)
	base := &find.FindQuery{RepTypes: []string{"git_commit_log"}}
	client := &autoTagAPIClient{}
	query := client.buildTaggedQuery(base, "gkill")

	if query.Tags == nil {
		t.Error("Tags should be non-nil (tag filter enabled)")
	}
	if !query.TagsAnd {
		t.Error("TagsAnd should be true")
	}
	if len(query.Tags) != 1 || query.Tags[0] != "gkill" {
		t.Errorf("Tags: got %#v, want [gkill]", query.Tags)
	}
	if query.RepTypes == nil || len(query.RepTypes) != 1 || query.RepTypes[0] != "git_commit_log" {
		t.Errorf("base query should be kept: got %#v", query)
	}
	// 呼び出し元のクエリを書き換えてはいけない（rep名ごとに使い回すため）
	if base.Tags != nil {
		t.Error("base query should not be modified")
	}
}

// newAutoTagTestClient は httptest.Server を宛先にした autoTagAPIClient を作る。
// ResolveLocalServerEndpoint は設定DB(server_config.db)が要るので、テストでは宛先を直接組み立てる。
func newAutoTagTestClient(server *httptest.Server) *autoTagAPIClient {
	return &autoTagAPIClient{
		Endpoint: &LocalServerEndpoint{
			BaseURL: server.URL,
			Device:  "test_device",
			Client:  server.Client(),
		},
		SessionID: "test_session",
	}
}

// post は応答本文の読み取りに失敗したら、ステータス付きの「読み取り失敗」エラーを返す。
//
// サーバがContent-Lengthぶんの本文を送りきらずに接続を切ると、
// クライアント側の io.ReadAll が途中で unexpected EOF になる(実装が実際に踏む形)。
// デコード失敗とは別のエラー文で、どの段階で壊れたかが分かることを固定する。
func TestAutoTagPost_ReadBodyFailureReturnsReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 宣言した長さより短い本文を書いてハンドラを終える。
		// net/httpサーバは書き足りないまま接続を閉じるので、クライアントは読み取り途中で失敗する
		w.Header().Set("Content-Length", "4096")
		_, _ = w.Write([]byte(`{"messages":null,"errors":null`))
	}))
	defer server.Close()

	client := newAutoTagTestClient(server)
	response := &req_res.GetAllRepNamesResponse{}
	err := client.post(context.Background(), "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{SessionID: client.SessionID, LocaleName: autoTagLocaleName}, response)
	if err == nil {
		t.Fatal("本文の読み取り失敗はエラーになるべき")
	}
	if !strings.Contains(err.Error(), "error at read response of") {
		t.Errorf("読み取り失敗のエラー文ではない: %v", err)
	}
	if !strings.Contains(err.Error(), "status = 200") {
		t.Errorf("エラー文にステータスが入っていない: %v", err)
	}
}

// 8MB(maxResponseBodyBytes)を超える本文は上限で切り詰められ、デコード失敗として現れる。
//
// io.LimitReader は上限超過をエラーにせず黙って打ち切るので、
// 「本文が大きすぎる」はこの実装では読み取り失敗ではなく、
// 途中で切れたJSONのデコード失敗になる。巨大応答でも読むのは上限まで、
// エラー文へ入る断片は1024バイトまでで、どちらも際限なく膨らまないことを固定する。
func TestAutoTagPost_HugeBodyIsCappedAndFailsDecode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"messages":null,"errors":null,"rep_names":["`))
		_, _ = w.Write(bytes.Repeat([]byte("a"), maxResponseBodyBytes))
		_, _ = w.Write([]byte(`"]}`))
	}))
	defer server.Close()

	client := newAutoTagTestClient(server)
	response := &req_res.GetAllRepNamesResponse{}
	err := client.post(context.Background(), "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{SessionID: client.SessionID, LocaleName: autoTagLocaleName}, response)
	if err == nil {
		t.Fatal("上限を超えて切り詰められた本文はエラーになるべき")
	}
	if !strings.Contains(err.Error(), "error at decode response of") {
		t.Errorf("デコード失敗のエラー文ではない: %v", err)
	}
	if len(err.Error()) > 4096 {
		t.Errorf("エラー文が長すぎる(断片が1024バイトで切られていない): %d bytes", len(err.Error()))
	}
}

// 本文がJSONですらないとき(プロキシのHTMLエラーページ、TLSサーバへ平文で繋いだ等)は、
// ステータスと本文の断片を添えたエラーになる。断片は1024バイトで打ち切る。
func TestAutoTagPost_NonJSONBodyReturnsSnippetError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		// 1024バイト目より後ろにマーカーを置き、断片へ入らないことを確かめる
		_, _ = w.Write([]byte("<html><body>Bad Gateway " + strings.Repeat("x", 1024) + "TAIL_MARKER_BEYOND_CAP</body></html>"))
	}))
	defer server.Close()

	client := newAutoTagTestClient(server)
	response := &req_res.GetAllRepNamesResponse{}
	err := client.post(context.Background(), "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{SessionID: client.SessionID, LocaleName: autoTagLocaleName}, response)
	if err == nil {
		t.Fatal("非JSON本文はエラーになるべき")
	}
	if !strings.Contains(err.Error(), "error at decode response of") {
		t.Errorf("デコード失敗のエラー文ではない: %v", err)
	}
	if !strings.Contains(err.Error(), "status = 502") {
		t.Errorf("エラー文にステータスが入っていない: %v", err)
	}
	if !strings.Contains(err.Error(), "Bad Gateway") {
		t.Errorf("エラー文に本文の断片が入っていない: %v", err)
	}
	if strings.Contains(err.Error(), "TAIL_MARKER_BEYOND_CAP") {
		t.Errorf("断片が1024バイトで切られていない: %v", err)
	}
}

// HTTP 200でも本文のerrorsに中身があれば失敗として扱う。
//
// gkillは2026-08より前は異常時も常に200を返していたし、今もエラーの中身は
// 本文のerrors配列が正なので、ステータスだけを見て成功と判定してはいけない。
func TestAutoTagGetAllRepNames_ErrorsWithHTTP200IsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"messages":null,"errors":[{"error_code":%q,"error_message":"管理者権限がありません"}],"rep_names":null}`, message.AccountNotHasAdminError)
	}))
	defer server.Close()

	client := newAutoTagTestClient(server)
	repNames, err := client.GetAllRepNames(context.Background())
	if err == nil {
		t.Fatalf("200 + errorsあり は失敗になるべき: rep names = %#v", repNames)
	}
	if !strings.Contains(err.Error(), message.AccountNotHasAdminError) {
		t.Errorf("エラー文にerror_codeが入っていない: %v", err)
	}
	if !strings.Contains(err.Error(), "管理者権限がありません") {
		t.Errorf("エラー文にerror_messageが入っていない: %v", err)
	}
}

// 非2xxでも本文のerrorsがデコードできるなら、error_code/error_messageを伝える。
//
// gkillは2026-08から異常時に4xx/5xxを返すが、エラーの中身は今までどおり
// 本文のerrors配列にしか入っていない。ステータスで打ち切ると「HTTP 401」しか
// 分からず、セッション切れなのか権限不足なのか判別できなくなる
// (本文のerrorsを優先する判断はMCPのgkill-client.mjsと同じ)。
func TestAutoTagGetAllRepNames_Non2xxCarriesErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"messages":null,"errors":[{"error_code":%q,"error_message":"セッションが見つかりませんでした"}],"rep_names":null}`, message.AccountSessionNotFoundError)
	}))
	defer server.Close()

	client := newAutoTagTestClient(server)
	_, err := client.GetAllRepNames(context.Background())
	if err == nil {
		t.Fatal("401 + errorsあり は失敗になるべき")
	}
	if !strings.Contains(err.Error(), message.AccountSessionNotFoundError) {
		t.Errorf("エラー文にerror_codeが入っていない: %v", err)
	}
	if !strings.Contains(err.Error(), "セッションが見つかりませんでした") {
		t.Errorf("エラー文にerror_messageが伝わっていない: %v", err)
	}
}

// 既存IDのタグはHTTP 409 + ERR000056で届き、AddTagは「既に付いている」(スキップ)として飲む。
//
// auto_tagの冪等性はこの経路が要で、「付いているか」の判定を取りこぼしても
// サーバが同じIDを弾いて二重登録にならず、手で消したタグも同じIDで弾かれて復活しない。
// 2026-08からERR000056はHTTP 409で届くようになったため、ステータスで
// 打ち切るとこのスキップに到達できず、冪等なはずの再実行が失敗になってしまう。
func TestAutoTagAddTag_AlreadyExistOver409IsSkip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, `{"messages":null,"errors":[{"error_code":%q,"error_message":"すでに存在するタグです"}],"added_tag":null}`, message.AlreadyExistTagError)
	}))
	defer server.Close()

	client := newAutoTagTestClient(server)
	alreadyExist, err := client.AddTag(context.Background(), reps.Tag{
		ID:       autoTagID("kyou1", "gkill"),
		TargetID: "kyou1",
		Tag:      "gkill",
	})
	if err != nil {
		t.Fatalf("409 + ERR000056 はスキップ扱いのはず: %v", err)
	}
	if !alreadyExist {
		t.Error("alreadyExist = false, want true")
	}
}

// 非2xxで本文にエラーの中身が無いなら、ステータスを唯一の手掛かりとしてエラーにする。
//
// ここでnilを返すと「4xxなのに成功・0件」になり、静かに壊れる。
// errors:[null](中身なしの要素だけ)も同じ扱い。
func TestAutoTagPost_Non2xxWithoutErrorContentFailsWithStatus(t *testing.T) {
	for _, body := range []string{
		`{"messages":null,"errors":null,"rep_names":null}`,
		`{"messages":null,"errors":[null],"rep_names":null}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(body))
		}))

		client := newAutoTagTestClient(server)
		response := &req_res.GetAllRepNamesResponse{}
		err := client.post(context.Background(), "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{SessionID: client.SessionID, LocaleName: autoTagLocaleName}, response)
		server.Close()

		if err == nil {
			t.Errorf("body %q: 500でerrorsに中身が無くてもエラーになるべき", body)
			continue
		}
		if !strings.Contains(err.Error(), "error at post") || !strings.Contains(err.Error(), "status = 500") {
			t.Errorf("body %q: ステータスを手掛かりにしたエラー文ではない: %v", body, err)
		}
	}
}

// 200 + errors:null が今までどおり成功として通ること(退行防止)。
// あわせて、リクエストが指定パスへJSONで届いていることも確かめる。
func TestAutoTagGetAllRepNames_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/get_all_rep_names" {
			t.Errorf("path = %q, want /api/get_all_rep_names", r.URL.Path)
		}
		request := &req_res.GetAllRepNamesRequest{}
		if err := json.NewDecoder(r.Body).Decode(request); err != nil {
			t.Errorf("リクエスト本文がJSONとして読めない: %v", err)
		} else if request.SessionID != "test_session" {
			t.Errorf("session id = %q, want test_session", request.SessionID)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messages":null,"errors":null,"rep_names":["gkill","gkill_autolog"]}`))
	}))
	defer server.Close()

	client := newAutoTagTestClient(server)
	repNames, err := client.GetAllRepNames(context.Background())
	if err != nil {
		t.Fatalf("GetAllRepNames: %v", err)
	}
	if len(repNames) != 2 || repNames[0] != "gkill" || repNames[1] != "gkill_autolog" {
		t.Errorf("rep names: got %#v, want [gkill gkill_autolog]", repNames)
	}
}
