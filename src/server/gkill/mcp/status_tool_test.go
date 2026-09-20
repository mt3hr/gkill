package mcp

// status_tool.go — ツール一覧の世代（schema_revision）と gkill_status への焼き込み。
//
// - 同じ一覧なら同じ値、1文字でも違えば別の値（手書きの版番号ではない）
// - gkill_status 自身は計算対象に入らない（description に自分の値を焼き込むため）
// - 焼き込みは静的な配列を書き換えず、gkill_status だけ差し替えた新しい配列を返す
// - 焼き込みは冪等（二重に付かない）
// - 3サーバはツール集合が違うので世代も違う

import (
	"fmt"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

func sampleTools() []*jsonobj.Object {
	return []*jsonobj.Object{
		obj("name", StatusToolName, "description", "status", "inputSchema", obj("type", "object", "properties", obj())),
		obj("name", "gkill_x", "description", "x", "inputSchema", obj("type", "object", "properties", obj("a", obj("type", "string")))),
	}
}

func cloneTools(tools []*jsonobj.Object) []*jsonobj.Object {
	out := make([]*jsonobj.Object, len(tools))
	for i, tool := range tools {
		out[i] = jsonobj.DeepClone(tool).(*jsonobj.Object)
	}
	return out
}

func TestComputeSchemaRevision(t *testing.T) {
	sample := sampleTools()

	t.Run("is deterministic and 12 hex chars", func(t *testing.T) {
		first := ComputeSchemaRevision(sample)
		mustMatch(t, first, fmt.Sprintf("^[0-9a-f]{%d}$", SchemaRevisionLength))
		expectTrue(t, ComputeSchemaRevision(sample) == first, "revision changed between calls")
	})

	t.Run("changes when any tool's description or schema changes", func(t *testing.T) {
		base := ComputeSchemaRevision(sample)
		renamedField := cloneTools(sample)
		schema, _ := renamedField[1].Object("inputSchema")
		schema.Set("properties", obj("b", obj("type", "string")))
		expectTrue(t, ComputeSchemaRevision(renamedField) != base, "renamed field kept the revision")
		editedDescription := cloneTools(sample)
		editedDescription[1].Set("description", "x!")
		expectTrue(t, ComputeSchemaRevision(editedDescription) != base, "edited description kept the revision")
	})

	t.Run("ignores gkill_status itself, so stamping does not move the value", func(t *testing.T) {
		before := ComputeSchemaRevision(sample)
		stamped := StampSchemaRevision(sample, before)
		expectTrue(t, ComputeSchemaRevision(stamped) == before, "stamping moved the revision")
		// gkill_status の説明文だけ変えても世代は動かない（自己参照を避けるため）
		statusEdited := cloneTools(sample)
		statusEdited[0].Set("description", "another status text")
		expectTrue(t, ComputeSchemaRevision(statusEdited) == before, "status description moved the revision")
	})

	t.Run("differs across the three real servers because their tool sets differ", func(t *testing.T) {
		read := ComputeSchemaRevision(concatTools(ReadTools, PluginTools))
		readwrite := ComputeSchemaRevision(concatTools(ReadTools, WriteTools, PluginTools))
		expectTrue(t, read != readwrite, "read and readwrite share a revision")
	})
}

func TestStampSchemaRevision(t *testing.T) {
	sample := sampleTools()

	t.Run("appends the mark to gkill_status only and leaves the source array untouched", func(t *testing.T) {
		revision := "0123456789ab"
		stamped := StampSchemaRevision(sample, revision)
		expectTrue(t, &stamped[0] != &sample[0], "stamped slice aliases the source")
		expectTrue(t, strAt(t, stamped[0], "description") == fmt.Sprintf("status [schema_revision: %s]", revision), "mark missing: %s", strAt(t, stamped[0], "description"))
		mustMatch(t, strAt(t, stamped[0], "description"), SchemaRevisionMarkRegex.String())
		// 元の配列は書き換えない（read / write / readwrite が同じ READ_TOOLS を共有する）
		expectTrue(t, strAt(t, sample[0], "description") == "status", "source description was rewritten")
		// gkill_status 以外は同じオブジェクト
		expectTrue(t, stamped[1] == sample[1], "non-status tool was copied")
	})

	t.Run("is idempotent: re-stamping replaces the mark instead of appending a second one", func(t *testing.T) {
		once := StampSchemaRevision(sample, "0123456789ab")
		twice := StampSchemaRevision(once, "ba9876543210")
		expectTrue(t, strAt(t, twice[0], "description") == "status [schema_revision: ba9876543210]", "got %s", strAt(t, twice[0], "description"))
		expectTrue(t, StripSchemaRevisionMark(strAt(t, twice[0], "description")) == "status", "strip failed")
	})

	t.Run("the real gkill_status definition carries no mark until a server stamps it", func(t *testing.T) {
		status := findTool(ReadTools, StatusToolName)
		expectTrue(t, status != nil, "gkill_status is not defined")
		mustNotMatch(t, strAt(t, status, "description"), SchemaRevisionMarkRegex.String())
		// 引数を取らない（引数を足すと古いスキーマの救済表の対象になり、
		// 「古さを確かめるツール自身が古いスキーマで壊れる」）
		properties := objAt(t, status, "inputSchema", "properties")
		expectEqual(t, properties.Keys(), []string{})
		additional, _ := objAt(t, status, "inputSchema").Bool("additionalProperties")
		expectTrue(t, !additional, "additionalProperties must be false")
	})
}

func concatTools(lists ...[]*jsonobj.Object) []*jsonobj.Object {
	out := []*jsonobj.Object{}
	for _, list := range lists {
		out = append(out, list...)
	}
	return out
}
