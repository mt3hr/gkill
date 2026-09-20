package mcp

// 「tools/list どおりに呼ぶと失敗しない」の契約（2026-09-14 レビュー P0）。
//
// ChatGPT から「公開スキーマに載っている検索条件名で呼ぶと未知の引数として拒否される」が
// 実測された。あの件はクライアントが古い一覧を握っていただけで、ソースは両側とも直っていたが、
// 同じ症状は「スキーマと正規化器の片方だけを改名する」だけで再現できる。
// ここではその事故を構造的に防ぐ:
//
//  1. スキーマのキー集合と正規化器の受理集合が一致する（廃止済みだけが受理側に多い）
//  2. 各ツールを「スキーマの全プロパティを指定して」呼んでも未知キーで拒否されない
//  3. read / write / readwrite の tools/list に載る同名ツールは同じ JSON
//     （gkill_status だけ description 末尾の schema_revision が違う。それが唯一の例外）
//  4. initialize の version・gkill_status の description・gkill_status の応答が同じ世代を指す

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ---------------------------------------------------------------------------
// 1. キー集合の一致
// ---------------------------------------------------------------------------
func TestAdvertisedKeysAndAcceptedKeysAgree(t *testing.T) {
	t.Run("FIND_QUERY_SCHEMA.properties == KYOUS_QUERY_ALL_FIELDS minus deprecated", func(t *testing.T) {
		advertised := NewStringSet(objAt(t, FindQuerySchema, "properties").Keys()...)
		accepted := NewStringSet()
		for _, key := range KyousQueryAllFields.Values() {
			if !DeprecatedQueryFields.Has(key) {
				accepted.Add(key)
			}
		}
		expectEqual(t, advertised.Sorted(), accepted.Sorted())
		// 廃止済みは受理側にだけある（公開しない・受理はする）
		for _, key := range DeprecatedQueryFields.Values() {
			expectTrue(t, KyousQueryAllFields.Has(key), "%s is not accepted", key)
			expectTrue(t, !advertised.Has(key), "%s is advertised", key)
		}
	})

	t.Run("gkill_get_kyous top-level properties == KYOUS_TOP_LEVEL_FIELDS minus deprecated", func(t *testing.T) {
		tool := findTool(ReadTools, "gkill_get_kyous")
		advertised := NewStringSet(objAt(t, tool, "inputSchema", "properties").Keys()...)
		accepted := NewStringSet()
		for _, key := range KyousTopLevelFields.Values() {
			if !DeprecatedTopLevelArgs.Has(key) {
				accepted.Add(key)
			}
		}
		expectEqual(t, advertised.Sorted(), accepted.Sorted())
		for _, key := range DeprecatedTopLevelArgs.Values() {
			expectTrue(t, KyousTopLevelFields.Has(key), "%s is not accepted", key)
			expectTrue(t, !advertised.Has(key), "%s is advertised", key)
		}
	})
}

// ---------------------------------------------------------------------------
// 2. 全プロパティ指定のスモーク
// ---------------------------------------------------------------------------

const sampleDateTime = "2026-01-02T03:04:05+09:00"
const sampleID = "00000000-0000-4000-8000-000000000001"

// sampleString は引数名から「型は string だが中身に形式がある」値を決める。
// スキーマの type / enum / minimum だけでは日時・URL・カーソルの形が分からない。
func sampleString(key string, schema *jsonobj.Object) string {
	if enum, ok := schema.Array("enum"); ok && len(enum) > 0 {
		return jsString(enum[0])
	}
	if key == "cursor" {
		description, _ := schema.String("description")
		if strings.Contains(description, "gkill_get_kyous cursor") {
			return EncodeGpsCursor(sampleDateTime, 0) // GPS 側（説明文が get_kyous のカーソルと別物だと言う）
		}
		return sampleDateTime + "::" + sampleID // Kyou 側（複合カーソル）
	}
	switch key {
	case "thumb":
		return "512x512"
	case "url":
		return "https://example.com/"
	case "locale_name":
		return "ja"
	case "playing_time":
		return "now"
	}
	if regexp.MustCompile(`(_date|_time)$`).MatchString(key) || key == "update_time" {
		return sampleDateTime
	}
	switch key {
	case "id", "target_id":
		return sampleID
	case "data_type":
		return "kmemo"
	case "board_name":
		return "Inbox"
	case "kftl_text":
		return "sample"
	}
	return "sample"
}

func sampleValue(t *testing.T, key string, schema *jsonobj.Object) any {
	t.Helper()
	typeName := ""
	if types, ok := schema.Array("type"); ok {
		for _, candidate := range types {
			if candidate != "null" {
				typeName = jsString(candidate)
				break
			}
		}
	} else {
		typeName, _ = schema.String("type")
	}
	switch typeName {
	case "string":
		return sampleString(key, schema)
	case "integer", "number":
		if minimum, ok := schema.Float("minimum"); ok {
			if minimum > 1 {
				return minimum
			}
		}
		return 1
	case "boolean":
		return true
	case "array":
		items, ok := schema.Object("items")
		if !ok {
			items = obj("type", "string")
		}
		return arr(sampleValue(t, key, items))
	case "object":
		return sampleObject(t, schema)
	default:
		t.Fatalf("no sample for %s: %s", key, jsonobj.MarshalString(schema))
		return nil
	}
}

func sampleObject(t *testing.T, schema *jsonobj.Object) *jsonobj.Object {
	t.Helper()
	out := obj()
	if properties, ok := schema.Object("properties"); ok {
		for _, key := range properties.Keys() {
			out.Set(key, sampleValue(t, key, objAt(t, properties, key)))
		}
	}
	// 時間帯の窓・数値レンジは「始 <= 終」でないと意味検証で落ちるので、同じ値にする
	if out.Has("period_of_time_start_time_second") {
		out.Set("period_of_time_start_time_second", 0)
	}
	if out.Has("period_of_time_end_time_second") {
		out.Set("period_of_time_end_time_second", 0)
	}
	if out.Has("period_of_time_week_of_days") {
		out.Set("period_of_time_week_of_days", arr(0))
	}
	if out.Has("num_min") && out.Has("num_max") {
		out.Set("num_max", out.Value("num_min"))
	}
	return out
}

// smokeApiResponse は型別エンドポイントの応答（履歴1件）と、その他の応答を1つの mock にまとめる。
// どのツールが何を読むかを個別に知らなくてよいよう、全部入りで返す。
// deleted: 履歴の最新版を削除済みにする（gkill_restore_kyou は未削除だと already active で断る）。
func smokeApiResponse(deleted bool) *jsonobj.Object {
	response := obj("errors", nil, "messages", nil, "kyous", arr(), "boards", strs("Inbox"), "tag_names", arr(), "rep_names", arr(), "gps_logs", arr(),
		"rep_infos", arr(), "canonical_rep_types", arr(), "plugins", arr(), "attached_data_reps", arr(),
		"application_config", obj("user_id", "testuser", "device", "testdevice"), "created", arr())
	for _, target := range EntityTargets {
		response.Set(target.HistoriesKey, arr(
			obj("id", sampleID, "data_type", target.DataType, "is_deleted", deleted, "update_time", "2020-01-01T00:00:00+09:00", "tag", "t", "text", "x", "content", "c", "title", "t", "url", "https://example.com/", "amount", 1, "mood", 1, "num_value", 1, "is_checked", false, "board_name", "Inbox"),
		))
	}
	return response
}

func smokeCtx(deleted bool) *CallContext {
	client := &mockClient{
		callApi: func(_ string, _ *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
			return smokeApiResponse(deleted), nil
		},
		fetchFile: func(_ string, _ string) (*FileResponse, error) {
			return &FileResponse{Buffer: []byte("x"), ContentType: "image/jpeg"}, nil
		},
		login: func() (string, error) { return "sid", nil },
	}
	return &CallContext{
		Client:           client,
		SID:              "sid",
		UserID:           "testuser",
		AppName:          "gkill_mcp_readwrite",
		IsLocalTransport: true,
		Server:           &Server{ServerKind: "readwrite", ServerName: "x", ServerVersion: "0", SchemaRevision: "0", Tools: nil, IsLocalTransport: true, StartedAt: time.Now()},
	}
}

// isUnknownKeyError: 未知キーの拒否だけを事故とみなす。意味検証（cursor と count_only の併用など）で落ちたら、
// その引数を除いて呼び直す —— 「スキーマにある名前を正規化器が知らない」だけを検出したい。
func isUnknownKeyError(err error) bool {
	apiErr, ok := AsGkillApiError(err)
	return ok && regexp.MustCompile(`is not supported`).MatchString(apiErr.Message)
}

func dropField(args *jsonobj.Object, field string) *jsonobj.Object {
	path := strings.Split(strings.TrimPrefix(field, "arguments."), ".")
	next := jsonobj.DeepClone(args).(*jsonobj.Object)
	cursor := next
	for _, segment := range path[:len(path)-1] {
		child, _ := cursor.Object(segment)
		cursor = child
	}
	cursor.Delete(path[len(path)-1])
	return next
}

type smokeResult struct {
	ok      bool
	dropped []string
}

func callWithEverySchemaField(t *testing.T, tool *jsonobj.Object, invoke func(args *jsonobj.Object) error) smokeResult {
	t.Helper()
	name := strAt(t, tool, "name")
	args := sampleObject(t, objAt(t, tool, "inputSchema"))
	dropped := []string{}
	for range 12 {
		err := invoke(args)
		if err == nil {
			return smokeResult{ok: true, dropped: dropped}
		}
		if isUnknownKeyError(err) {
			t.Fatalf("%s: a field advertised in inputSchema is rejected as unknown — %s", name, err.Error())
		}
		field, ok := DetailField(err)
		if !ok {
			t.Fatalf("%s: unexpected failure with all schema fields set — %s", name, err.Error())
		}
		dropped = append(dropped, field)
		args = dropField(args, field)
	}
	t.Fatalf("%s: gave up after dropping %s", name, strings.Join(dropped, ", "))
	return smokeResult{}
}

func TestEveryAdvertisedPropertyIsAcceptedByTheToolsNormalizer(t *testing.T) {
	for _, tool := range ReadTools {
		name := strAt(t, tool, "name")
		t.Run(name+" (read)", func(t *testing.T) {
			ctx := smokeCtx(false)
			result := callWithEverySchemaField(t, tool, func(args *jsonobj.Object) error {
				_, err := HandleReadToolCall(ctx, name, args)
				return err
			})
			expectTrue(t, result.ok, "not ok")
		})
	}
	for _, tool := range WriteTools {
		name := strAt(t, tool, "name")
		t.Run(name+" (write)", func(t *testing.T) {
			ctx := smokeCtx(name == "gkill_restore_kyou")
			result := callWithEverySchemaField(t, tool, func(args *jsonobj.Object) error {
				_, err := HandleWriteToolCall(ctx, name, args)
				return err
			})
			expectTrue(t, result.ok, "not ok")
		})
	}
	for _, tool := range PluginTools {
		name := strAt(t, tool, "name")
		t.Run(name+" (plugin)", func(t *testing.T) {
			ctx := smokeCtx(false)
			result := callWithEverySchemaField(t, tool, func(args *jsonobj.Object) error {
				_, err := HandlePluginToolCall(func(pathname string, body *jsonobj.Object) (*jsonobj.Object, error) {
					return ctx.Client.CallApi(context.Background(), pathname, body, false, "")
				}, name, args)
				return err
			})
			expectTrue(t, result.ok, "not ok")
		})
	}
}

// ---------------------------------------------------------------------------
// 3. 3サーバの tools/list に載る同名ツールは同じ JSON
// ---------------------------------------------------------------------------

func contractMockClient() *mockClient {
	return &mockClient{
		callApi: func(_ string, _ *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
			return obj("application_config", obj("user_id", "testuser", "device", "testdevice")), nil
		},
		login:         func() (string, error) { return "sid", nil },
		defaultLocale: "ja",
	}
}

func toolsListOf(t *testing.T, server *Server) []*jsonobj.Object {
	t.Helper()
	response := server.HandleMessage(context.Background(), obj("jsonrpc", "2.0", "id", 1, "method", "tools/list"), nil)
	tools := []*jsonobj.Object{}
	for _, tool := range arrAt(t, response, "result", "tools") {
		tools = append(tools, tool.(*jsonobj.Object))
	}
	return tools
}

func TestTheThreeServersAdvertiseIdenticalDefinitionsForASharedToolName(t *testing.T) {
	t.Run("same name => same JSON (gkill_status differs only by its schema_revision mark)", func(t *testing.T) {
		servers := []*Server{NewReadServer(contractMockClient(), nil), NewWriteServer(contractMockClient(), nil), NewReadWriteServer(contractMockClient(), nil)}
		lists := [][]*jsonobj.Object{}
		for _, server := range servers {
			lists = append(lists, toolsListOf(t, server))
		}
		byName := map[string]string{}
		for _, tools := range lists {
			for _, tool := range tools {
				name := strAt(t, tool, "name")
				canonical := jsonobj.MarshalString(tool.Clone().Set("description", StripSchemaRevisionMark(strAt(t, tool, "description"))))
				seen, ok := byName[name]
				if !ok {
					byName[name] = canonical
				} else {
					expectTrue(t, canonical == seen, "%s differs between servers", name)
				}
			}
		}
		// gkill_status は3サーバ全部に載る
		for _, tools := range lists {
			expectTrue(t, findTool(tools, StatusToolName) != nil, "gkill_status missing")
		}
	})
}

// ---------------------------------------------------------------------------
// 4. 世代の一致
// ---------------------------------------------------------------------------
func TestSchemaRevisionIsConsistentPerServer(t *testing.T) {
	cases := []struct {
		kind string
		make func() *Server
	}{
		{"read", func() *Server { return NewReadServer(contractMockClient(), nil) }},
		{"write", func() *Server { return NewWriteServer(contractMockClient(), nil) }},
		{"readwrite", func() *Server { return NewReadWriteServer(contractMockClient(), nil) }},
	}
	for _, tc := range cases {
		t.Run(tc.kind+": initialize version, stamped description, response and recomputation agree", func(t *testing.T) {
			server := tc.make()
			initialize := server.HandleMessage(context.Background(), obj("jsonrpc", "2.0", "id", 1, "method", "initialize"), nil)
			version := strAt(t, objAt(t, initialize, "result", "serverInfo"), "version")
			fromVersion := regexp.MustCompile(`\+schema\.([0-9a-f]{12})$`).FindStringSubmatch(version)
			tools := toolsListOf(t, server)
			status := findTool(tools, StatusToolName)
			fromDescription := regexp.MustCompile(` \[schema_revision: ([0-9a-f]{12})\]$`).FindStringSubmatch(strAt(t, status, "description"))
			call := server.HandleMessage(context.Background(), obj(
				"jsonrpc", "2.0",
				"id", 2,
				"method", "tools/call",
				"params", obj("name", StatusToolName, "arguments", obj()),
			), nil)
			expectEqual(t, objAt(t, call, "result").Value("isError"), false)
			structured := objAt(t, call, "result", "structuredContent")
			fromResponse := strAt(t, structured, "schema_revision")
			expectTrue(t, fromVersion != nil, "version %q carries no schema revision", version)
			expectTrue(t, fromDescription != nil, "description carries no schema revision")
			expectEqual(t, fromDescription[1], fromVersion[1])
			expectEqual(t, fromResponse, fromVersion[1])
			// 配られた一覧から計算し直しても同じ（焼き込みは計算対象に入らない）
			expectEqual(t, ComputeSchemaRevision(tools), fromVersion[1])
			expectEqual(t, structured.Value("server_kind"), tc.kind)
			expectEqual(t, structured.Value("tool_count"), len(tools))
		})
	}
}

// sortedKeys はテストの補助（map のキーを安定順で）。
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var _ = fmt.Sprintf
