package mcp

// 旧 Node 実装から採ったゴールデン（testdata/golden/）との一致検査。新旧の結果一致の機械証明。
//
//   - tools/list の JSON（3サーバ）と schema_revision がバイト単位で一致する
//   - 要求コーパス（requests.json、331 件）に対する tools/call の応答が、stdio / http の両モードで
//     バイト単位で一致する（時刻・UUID・トークンは採取時と同じ固定列）
//   - 同じコーパスで gkill へ送った要求（パス・クエリ・Cookie・本文）がバイト単位で一致する
//
// 採取の手順（Node 実装が消える前に1回だけ行った）: 偽 gkill（internal/fakegkill）を別プロセスで立て、
// 時刻と乱数を固定したプリロード付きで 3 つの stdio サーバへ NDJSON を流し、応答と偽 gkill の記録を保存した。
// http モードはサーバモジュールを直接 import して handlePayload(message, requestContext) を呼んだ。
// 唯一マスクするのは JSON でない応答本文のパーサ文言（detail.cause。V8 と Go で違う）だけ。
//
// 更新の手順（Node 実装はもう無い）: ツールの説明・引数・応答を意図して変えたときは
//
//	GKILL_MCP_UPDATE_GOLDEN=1 go test ./gkill/mcp/ -run Golden
//
// でゴールデンを Go の出力で書き直し、`git diff testdata/golden` を読んで意図した差分だけであることを
// 確かめてからコミットする（そのあと環境変数なしで走らせて緑を確認する）。以後のゴールデンは
// 「旧 Node との一致」ではなく「前回コミットした Go の出力との一致」を固定する回帰検査になる。

import (
	"bufio"
	"encoding/binary"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/internal/fakegkill"
	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

const goldenPasswordSha256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// goldenNow は採取時の固定時刻（2026-09-20T18:30:45.123Z = 2026-09-21T03:30:45.123+09:00）。
var goldenNow = time.Date(2026, 9, 20, 18, 30, 45, 123000000, time.UTC)

type goldenCase struct {
	Name        string
	KeepMessage bool
	Substitute  *jsonobj.Object
	Message     any
}

// goldenDir はゴールデンの置き場所。更新経路の自己検査（TestGoldenUpdatePathRewritesAndThenMatches）が
// 一時ディレクトリへ差し替える。
var goldenDir = filepath.Join("testdata", "golden")

func goldenPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(goldenDir, name)
}

func readGoldenFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(goldenPath(t, name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(data)
}

// updateGolden は GKILL_MCP_UPDATE_GOLDEN=1 のとき真。ゴールデンを Go の出力で書き直す。
func updateGolden() bool {
	return os.Getenv("GKILL_MCP_UPDATE_GOLDEN") == "1"
}

func writeGoldenFile(t *testing.T, name string, content string) {
	t.Helper()
	if err := os.WriteFile(goldenPath(t, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write golden %s: %v", name, err)
	}
	t.Logf("golden updated: %s", name)
}

func loadGoldenCases(t *testing.T) []goldenCase {
	t.Helper()
	parsed, err := jsonobj.Unmarshal([]byte(readGoldenFile(t, "requests.json")))
	expectNoError(t, err)
	items, ok := jsonobj.AsArray(parsed)
	expectTrue(t, ok, "requests.json is not an array")
	cases := []goldenCase{}
	for _, item := range items {
		o := item.(*jsonobj.Object)
		c := goldenCase{Name: jsString(o.Value("name")), Message: o.Value("message")}
		if keep, _ := o.Bool("keep_message"); keep {
			c.KeepMessage = true
		}
		if sub, ok := o.Object("substitute"); ok {
			c.Substitute = sub
		}
		cases = append(cases, c)
	}
	return cases
}

// loadGoldenLines は {"name":..., ...} の JSONL を name → 行の対応にする。
func loadGoldenLines(t *testing.T, name string) map[string]*jsonobj.Object {
	t.Helper()
	out := map[string]*jsonobj.Object{}
	scanner := bufio.NewScanner(strings.NewReader(readGoldenFile(t, name)))
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		o, err := jsonobj.UnmarshalObject([]byte(line))
		expectNoError(t, err)
		out[jsString(o.Value("name"))] = o
	}
	return out
}

// buildGoldenMessage は採取ハーネスの buildMessage と同じ規則（id の付与と substitute の解決）。
func buildGoldenMessage(c goldenCase, index int, previous map[string]any) any {
	var message any
	if o, ok := c.Message.(*jsonobj.Object); ok {
		message = o.Clone()
	} else if items, ok := jsonobj.AsArray(c.Message); ok {
		cloned := make([]any, len(items))
		for i, item := range items {
			if o, ok := item.(*jsonobj.Object); ok {
				cloned[i] = o.Clone()
			} else {
				cloned[i] = item
			}
		}
		message = cloned
	} else {
		message = c.Message
	}
	if !c.KeepMessage {
		if o, ok := message.(*jsonobj.Object); ok {
			o.Set("id", int64(index+1))
		}
	}
	if c.Substitute != nil {
		msg := message.(*jsonobj.Object)
		params, _ := msg.Object("params")
		arguments, _ := params.Object("arguments")
		for _, argName := range c.Substitute.Keys() {
			spec, _ := c.Substitute.Object(argName)
			var value any = previous[jsString(spec.Value("from"))]
			for _, part := range strings.Split(jsString(spec.Value("path")), ".") {
				o, ok := value.(*jsonobj.Object)
				if !ok || o == nil {
					value = jsonobj.Undefined
					break
				}
				value = o.Value(part)
			}
			if jsonobj.IsUndefined(value) {
				value = nil
			}
			arguments.Set(argName, value)
		}
	}
	return message
}

// goldenClock は採取時のプリロード（freeze.mjs）と同じ固定列を作る。
type goldenClock struct {
	uuidCounter  int
	tokenCounter int
}

func (g *goldenClock) install(t *testing.T) {
	t.Helper()
	originalNow := timeNow
	originalUUID := newUUID
	originalRandom := randomBytes
	originalLocal := time.Local
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	expectNoError(t, err)
	time.Local = tokyo
	timeNow = func() time.Time { return goldenNow }
	newUUID = func() string {
		g.uuidCounter++
		return "00000000-0000-4000-8000-" + padStart(itoa(g.uuidCounter), 12, "0")
	}
	randomBytes = func(n int) []byte {
		g.tokenCounter++
		buf := make([]byte, n)
		binary.BigEndian.PutUint32(buf[n-4:], uint32(g.tokenCounter))
		return buf
	}
	t.Cleanup(func() {
		timeNow = originalNow
		newUUID = originalUUID
		randomBytes = originalRandom
		time.Local = originalLocal
	})
}

// maskCause は JSON でない応答本文を読んだときのパーサ文言（detail.cause）を潰す。
// V8（"Unexpected token '<' ..."）と Go（"invalid character '<' ..."）で違う、唯一の意図した不一致。
var causeInTextRegex = regexp.MustCompile(`"cause": "(?:[^"\\]|\\.)*"`)

func maskCause(value any) any {
	switch x := value.(type) {
	case *jsonobj.Object:
		out := jsonobj.New()
		for _, key := range x.Keys() {
			if key == "cause" {
				out.Set(key, "<masked>")
				continue
			}
			out.Set(key, maskCause(x.Value(key)))
		}
		return out
	case string:
		return causeInTextRegex.ReplaceAllString(x, `"cause": "<masked>"`)
	}
	if items, ok := jsonobj.AsArray(value); ok {
		out := make([]any, len(items))
		for i, item := range items {
			out[i] = maskCause(item)
		}
		return out
	}
	return value
}

func maskedString(t *testing.T, raw string) string {
	t.Helper()
	parsed, err := jsonobj.Unmarshal([]byte(raw))
	expectNoError(t, err)
	return jsonobj.MarshalString(maskCause(parsed))
}

// goldenClient は採取時と同じ接続設定（ログインで sess-login を得る）。
func goldenClient(baseURL string) *GkillClient {
	return NewGkillClient(ClientConfig{BaseURL: baseURL, UserID: "testuser", PasswordSha256: goldenPasswordSha256, Locale: "ja"})
}

// sortConcurrentUpstream は rep をまたいで並列に投げるプラグイン本文の取得だけ、要求の並びを本文順に揃える。
// 並列の着順は Node でも Go でも決まっていない（同じ rep 内は直列で順序が決まる）。それ以外の要求の順序は触らない。
func sortConcurrentUpstream(items []any) []any {
	out := append([]any{}, items...)
	isPluginContent := func(item any) bool {
		o, _ := item.(*jsonobj.Object)
		return o != nil && o.Value("path") == GetPluginContentHTMLEndpoint
	}
	for start := 0; start < len(out); {
		if !isPluginContent(out[start]) {
			start++
			continue
		}
		end := start
		for end < len(out) && isPluginContent(out[end]) {
			end++
		}
		run := out[start:end]
		sort.SliceStable(run, func(i, j int) bool {
			return jsonobj.MarshalString(run[i]) < jsonobj.MarshalString(run[j])
		})
		start = end
	}
	return out
}

func recordsToAny(records []fakegkill.Record) []any {
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
	return items
}

// causeMaskedCases は detail.cause の文言差だけを許すケース（HTML の 502 を読んだもの）。
var causeMaskedCases = NewStringSet("status: gkill returns HTML 502", "kyous: gkill HTML 502")

// tools/list と schema_revision がコミット済みのゴールデン（採取時は旧 Node 実装の出力、以後は前回コミットした
// Go の出力）とバイト単位で一致すること。GKILL_MCP_UPDATE_GOLDEN=1 のときは先に書き直してから比べる。
func TestGoldenToolsListMatchesTheCommittedGoldenByteForByte(t *testing.T) {
	for _, kind := range ServerKinds {
		t.Run(kind, func(t *testing.T) {
			checkGoldenToolsList(t, kind)
		})
	}
}

func checkGoldenToolsList(t *testing.T, kind string) {
	t.Helper()
	server := NewServerForKind(kind, &mockClient{}, nil)
	got := jsonobj.MarshalString(toolsToAny(server.Tools))
	if updateGolden() {
		writeGoldenFile(t, "tools_list_"+kind+".json", got+"\n")
		writeGoldenFile(t, "schema_revision_"+kind+".txt", server.SchemaRevision+"\n")
	}
	expected := strings.TrimRight(readGoldenFile(t, "tools_list_"+kind+".json"), "\r\n")
	if got != expected {
		t.Fatalf("tools/list differs from the committed golden (kind=%s): got %d bytes, want %d bytes; first difference at %d", kind, len(got), len(expected), firstDifference(got, expected))
	}
	expectEqual(t, server.SchemaRevision, strings.TrimSpace(readGoldenFile(t, "schema_revision_"+kind+".txt")))
}

// 更新経路の自己検査。GKILL_MCP_UPDATE_GOLDEN=1 で一時ディレクトリへ書き直したゴールデンが、
// 環境変数なしの比較で通り、かつコミット済みのゴールデンと同じバイト列であること
// （＝ 更新経路と比較経路が同じ出力を扱っている。壊れていると「更新したのに赤い」か
// 「更新で黙ってコミット済みと違う内容になる」のどちらかが起きる）。
func TestGoldenUpdatePathRewritesAndThenMatches(t *testing.T) {
	committedDir := goldenDir
	goldenDir = t.TempDir()
	t.Cleanup(func() { goldenDir = committedDir })

	t.Setenv("GKILL_MCP_UPDATE_GOLDEN", "1")
	for _, kind := range ServerKinds {
		checkGoldenToolsList(t, kind) // 書き直してから比べる
	}
	t.Setenv("GKILL_MCP_UPDATE_GOLDEN", "")
	for _, kind := range ServerKinds {
		checkGoldenToolsList(t, kind) // 書き直したものと比べるだけ
		for _, name := range []string{"tools_list_" + kind + ".json", "schema_revision_" + kind + ".txt"} {
			rewritten, err := os.ReadFile(filepath.Join(goldenDir, name))
			expectNoError(t, err)
			committed, err := os.ReadFile(filepath.Join(committedDir, name))
			expectNoError(t, err)
			if string(rewritten) != string(committed) {
				t.Fatalf("%s: 更新経路の出力がコミット済みのゴールデンと違う（first difference at %d）", name, firstDifference(string(rewritten), string(committed)))
			}
		}
	}
}

func firstDifference(a, b string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) < len(b) {
		return len(a)
	}
	return len(b)
}

func replayGolden(t *testing.T, kind string, mode string, clock *goldenClock, fake *fakegkill.Server, baseURL string) {
	t.Helper()
	cases := loadGoldenCases(t)
	update := updateGolden()
	expectedResponses := loadGoldenLines(t, "responses_"+kind+"_"+mode+".jsonl")
	expectedUpstream := loadGoldenLines(t, "upstream_"+kind+"_"+mode+".jsonl")
	if !update {
		expectEqual(t, len(expectedResponses), len(cases))
	}
	// 更新モードで書き出す行（採取ハーネスと同じ {name, raw} / {name, requests} の JSONL）
	newResponses := make([]string, 0, len(cases))
	newUpstream := make([]string, 0, len(cases))

	originalVersion := ServerVersion
	ServerVersion = strings.TrimSpace(readGoldenFile(t, "package_version.txt"))
	t.Cleanup(func() { ServerVersion = originalVersion })

	fake.Reset()
	client := goldenClient(baseURL)
	server := NewServerForKind(kind, client, nil)
	var requestContext *RequestContext
	if mode == "http" {
		server.IsLocalTransport = false
		server.FileLinkContext = &FileLinkContext{PublicBaseURL: "https://mcp.example.test", Store: NewFileLinkStore(0)}
		requestContext = &RequestContext{SessionID: "sess-http", UserID: "httpuser", RemoteAddr: "127.0.0.1"}
	} else {
		server.IsLocalTransport = true
		server.CurrentUserID = client.UserID()
	}

	previous := map[string]any{}
	failures := 0
	for index, c := range cases {
		message := buildGoldenMessage(c, index, previous)
		response := server.HandlePayload(t.Context(), message, requestContext)
		raw := "null"
		var rawValue any
		if response != nil {
			raw = jsonobj.MarshalString(response)
			rawValue = raw
		}
		previous[c.Name] = response
		gotUpstream := recordsToAny(fake.Drain())
		newResponses = append(newResponses, jsonobj.MarshalString(jsonobj.Obj("name", c.Name, "raw", rawValue)))
		newUpstream = append(newUpstream, jsonobj.MarshalString(jsonobj.Obj("name", c.Name, "requests", gotUpstream)))
		if update {
			continue
		}

		expectedLine := expectedResponses[c.Name]
		expectedRaw := "null"
		if expectedLine != nil && expectedLine.Value("raw") != nil {
			expectedRaw = jsString(expectedLine.Value("raw"))
		}
		if raw != expectedRaw {
			ok := false
			if causeMaskedCases.Has(c.Name) && raw != "null" && expectedRaw != "null" {
				ok = maskedString(t, raw) == maskedString(t, expectedRaw)
			}
			if !ok {
				failures++
				if failures <= 5 {
					t.Errorf("[%s/%s] case %q response differs (diff at byte %d)\n got: %s\nwant: %s", kind, mode, c.Name, firstDifference(raw, expectedRaw), clip(raw, 600), clip(expectedRaw, 600))
				}
			}
		}

		wantUpstream := []any{}
		if line := expectedUpstream[c.Name]; line != nil {
			wantUpstream = arrayOrEmpty(line.Value("requests"))
		}
		gotText := jsonobj.MarshalString(sortConcurrentUpstream(gotUpstream))
		wantText := jsonobj.MarshalString(sortConcurrentUpstream(wantUpstream))
		if gotText != wantText {
			failures++
			if failures <= 5 {
				t.Errorf("[%s/%s] case %q upstream requests differ (diff at byte %d)\n got: %s\nwant: %s", kind, mode, c.Name, firstDifference(gotText, wantText), clip(gotText, 600), clip(wantText, 600))
			}
		}
	}
	if failures > 5 {
		t.Errorf("[%s/%s] %d cases differ in total (first 5 shown)", kind, mode, failures)
	}
	if update {
		writeGoldenFile(t, "responses_"+kind+"_"+mode+".jsonl", strings.Join(newResponses, "\n")+"\n")
		writeGoldenFile(t, "upstream_"+kind+"_"+mode+".jsonl", strings.Join(newUpstream, "\n")+"\n")
	}
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func TestGoldenReplayStdio(t *testing.T) {
	fake := fakegkill.New()
	upstream := httptest.NewServer(fake)
	t.Cleanup(upstream.Close)
	for _, kind := range ServerKinds {
		t.Run(kind, func(t *testing.T) {
			// 採取は種別ごとに別プロセスだったので、乱数の列も種別ごとに始めからにする。
			clock := &goldenClock{}
			clock.install(t)
			replayGolden(t, kind, "stdio", clock, fake, upstream.URL)
		})
	}
}

func TestGoldenReplayHTTP(t *testing.T) {
	fake := fakegkill.New()
	upstream := httptest.NewServer(fake)
	t.Cleanup(upstream.Close)
	// 採取は1プロセスで read → write → readwrite の順に回したので、UUID とトークンの列は種別をまたいで続く。
	clock := &goldenClock{}
	clock.install(t)
	for _, kind := range ServerKinds {
		t.Run(kind, func(t *testing.T) {
			replayGolden(t, kind, "http", clock, fake, upstream.URL)
		})
	}
}

// TestGoldenFilesRoundTripThroughJsonobj は採取した応答が jsonobj で読み書きしてもバイト単位で変わらないことを固定する
// （JSON.stringify 互換の直列化の実証。ここが崩れると再生の比較が「直列化の差」で落ちる）。
func TestGoldenFilesRoundTripThroughJsonobj(t *testing.T) {
	for _, kind := range ServerKinds {
		for _, mode := range []string{"stdio", "http"} {
			lines := loadGoldenLines(t, "responses_"+kind+"_"+mode+".jsonl")
			for name, line := range lines {
				raw := line.Value("raw")
				if raw == nil {
					continue
				}
				text := jsString(raw)
				parsed, err := jsonobj.Unmarshal([]byte(text))
				expectNoError(t, err)
				if again := jsonobj.MarshalString(parsed); again != text {
					t.Fatalf("[%s/%s] %q does not round-trip (diff at byte %d)", kind, mode, name, firstDifference(again, text))
				}
			}
		}
	}
}
