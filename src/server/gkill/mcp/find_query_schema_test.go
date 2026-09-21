package mcp

import (
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// query スキーマの形。キー集合は schema_contract_test.go が受理集合と突き合わせるので、
// ここは「全プロパティに type（か enum）と description があり、配列は items を持つ」ことだけを固定する
// （GKILL_MCP_UPDATE_GOLDEN で黙って追随してしまう欠けを、ゴールデンの外で止める）。
func TestFindQuerySchemaEveryPropertyIsTypedAndDescribed(t *testing.T) {
	schema := buildFindQuerySchema()
	typ, _ := schema.String("type")
	expectEqual(t, typ, "object")
	props, ok := schema.Object("properties")
	expectTrue(t, ok, "properties missing")
	expectTrue(t, len(props.Keys()) > 30, "suspiciously few properties: %d", len(props.Keys()))
	for _, key := range props.Keys() {
		prop, ok := props.Object(key)
		expectTrue(t, ok, "%s is not an object", key)
		propType, hasType := prop.String("type")
		_, hasEnum := jsonobj.AsArray(prop.Value("enum"))
		expectTrue(t, hasType || hasEnum, "%s has neither type nor enum", key)
		description, hasDescription := prop.String("description")
		expectTrue(t, hasDescription && description != "", "%s has no description", key)
		if propType == "array" {
			_, hasItems := prop.Object("items")
			expectTrue(t, hasItems, "%s is an array without items", key)
		}
	}
	// 廃止済みのキーは載せない（受理はするが公開しない。ADR-0620）
	for _, retired := range []string{"only_latest_data", "use_words", "use_tags"} {
		_, advertised := props.Object(retired)
		expectTrue(t, !advertised, "%s is advertised", retired)
	}
	// パッケージ変数は同じ生成結果
	expectEqual(t, jsonobj.MarshalString(FindQuerySchema), jsonobj.MarshalString(schema))
}
