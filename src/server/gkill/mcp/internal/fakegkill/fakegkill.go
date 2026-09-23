// Package fakegkill は MCP のゴールデン採取・再生に使う偽の gkill サーバ。
//
// 固定応答を返し、受け取った要求（パス・クエリ・生の本文）を順番に記録する。
// Node 実装からの採取（fakegkillcmd で別プロセス起動）と Go 実装の再生テスト
// （httptest で同居）が同じ実体を使うので、両者の入力は同じになる。
//
// 応答の規則はすべてこのファイルにある。状態を持つのは「セッション切れを1回だけ返す」
// トリガと「同じ冪等キーの再送」だけで、Reset で消える。
package fakegkill

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// SessionID はログインが返すセッション ID。
const SessionID = "sess-login"

// Record は受け取った要求1件。
type Record struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query,omitempty"`
	Cookie string `json:"cookie,omitempty"`
	Body   string `json:"body,omitempty"`
}

// Server は偽 gkill。http.Handler として使う。
type Server struct {
	mu          sync.Mutex
	records     []Record
	expiredOnce map[string]bool
}

// New は空の状態の偽 gkill を作る。
func New() *Server {
	return &Server{expiredOnce: map[string]bool{}}
}

// Drain は記録した要求を返して空にする。
func (s *Server) Drain() []Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.records
	s.records = nil
	return out
}

// Reset は記録と状態を消す。
func (s *Server) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = nil
	s.expiredOnce = map[string]bool{}
}

func (s *Server) record(r Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, r)
}

func mustObj(text string) *jsonobj.Object {
	o, err := jsonobj.UnmarshalObject([]byte(text))
	if err != nil {
		panic("fakegkill fixture: " + err.Error())
	}
	return o
}

func mustArr(text string) []any {
	v, err := jsonobj.Unmarshal([]byte(text))
	if err != nil {
		panic("fakegkill fixture: " + err.Error())
	}
	items, ok := jsonobj.AsArray(v)
	if !ok {
		panic("fakegkill fixture: not an array")
	}
	return items
}

func writeJSON(w http.ResponseWriter, status int, body *jsonobj.Object) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, jsonobj.MarshalString(body))
}

func envelope(kv ...any) *jsonobj.Object {
	return jsonobj.Obj("messages", jsonobj.Arr(), "errors", jsonobj.Arr()).Merge(jsonobj.Obj(kv...))
}

func errorEnvelope(code, message, kind, reason string) *jsonobj.Object {
	e := jsonobj.Obj("error_code", code, "error_message", message, "error_kind", kind)
	if reason != "" {
		e.Set("reason", reason)
	}
	return jsonobj.Obj("messages", jsonobj.Arr(), "errors", jsonobj.Arr(e))
}

// ServeHTTP は経路ごとの規則で応答する。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/__control/") {
		s.serveControl(w, r)
		return
	}
	rawBody, _ := io.ReadAll(r.Body)
	s.record(Record{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.RawQuery,
		Cookie: r.Header.Get("Cookie"),
		Body:   string(rawBody),
	})
	if strings.HasPrefix(r.URL.Path, "/files/") {
		s.serveFile(w, r)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		writeJSON(w, 404, jsonobj.Obj("error", "not found"))
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, 405, jsonobj.Obj("error", "method not allowed"))
		return
	}
	body, err := jsonobj.UnmarshalObject(rawBody)
	if err != nil {
		writeJSON(w, 400, errorEnvelope("ERR000001", "invalid json", "bad_request", ""))
		return
	}
	s.serveAPI(w, r.URL.Path, body)
}

func (s *Server) serveControl(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/__control/drain":
		records := s.Drain()
		items := make([]any, 0, len(records))
		for _, rec := range records {
			o := jsonobj.Obj("method", rec.Method, "path", rec.Path)
			if rec.Query != "" {
				o.Set("query", rec.Query)
			}
			if rec.Cookie != "" {
				o.Set("cookie", rec.Cookie)
			}
			if rec.Body != "" {
				o.Set("body", rec.Body)
			}
			items = append(items, o)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, jsonobj.MarshalString(items))
	case "/__control/reset":
		s.Reset()
		writeJSON(w, 200, jsonobj.Obj("ok", true))
	default:
		writeJSON(w, 404, jsonobj.Obj("error", "unknown control"))
	}
}

// serveFile は /files/{rep}/{path} の配信。Cookie の gkill_session_id が無ければ 401。
func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	cookie := r.Header.Get("Cookie")
	sid := strings.TrimPrefix(cookie, "gkill_session_id=")
	if sid == "" || sid == "expired-file-session" {
		w.WriteHeader(401)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/files/")
	slash := strings.Index(rest, "/")
	if slash < 0 {
		w.WriteHeader(404)
		return
	}
	repName, _ := url.PathUnescape(rest[:slash])
	fileName, _ := url.PathUnescape(rest[slash+1:])
	query := r.URL.Query()
	if repName != "Files" {
		w.WriteHeader(404)
		return
	}
	var data []byte
	contentType := ""
	switch fileName {
	case "photo.png":
		if query.Get("thumb") != "" {
			data = []byte("JPEGTHUMB:" + query.Get("thumb"))
			contentType = "image/jpeg"
		} else {
			data = []byte("\x89PNG\r\n\x1a\nFAKEPNG")
			contentType = "image/png; charset=binary"
		}
	case "docs/doc.pdf":
		data = []byte("%PDF-1.4 FAKE")
		contentType = "application/pdf"
	case "notes/sub dir/memo.txt":
		data = []byte("hello メモ")
		contentType = "text/plain; charset=utf-8"
	case "movie.mp4":
		if query.Get("is_video") == "true" && query.Get("thumb") != "" {
			data = []byte("JPEGVIDEOTHUMB")
			contentType = "image/jpeg"
		} else {
			data = []byte("FAKEMP4")
			contentType = "video/mp4"
		}
	case "big.bin":
		data = bytes.Repeat([]byte{0x41}, 8*1024*1024+1)
		contentType = "application/octet-stream"
	case "no-type.dat":
		data = []byte("NOTYPE")
	default:
		w.WriteHeader(404)
		return
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(200)
	_, _ = w.Write(data)
}

func stringOf(body *jsonobj.Object, key string) string {
	if body == nil {
		return ""
	}
	v, _ := body.String(key)
	return v
}

func (s *Server) serveAPI(w http.ResponseWriter, path string, body *jsonobj.Object) {
	if path == "/api/login" {
		userID := stringOf(body, "user_id")
		if userID == "" || userID == "baduser" || stringOf(body, "password_sha256") == "" {
			writeJSON(w, 401, errorEnvelope("ERR000005", "ログインに失敗しました", "unauthorized", "invalid_credentials"))
			return
		}
		writeJSON(w, 200, envelope("session_id", SessionID))
		return
	}

	sid := stringOf(body, "session_id")
	if sid == "" {
		writeJSON(w, 401, errorEnvelope("ERR000013", "セッションがありません", "unauthorized", "session_missing"))
		return
	}
	// locale_name "expire-session" は経路ごとに1回だけセッション切れを返す（再ログインの検査）。
	if stringOf(body, "locale_name") == "expire-session" {
		s.mu.Lock()
		first := !s.expiredOnce[path]
		s.expiredOnce[path] = true
		s.mu.Unlock()
		if first {
			writeJSON(w, 401, errorEnvelope("ERR000373", "セッションの期限が切れました", "unauthorized", "session_expired"))
			return
		}
	}
	// locale_name "server-error" は HTML の 502（JSON でない本文）。
	if stringOf(body, "locale_name") == "server-error" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(502)
		_, _ = io.WriteString(w, "<html><body>Bad Gateway</body></html>")
		return
	}
	// locale_name "gkill-error" は業務エラー（envelope 付き 400）。
	if stringOf(body, "locale_name") == "gkill-error" {
		writeJSON(w, 400, errorEnvelope("ERR000200", "検索に失敗しました", "internal", "db_busy"))
		return
	}

	switch path {
	case "/api/get_kyous_mcp":
		s.serveGetKyous(w, body)
	case "/api/get_plugin_content_html":
		kyouID := stringOf(body, "kyou_id")
		html, ok := pluginContentHTML[kyouID]
		if !ok {
			writeJSON(w, 400, errorEnvelope("ERR000999", "プラグインの本文取得に失敗しました: "+kyouID, "internal", "plugin_busy"))
			return
		}
		writeJSON(w, 200, envelope("html", html))
	case "/api/get_plugin_list":
		writeJSON(w, 200, envelope("plugins", mustArr(pluginList)))
	case "/api/get_mi_board_list":
		writeJSON(w, 200, envelope("boards", mustArr(miBoards)))
	case "/api/get_all_tag_names":
		writeJSON(w, 200, envelope("tag_names", mustArr(tagNames)))
	case "/api/get_all_rep_names":
		writeJSON(w, 200, envelope("rep_names", mustArr(repNames)))
	case "/api/get_gps_log":
		if strings.HasPrefix(stringOf(body, "start_date"), "2030") {
			writeJSON(w, 200, envelope("gps_logs", jsonobj.Arr()))
			return
		}
		writeJSON(w, 200, envelope("gps_logs", mustArr(gpsLogs)))
	case "/api/get_rep_infos_mcp":
		writeJSON(w, 200, envelope().Merge(mustObj(repInfos)))
	case "/api/get_application_config":
		writeJSON(w, 200, envelope("application_config", mustObj(applicationConfig)))
	case "/api/submit_kftl_text":
		s.serveKftl(w, body)
	case "/api/get_skill_list":
		writeJSON(w, 200, envelope("skills", mustArr(skillList)))
	case "/api/get_skill":
		serveGetSkill(w, body)
	case "/api/write_skill_file":
		serveWriteSkillFile(w, body)
	case "/api/delete_skill":
		if stringOf(body, "name") != fakeSkillName {
			writeJSON(w, 404, errorEnvelope("ERR000430", "スキルが見つかりません ("+stringOf(body, "name")+")", "not_found", ""))
			return
		}
		writeJSON(w, 200, envelope())
	default:
		if strings.HasPrefix(path, "/api/get_") {
			s.serveHistory(w, path, body)
			return
		}
		if strings.HasPrefix(path, "/api/add_") {
			s.serveAdd(w, path, body)
			return
		}
		if strings.HasPrefix(path, "/api/update_") {
			s.serveUpdate(w, path, body)
			return
		}
		writeJSON(w, 404, errorEnvelope("ERR000404", "unknown endpoint "+path, "bad_request", ""))
	}
}

func queryArray(body *jsonobj.Object, key string) []any {
	query, ok := body.Object("query")
	if !ok || query == nil {
		return nil
	}
	items, _ := query.Array(key)
	return items
}

func hasWord(body *jsonobj.Object, word string) bool {
	for _, item := range queryArray(body, "words") {
		if item == word {
			return true
		}
	}
	return false
}

func (s *Server) serveGetKyous(w http.ResponseWriter, body *jsonobj.Object) {
	limit := 50
	if n, ok := body.Int("limit"); ok && n > 0 {
		limit = int(n)
	}
	countOnly, _ := body.Bool("count_only")
	groupBy := stringOf(body, "group_by")
	cursor := stringOf(body, "cursor")
	if hasWord(body, "nothing") {
		writeJSON(w, 200, envelope("returned_count", 0, "remaining_count", 0, "has_more", false, "total_count", 0))
		return
	}
	if countOnly {
		writeJSON(w, 200, envelope("returned_count", 0, "remaining_count", 0, "has_more", false, "total_count", 42))
		return
	}
	if groupBy != "" {
		writeJSON(w, 200, envelope("returned_count", 0, "remaining_count", 0, "has_more", false, "total_count", 8, "buckets", mustArr(kyousBuckets)))
		return
	}
	var all []any
	if cursor != "" {
		all = mustArr(kyousPage2)
	} else {
		all = mustArr(kyousPage1)
	}
	page := all
	if limit < len(all) {
		page = all[:limit]
	}
	remaining := len(all) - len(page)
	if cursor == "" {
		remaining += 2 // 2ページ目のぶん
	}
	res := envelope("kyous", page)
	if cursor == "" {
		res.Set("total_count", len(mustArr(kyousPage1))+len(mustArr(kyousPage2)))
	}
	res.Set("returned_count", len(page))
	res.Set("remaining_count", remaining)
	res.Set("has_more", remaining > 0)
	if remaining > 0 && len(page) > 0 {
		last := page[len(page)-1].(*jsonobj.Object)
		res.Set("next_cursor", stringOf(last, "related_time")+"::"+stringOf(last, "id"))
	}
	if cursor == "" {
		res.Set("plugins", mustArr(kyousPlugins))
	}
	if hasWord(body, "partial") {
		res.Set("partial", true)
		res.Set("warnings", jsonobj.Strings("some attached data could not be loaded (sample warning)"))
	}
	if hasWord(body, "warn") {
		res.Set("warnings", jsonobj.Strings("unknown data_type filter value ignored: sample"))
	}
	writeJSON(w, 200, res)
}

func (s *Server) serveKftl(w http.ResponseWriter, body *jsonobj.Object) {
	text := stringOf(body, "kftl_text")
	key := stringOf(body, "idempotency_key")
	if key == "conflict-key" {
		writeJSON(w, 409, errorEnvelope("ERR000423", "同じ冪等キーで別の本文が送られました", "conflict", "idempotency_conflict"))
		return
	}
	if key == "replay-key" {
		writeJSON(w, 200, envelope("created", mustArr(kftlCreated), "replayed", true))
		return
	}
	if strings.TrimSpace(text) == "" {
		writeJSON(w, 200, envelope("created", jsonobj.Arr(), "replayed", false))
		return
	}
	msgs := jsonobj.Arr(jsonobj.Obj("message", "3件の記録を保存しました", "level", "info"))
	res := envelope("created", mustArr(kftlCreated), "replayed", false)
	res.Set("messages", msgs)
	writeJSON(w, 200, res)
}

// historyEndpoints は型別 get エンドポイント → (型名, 応答キー)。
var historyEndpoints = map[string][2]string{
	"/api/get_kmemo":                     {"kmemo", "kmemo_histories"},
	"/api/get_urlog":                     {"urlog", "urlog_histories"},
	"/api/get_nlog":                      {"nlog", "nlog_histories"},
	"/api/get_lantana":                   {"lantana", "lantana_histories"},
	"/api/get_timeis":                    {"timeis", "timeis_histories"},
	"/api/get_mi":                        {"mi", "mi_histories"},
	"/api/get_kc":                        {"kc", "kc_histories"},
	"/api/get_tag_histories_by_tag_id":   {"tag", "tag_histories"},
	"/api/get_text_histories_by_text_id": {"text", "text_histories"},
	"/api/get_rekyou":                    {"rekyou", "rekyou_histories"},
	"/api/get_mirekyou":                  {"mirekyou", "mirekyou_histories"},
	"/api/get_gkill_notification_histories_by_notification_id": {"notification", "notification_histories"},
}

func (s *Server) serveHistory(w http.ResponseWriter, path string, body *jsonobj.Object) {
	spec, ok := historyEndpoints[path]
	if !ok {
		writeJSON(w, 404, errorEnvelope("ERR000404", "unknown endpoint "+path, "bad_request", ""))
		return
	}
	id := stringOf(body, "id")
	text, found := histories[spec[0]+"/"+id]
	if !found {
		writeJSON(w, 200, envelope(spec[1], jsonobj.Arr()))
		return
	}
	writeJSON(w, 200, envelope(spec[1], mustArr(text)))
}

// entityDataTypes は gkill が add / update 応答の実体に付ける射影名（エンティティ名と違う型）。
var entityDataTypes = map[string]string{
	"mi":       "mi_create",
	"timeis":   "timeis_start",
	"mirekyou": "mirekyou_create",
}

func responseKyou(entity *jsonobj.Object, dataType string) *jsonobj.Object {
	relatedTime := stringOf(entity, "related_time")
	if relatedTime == "" {
		relatedTime = stringOf(entity, "start_time")
	}
	if relatedTime == "" {
		relatedTime = stringOf(entity, "create_time")
	}
	return jsonobj.Obj(
		"id", entity.Value("id"),
		"rep_name", "Rep_fakepc",
		"data_type", dataType,
		"related_time", relatedTime,
		"create_time", entity.Value("create_time"),
		"update_time", entity.Value("update_time"),
		"is_deleted", entity.Value("is_deleted"),
	)
}

func (s *Server) serveAdd(w http.ResponseWriter, path string, body *jsonobj.Object) {
	kind := strings.TrimPrefix(path, "/api/add_")
	entity, ok := body.Object(kind)
	if !ok || entity == nil {
		writeJSON(w, 400, errorEnvelope("ERR000002", "request body lacks "+kind, "bad_request", ""))
		return
	}
	if kind == "tag" || kind == "text" {
		if stringOf(entity, "target_id") == "missing-target" {
			writeJSON(w, 404, errorEnvelope("ERR000092", "対象が見つかりません", "not_found", "target_missing"))
			return
		}
	}
	stored := entity.Clone()
	stored.Set("rep_name", "Rep_fakepc")
	if projected, ok := entityDataTypes[kind]; ok {
		stored.Set("data_type", projected)
	}
	if kind == "urlog" {
		stored.Set("title", "取得したタイトル")
		stored.Set("description", "取得した説明")
		stored.Set("favicon_image", "data:image/png;base64,AAAA")
		stored.Set("thumbnail_image", "data:image/png;base64,BBBB")
	}
	res := envelope("added_"+kind, stored)
	if kind != "tag" && kind != "text" {
		res.Set("added_kyou", responseKyou(stored, stringOf(stored, "data_type")))
	}
	writeJSON(w, 200, res)
}

func (s *Server) serveUpdate(w http.ResponseWriter, path string, body *jsonobj.Object) {
	kind := strings.TrimPrefix(path, "/api/update_")
	if kind == "gkill_notification" {
		kind = "notification"
	}
	entity, ok := body.Object(kind)
	if !ok || entity == nil {
		writeJSON(w, 400, errorEnvelope("ERR000002", "request body lacks "+kind, "bad_request", ""))
		return
	}
	stored := entity.Clone()
	stored.Set("rep_name", "Rep_fakepc")
	if projected, ok := entityDataTypes[kind]; ok {
		stored.Set("data_type", projected)
	}
	res := envelope("updated_"+kind, stored)
	res.Set("updated_kyou", responseKyou(stored, stringOf(stored, "data_type")))
	writeJSON(w, 200, res)
}

// serveGetSkill は /api/get_skill の固定応答。path を省くと SKILL.md とファイル一覧、指定するとそのファイル。
func serveGetSkill(w http.ResponseWriter, body *jsonobj.Object) {
	name := stringOf(body, "name")
	if name != fakeSkillName {
		writeJSON(w, 404, errorEnvelope("ERR000430", "スキルが見つかりません ("+name+")", "not_found", ""))
		return
	}
	filePath := stringOf(body, "path")
	file := func(size int64, isText bool, revision string) *jsonobj.Object {
		return jsonobj.Obj("path", filePath, "size", size, "is_text", isText, "revision", revision, "updated_time", "2026-09-20T10:00:00+09:00")
	}
	switch filePath {
	case "":
		writeJSON(w, 200, envelope("skill", mustObj(skillDetail), "file", nil))
	case "references/tags.md":
		f := file(22, true, "1234567890abcdef")
		f.Set("content", "# タグの意味\n- 日記\n")
		f.Set("content_base64", "")
		f.Set("content_omitted", false)
		writeJSON(w, 200, envelope("skill", nil, "file", f))
	case "assets/logo.png":
		f := file(8, false, "0f0f0f0f0f0f0f0f")
		f.Set("content", "")
		f.Set("content_base64", "iVBORw0KGgo=")
		f.Set("content_omitted", false)
		writeJSON(w, 200, envelope("skill", nil, "file", f))
	case "assets/big.bin":
		f := file(20000000, false, "ffffffffffffffff")
		f.Set("content", "")
		f.Set("content_base64", "")
		f.Set("content_omitted", true)
		writeJSON(w, 200, envelope("skill", nil, "file", f))
	default:
		writeJSON(w, 404, errorEnvelope("ERR000431", "スキルの中にそのファイルがありません ("+filePath+")", "not_found", ""))
	}
}

// serveWriteSkillFile は /api/write_skill_file の固定応答。revision "stale" は食い違い、
// 既にあるスキルの SKILL.md を revision なしで書くのは「既にある」。
func serveWriteSkillFile(w http.ResponseWriter, body *jsonobj.Object) {
	revision := stringOf(body, "revision")
	path := stringOf(body, "path")
	if revision == "stale" {
		writeJSON(w, 409, errorEnvelope("ERR000437", "ファイルが読んだ後に書き換えられています。読み直してから書いてください ("+path+": the current revision is a1b2c3d4e5f60718 (re-read the file before writing))", "conflict", ""))
		return
	}
	if revision == "" && path == "SKILL.md" && stringOf(body, "name") == fakeSkillName {
		writeJSON(w, 409, errorEnvelope("ERR000436", "そのファイルは既にあります（上書きするには revision を渡してください） (SKILL.md)", "conflict", ""))
		return
	}
	writeJSON(w, 200, envelope("path", path, "revision", "0123456789abcdef"))
}
