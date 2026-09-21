package mcp

// tools/list のバイト量の予算（2026-09-14 レビュー P0）。
//
// 予算ファイル tool_schema_budget.json は「現状の実測」で、増やすときは
// `gkill_server mcp schema-budget --update`（npm run mcp:schema-budget -- --update）で明示的に更新する。
// 減ったときも追随させる。判定と文言の正本は tool_schema_budget.go（サブコマンドと共用）。

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestToolsListByteBudget(t *testing.T) {
	current := MeasureToolSchemaBytes()
	budget, err := ReadToolSchemaBudget()
	expectNoError(t, err)
	rows := CompareToolSchemaBudget(current, budget)

	for _, kind := range ServerKinds {
		t.Run(kind+" server stays within its recorded budget", func(t *testing.T) {
			var row BudgetRow
			for _, candidate := range rows {
				if candidate.Kind == kind {
					row = candidate
				}
			}
			expectTrue(t, row.Verdict == "ok", "%s", DescribeBudgetRow(row))
		})
	}

	t.Run("measurement is deterministic (the schema_revision mark is fixed-length)", func(t *testing.T) {
		expectEqual(t, budgetToObj(MeasureToolSchemaBytes()), budgetToObj(current))
	})
}

func TestCompareToolSchemaBudget(t *testing.T) {
	t.Run("classifies over / under / ok / missing", func(t *testing.T) {
		budget := map[string]int{"read": 1000, "write": 1000, "readwrite": 1000}
		rows := CompareToolSchemaBudget(
			map[string]int{"read": 1001, "write": 1000 - ToolSchemaBudgetSlackBytes - 1, "readwrite": 1000 - ToolSchemaBudgetSlackBytes},
			budget,
		)
		verdicts := []string{}
		for _, row := range rows {
			verdicts = append(verdicts, row.Verdict)
		}
		expectEqual(t, verdicts, []string{"over", "under", "ok"})
		mustMatch(t, DescribeBudgetRow(rows[0]), `(?s)\+1 over the budget of 1000.*--update`)
		mustMatch(t, DescribeBudgetRow(rows[1]), `(?s)under the budget.*--update`)
		expectEqual(t, CompareToolSchemaBudget(map[string]int{"read": 1, "write": 1, "readwrite": 1}, map[string]int{})[0].Verdict, "missing")
		expectEqual(t, CompareToolSchemaBudget(map[string]int{"read": 1, "write": 1, "readwrite": 1}, nil)[0].Verdict, "missing")
	})
}

func budgetToObj(budget map[string]int) any {
	out := obj()
	for _, kind := range ServerKinds {
		out.Set(kind, budget[kind])
	}
	return out
}

// `mcp schema-budget --update` の書き戻し経路。WriteToolSchemaBudget が書いたファイルを
// ReadToolSchemaBudgetFile が同じ値で読み戻し、Compare が全行 ok になること。
// 書式（2スペース・行は ServerKinds の順・末尾改行）も固定する: 旧 tool-schema-budget.json と同じ形にして
// `git diff` で増減だけが見えるようにするため。
func TestWriteToolSchemaBudgetRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool_schema_budget.json")
	current := MeasureToolSchemaBytes()

	expectNoError(t, WriteToolSchemaBudget(current, path))
	written, err := os.ReadFile(path)
	expectNoError(t, err)
	want := "{\n  \"read\": " + strconv.Itoa(current["read"]) +
		",\n  \"write\": " + strconv.Itoa(current["write"]) +
		",\n  \"readwrite\": " + strconv.Itoa(current["readwrite"]) + "\n}\n"
	expectEqual(t, string(written), want)

	readBack, err := ReadToolSchemaBudgetFile(path)
	expectNoError(t, err)
	for _, row := range CompareToolSchemaBudget(current, readBack) {
		expectEqual(t, row.Verdict, "ok")
		expectEqual(t, *row.Delta, 0)
	}

	// 書き戻し先はソースツリー上の tool_schema_budget.json（埋め込み元）
	expectEqual(t, filepath.Base(ToolSchemaBudgetPath()), "tool_schema_budget.json")
	if _, err := os.Stat(ToolSchemaBudgetPath()); err != nil {
		t.Fatalf("ToolSchemaBudgetPath() がソースツリーの予算ファイルを指していない: %v", err)
	}

	// 壊れたファイルはエラー（黙って空の予算にしない）
	expectNoError(t, os.WriteFile(path, []byte("{broken"), 0o644))
	_, err = ReadToolSchemaBudgetFile(path)
	expectErrorContains(t, err, "tool_schema_budget.json")
}
