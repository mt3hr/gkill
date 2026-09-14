package gkill_server_api

// ルート表（gkill_server_api_address.go の apiRoutes）が唯一の正本であることを守るテスト群。
//
// 表・ハンドラ・doc コメント・認証区分のどれか1つだけを直すと、コンパイルは通るのに
// 「ハンドラはあるのに 404」「認証区分が黙って変わる」という形で壊れる。
// どれも目の前ではエラーにならないので、ここで全数を機械的に突き合わせる（ADR-0709）。

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// handlerFuncName は表に載せたメソッド値（g.HandleXxx）から "HandleXxx" を取り出す。
// メソッド値の関数名は "...(*GkillServerAPI).HandleXxx-fm" の形になる。
func handlerFuncName(h http.HandlerFunc) string {
	if h == nil {
		return ""
	}
	name := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	name = strings.TrimSuffix(name, "-fm")
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	return name
}

// declaredHandlerNames は *GkillServerAPI の HandleXxx(w, r) 形のメソッド名を反射で列挙する。
func declaredHandlerNames(t *testing.T) map[string]bool {
	t.Helper()
	want := reflect.TypeOf(func(http.ResponseWriter, *http.Request) {})
	typ := reflect.TypeOf(&GkillServerAPI{})
	names := map[string]bool{}
	for i := range typ.NumMethod() {
		m := typ.Method(i)
		if !strings.HasPrefix(m.Name, "Handle") {
			continue
		}
		// m.Type はレシーバを第1引数に含む
		if m.Type.NumIn() != 3 || m.Type.NumOut() != 0 ||
			m.Type.In(1) != want.In(0) || m.Type.In(2) != want.In(1) {
			continue
		}
		names[m.Name] = true
	}
	if len(names) < 80 {
		t.Fatalf("HandleXxx の列挙が想定外に少ない: %d", len(names))
	}
	return names
}

// TestAPIRoutes_Validate は起動時検査（validateAPIRoutes）が表の壊れ方を拒否することを固定する。
func TestAPIRoutes_Validate(t *testing.T) {
	g := &GkillServerAPI{}
	if err := validateAPIRoutes(g.apiRoutes()); err != nil {
		t.Fatalf("現在の表が検査を通らない: %v", err)
	}

	ok := apiRoute{Path: "/api/x", Method: "POST", Auth: authSessionRepos, Body: bodyNone, Handler: g.HandleGetKyous}
	cases := []struct {
		name   string
		routes []apiRoute
	}{
		{"重複", []apiRoute{ok, ok}},
		{"/api/ 以外", []apiRoute{{Path: "/files/x", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleGetKyous}}},
		{"/api/ だけ", []apiRoute{{Path: "/api/", Method: "POST", Auth: authNone, Body: bodyAuth, Handler: g.HandleGetKyous}}},
		{"不明なメソッド", []apiRoute{{Path: "/api/x", Method: "PUT", Auth: authSession, Body: bodyNone, Handler: g.HandleGetKyous}}},
		{"ハンドラ無し", []apiRoute{{Path: "/api/x", Method: "POST", Auth: authSession, Body: bodyNone, Handler: nil}}},
		{"無認証 POST に上限なし", []apiRoute{{Path: "/api/x", Method: "POST", Auth: authNone, Body: bodyNone, Handler: g.HandleGetKyous}}},
		{"認証つきに上限指定", []apiRoute{{Path: "/api/x", Method: "POST", Auth: authSession, Body: bodyAuth, Handler: g.HandleGetKyous}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := validateAPIRoutes(c.routes); err == nil {
				t.Errorf("検査を通ってしまう: %+v", c.routes)
			}
		})
	}
}

// TestAPIRoutes_EveryHandlerIsRouted は HandleXxx と表の行を双方向に突き合わせる。
//
// **落ちたら、ハンドラを書いたのに表へ足していない（実行時 404）か、
// 表の行を消したのにハンドラが残っている。** 表に載せないハンドラは
// PathPrefix 配信の2本だけ（免除リスト）。
func TestAPIRoutes_EveryHandlerIsRouted(t *testing.T) {
	// serve.go が PathPrefix("/files/") / PathPrefix("/zip_cache/") で直接登録する配信系。
	// 完全一致のパスを持たないので表には載らない。
	notInTable := map[string]bool{
		"HandleFileServe":         true,
		"HandleZipCacheFileServe": true,
	}

	g := &GkillServerAPI{}
	routed := map[string]string{}
	for _, rt := range g.apiRoutes() {
		name := handlerFuncName(rt.Handler)
		if name == "" {
			t.Errorf("%s %s: ハンドラの名前が取れない", rt.Method, rt.Path)
			continue
		}
		if prev, dup := routed[name]; dup {
			t.Errorf("%s が %s と %s の2行に載っている", name, prev, rt.Path)
		}
		routed[name] = rt.Path
	}

	declared := declaredHandlerNames(t)
	for name := range declared {
		if notInTable[name] {
			continue
		}
		if _, ok := routed[name]; !ok {
			t.Errorf("%s が apiRoutes の表に無い。gkill_server_api_address.go へ1行足すこと（足さないと実行時 404）", name)
		}
	}
	for name, path := range routed {
		if !declared[name] {
			t.Errorf("表の %s が指す %s は HandleXxx(w, r) の形のメソッドではない", path, name)
		}
		if notInTable[name] {
			t.Errorf("%s は PathPrefix 配信なので表に載せない", name)
		}
	}
	for name := range notInTable {
		if !declared[name] {
			t.Errorf("免除リストの %s が存在しない。リストから消すこと", name)
		}
	}
}

// TestAPIRoutes_DocCommentMatchesTable は各 handle_*.go の doc コメント
// 「// POST /api/xxx（wrapAuthRepos）」が表の Path / Method / 認証区分と一致することを固定する。
//
// doc コメントは資料（verify_docs の網羅率検査）が頼りにしている唯一の
// 「ハンドラ → パス・認証区分」の記述で、表とずれると読む人だけが騙される。
func TestAPIRoutes_DocCommentMatchesTable(t *testing.T) {
	wrapLabel := map[apiAuthKind]string{
		authNone:         "wrapNoAuth",
		authSession:      "wrapAuth",
		authSessionRepos: "wrapAuthRepos",
	}

	g := &GkillServerAPI{}
	byHandler := map[string]apiRoute{}
	for _, rt := range g.apiRoutes() {
		byHandler[handlerFuncName(rt.Handler)] = rt
	}

	docRe := regexp.MustCompile(`^// (POST|GET) (/\S+)（(wrapNoAuth|wrapAuth|wrapAuthRepos)）$`)
	funcRe := regexp.MustCompile(`^func \(g \*GkillServerAPI\) (Handle\w+)\(`)

	files, err := filepath.Glob("handle_*.go")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(string(src), "\n")
		for i, line := range lines {
			m := docRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
			if m == nil {
				continue
			}
			// doc コメントの直後（数行以内）にある func 宣言を対にする
			handler := ""
			for j := i + 1; j < len(lines) && j <= i+40; j++ {
				if fm := funcRe.FindStringSubmatch(lines[j]); fm != nil {
					handler = fm[1]
					break
				}
			}
			if handler == "" {
				t.Errorf("%s:%d: doc コメントの後ろに HandleXxx の宣言が見つからない", file, i+1)
				continue
			}
			rt, ok := byHandler[handler]
			if !ok {
				t.Errorf("%s: %s が表に無い", file, handler)
				continue
			}
			checked++
			if m[1] != rt.Method || m[2] != rt.Path || m[3] != wrapLabel[rt.Auth] {
				t.Errorf("%s: doc コメント「%s %s（%s）」が表「%s %s（%s）」と食い違う",
					file, m[1], m[2], m[3], rt.Method, rt.Path, wrapLabel[rt.Auth])
			}
		}
	}
	if checked != len(byHandler) {
		t.Errorf("doc コメントを突き合わせたハンドラ %d 本 ≠ 表の行 %d 本。書式「// POST /api/xxx（wrapXxx）」から外れた doc コメントがある", checked, len(byHandler))
	}
}

// TestAPIRoutes_AuthKindGolden は全ルートの認証区分とボディ上限を名指しで固定する。
//
// 認証区分の変更は「表と、ここの golden の2箇所を意図して直す」作業にする。
// 表だけ直して通ると、wrapAuthRepos → wrapNoAuth のような退行が
// テストを1本も落とさずに本番へ届く。
func TestAPIRoutes_AuthKindGolden(t *testing.T) {
	type kind struct {
		auth apiAuthKind
		body apiBodyCap
	}
	golden := map[string]kind{
		"/api/login":                             {authNone, bodyAuth},
		"/api/logout":                            {authNone, bodyAuth},
		"/api/reset_password":                    {authNone, bodyAuth},
		"/api/set_new_password":                  {authNone, bodyAuth},
		"/api/get_shared_kyous":                  {authNone, bodyAuth},
		"/api/urlog_bookmarklet":                 {authNone, bodyAuth},
		"/api/urlog_bookmarklet_page":            {authNone, bodyNone},
		"/api/get_kyous_mcp":                     {authNone, bodyAuth},
		"/api/get_rep_infos_mcp":                 {authNone, bodyAuth},
		"/api/upload_files":                      {authNone, bodyUpload},
		"/api/upload_gpslog_files":               {authNone, bodyUpload},
		"/api/browse_zip_contents":               {authNone, bodyAuth},
		"/api/get_idf_kyou_by_relative_path":     {authNone, bodyAuth},
		"/api/get_application_config":            {authSession, bodyNone},
		"/api/get_server_configs":                {authSession, bodyNone},
		"/api/update_application_config":         {authSession, bodyNone},
		"/api/update_account_status":             {authSession, bodyNone},
		"/api/update_user_reps":                  {authSession, bodyNone},
		"/api/update_server_configs":             {authSession, bodyNone},
		"/api/add_user":                          {authSession, bodyNone},
		"/api/generate_tls_file":                 {authSession, bodyNone},
		"/api/get_gkill_notification_public_key": {authSession, bodyNone},
		"/api/register_gkill_notification":       {authSession, bodyNone},
		"/api/open_directory":                    {authSession, bodyNone},
		"/api/open_file":                         {authSession, bodyNone},
		"/api/reload_repositories":               {authSession, bodyNone},
		"/api/get_updated_datas_by_time":         {authSession, bodyNone},
		"/api/update_cache":                      {authSession, bodyNone},
		"/api/get_plugin_list":                   {authSession, bodyNone},
		"/api/get_plugin_content_html":           {authSession, bodyNone},
		"/api/get_plugin_config_html":            {authSession, bodyNone},
		"/api/post_plugin_config":                {authSession, bodyNone},
		"/api/add_tag":                           {authSessionRepos, bodyNone},
		"/api/add_text":                          {authSessionRepos, bodyNone},
		"/api/add_gkill_notification":            {authSessionRepos, bodyNone},
		"/api/add_kmemo":                         {authSessionRepos, bodyNone},
		"/api/add_kc":                            {authSessionRepos, bodyNone},
		"/api/add_urlog":                         {authSessionRepos, bodyNone},
		"/api/add_nlog":                          {authSessionRepos, bodyNone},
		"/api/add_timeis":                        {authSessionRepos, bodyNone},
		"/api/add_mi":                            {authSessionRepos, bodyNone},
		"/api/add_lantana":                       {authSessionRepos, bodyNone},
		"/api/add_rekyou":                        {authSessionRepos, bodyNone},
		"/api/add_mirekyou":                      {authSessionRepos, bodyNone},
		"/api/update_tag":                        {authSessionRepos, bodyNone},
		"/api/update_text":                       {authSessionRepos, bodyNone},
		"/api/update_gkill_notification":         {authSessionRepos, bodyNone},
		"/api/update_kmemo":                      {authSessionRepos, bodyNone},
		"/api/update_kc":                         {authSessionRepos, bodyNone},
		"/api/update_urlog":                      {authSessionRepos, bodyNone},
		"/api/update_nlog":                       {authSessionRepos, bodyNone},
		"/api/update_timeis":                     {authSessionRepos, bodyNone},
		"/api/update_lantana":                    {authSessionRepos, bodyNone},
		"/api/update_idf_kyou":                   {authSessionRepos, bodyNone},
		"/api/update_mi":                         {authSessionRepos, bodyNone},
		"/api/update_rekyou":                     {authSessionRepos, bodyNone},
		"/api/update_mirekyou":                   {authSessionRepos, bodyNone},
		"/api/get_kyous":                         {authSessionRepos, bodyNone},
		"/api/get_kyou":                          {authSessionRepos, bodyNone},
		"/api/get_kmemo":                         {authSessionRepos, bodyNone},
		"/api/get_kc":                            {authSessionRepos, bodyNone},
		"/api/get_urlog":                         {authSessionRepos, bodyNone},
		"/api/get_nlog":                          {authSessionRepos, bodyNone},
		"/api/get_timeis":                        {authSessionRepos, bodyNone},
		"/api/get_mi":                            {authSessionRepos, bodyNone},
		"/api/get_lantana":                       {authSessionRepos, bodyNone},
		"/api/get_rekyou":                        {authSessionRepos, bodyNone},
		"/api/get_mirekyou":                      {authSessionRepos, bodyNone},
		"/api/get_rekyous_by_target_id":          {authSessionRepos, bodyNone},
		"/api/get_mirekyous_by_target_id":        {authSessionRepos, bodyNone},
		"/api/get_git_commit_log":                {authSessionRepos, bodyNone},
		"/api/get_idf_kyou":                      {authSessionRepos, bodyNone},
		"/api/get_mi_board_list":                 {authSessionRepos, bodyNone},
		"/api/get_all_tag_names":                 {authSessionRepos, bodyNone},
		"/api/get_all_rep_names":                 {authSessionRepos, bodyNone},
		"/api/get_tags_by_id":                    {authSessionRepos, bodyNone},
		"/api/get_tag_histories_by_tag_id":       {authSessionRepos, bodyNone},
		"/api/get_texts_by_id":                   {authSessionRepos, bodyNone},
		"/api/get_gkill_notifications_by_id":     {authSessionRepos, bodyNone},
		"/api/get_text_histories_by_text_id":     {authSessionRepos, bodyNone},
		"/api/get_gkill_notification_histories_by_notification_id": {authSessionRepos, bodyNone},
		"/api/get_gps_log":                  {authSessionRepos, bodyNone},
		"/api/add_share_kyou_list_info":     {authSessionRepos, bodyNone},
		"/api/update_share_kyou_list_info":  {authSessionRepos, bodyNone},
		"/api/get_share_kyou_list_infos":    {authSessionRepos, bodyNone},
		"/api/delete_share_kyou_list_infos": {authSessionRepos, bodyNone},
		"/api/get_repositories":             {authSessionRepos, bodyNone},
		"/api/commit_tx":                    {authSessionRepos, bodyNone},
		"/api/discard_tx":                   {authSessionRepos, bodyNone},
		"/api/submit_kftl_text":             {authSessionRepos, bodyNone}}

	g := &GkillServerAPI{}
	seen := map[string]bool{}
	for _, rt := range g.apiRoutes() {
		seen[rt.Path] = true
		want, ok := golden[rt.Path]
		if !ok {
			t.Errorf("%s が golden に無い。認証区分を決めてここへ1行足すこと", rt.Path)
			continue
		}
		if rt.Auth != want.auth || rt.Body != want.body {
			t.Errorf("%s: 表は (auth=%d, body=%d)、golden は (auth=%d, body=%d)。意図した変更なら両方を直すこと",
				rt.Path, rt.Auth, rt.Body, want.auth, want.body)
		}
	}
	for path := range golden {
		if !seen[path] {
			t.Errorf("golden の %s が表に無い。消したなら golden からも消すこと", path)
		}
	}
}
