package mcp

// tools/list のバイト量の予算（旧 tool-schema-budget.mjs / .json）。
//
// tools/list は AI セッションごと・接続ごとに丸ごと送られる。description を1文足すたびに
// 全利用者が毎回そのバイト数を払うので、「現状の実測」を予算ファイルに記録し、
// 増減があれば `gkill_server mcp schema-budget --update` で明示的に更新する。
// 判定と文言はテスト（tool_schema_budget_test.go）とサブコマンドで共用する。

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

//go:embed tool_schema_budget.json
var toolSchemaBudgetJSON []byte

// ToolSchemaBudgetSlackBytes は「減った」と判定するまでの余裕。これ以下の減少は ok。
const ToolSchemaBudgetSlackBytes = 1024

// ServerKinds は予算表の行（順序固定）。
var ServerKinds = []string{"read", "write", "readwrite"}

// BudgetRow は1行の判定。Budget / Delta は予算が無いとき nil。
type BudgetRow struct {
	Kind    string
	Current int
	Budget  *int
	Delta   *int
	Verdict string
}

// MeasureToolSchemaBytes は3サーバの tools/list（schema_revision 焼き込み済み）の UTF-8 バイト数。
func MeasureToolSchemaBytes() map[string]int {
	out := map[string]int{}
	for _, kind := range ServerKinds {
		server := NewServerForKind(kind, nil, nil)
		out[kind] = jsonobj.ByteLength(toolsToAny(server.Tools))
	}
	return out
}

func toolsToAny(tools []*jsonobj.Object) []any {
	out := make([]any, len(tools))
	for i, tool := range tools {
		out[i] = tool
	}
	return out
}

// ReadToolSchemaBudget はバイナリに埋め込んだ予算を返す。
func ReadToolSchemaBudget() (map[string]int, error) {
	return parseToolSchemaBudget(toolSchemaBudgetJSON)
}

// ReadToolSchemaBudgetFile はファイルから予算を読む（--update の前の表示用）。
func ReadToolSchemaBudgetFile(path string) (map[string]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseToolSchemaBudget(data)
}

func parseToolSchemaBudget(data []byte) (map[string]int, error) {
	budget := map[string]int{}
	if err := json.Unmarshal(data, &budget); err != nil {
		return nil, fmt.Errorf("tool_schema_budget.json: %w", err)
	}
	return budget, nil
}

// ToolSchemaBudgetPath はソースツリー上の予算ファイル（--update の書き込み先）。
// ビルド時のソースパスから引くので、リポジトリの checkout 上で動かすときだけ意味を持つ。
func ToolSchemaBudgetPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "tool_schema_budget.json"
	}
	return filepath.Join(filepath.Dir(file), "tool_schema_budget.json")
}

// WriteToolSchemaBudget は予算を JSON（2スペース・末尾改行・行は ServerKinds の順）で書く。
func WriteToolSchemaBudget(budget map[string]int, path string) error {
	o := jsonobj.New()
	for _, kind := range ServerKinds {
		o.Set(kind, budget[kind])
	}
	return os.WriteFile(path, []byte(jsonobj.MarshalIndentString(o, "  ")+"\n"), 0o644)
}

// CompareToolSchemaBudget は実測と予算を突き合わせる。budget が nil でも動く（全行 missing）。
func CompareToolSchemaBudget(current, budget map[string]int) []BudgetRow {
	rows := []BudgetRow{}
	for _, kind := range ServerKinds {
		measured := current[kind]
		allowed, ok := budget[kind]
		if !ok {
			rows = append(rows, BudgetRow{Kind: kind, Current: measured, Verdict: "missing"})
			continue
		}
		delta := measured - allowed
		verdict := "ok"
		if delta > 0 {
			verdict = "over"
		} else if -delta > ToolSchemaBudgetSlackBytes {
			verdict = "under"
		}
		allowedCopy, deltaCopy := allowed, delta
		rows = append(rows, BudgetRow{Kind: kind, Current: measured, Budget: &allowedCopy, Delta: &deltaCopy, Verdict: verdict})
	}
	return rows
}

// DescribeBudgetRow は1行の判定を人が読む文にする（テストの失敗文とサブコマンドの表示で共用）。
func DescribeBudgetRow(row BudgetRow) string {
	sign := ""
	if row.Delta != nil && *row.Delta >= 0 {
		sign = "+"
	}
	switch row.Verdict {
	case "over":
		return fmt.Sprintf("%s: tools/list is %d bytes, %s%d over the budget of %d. ", row.Kind, row.Current, sign, *row.Delta, *row.Budget) +
			"Every byte here is paid by every AI session on every request. If the growth is intended, run " +
			"`gkill_server mcp schema-budget --update` and say why in the commit message; otherwise trim the description."
	case "under":
		return fmt.Sprintf("%s: tools/list is %d bytes, %s%d under the budget of %d ", row.Kind, row.Current, sign, *row.Delta, *row.Budget) +
			fmt.Sprintf("(more than %d bytes of slack). Run `gkill_server mcp schema-budget --update` ", ToolSchemaBudgetSlackBytes) +
			"so the budget follows the reduction instead of quietly absorbing the next increase."
	case "missing":
		return fmt.Sprintf("%s: no budget recorded (current %d bytes). Run `gkill_server mcp schema-budget --update`.", row.Kind, row.Current)
	default:
		return fmt.Sprintf("%s: %d bytes (budget %d, %s%d).", row.Kind, row.Current, *row.Budget, sign, *row.Delta)
	}
}
