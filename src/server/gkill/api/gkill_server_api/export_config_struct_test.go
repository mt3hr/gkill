package gkill_server_api

// 焼き込む木からチェックされていないノードを取り除く。
//
// **表示を止めるだけでは足りない。** 木には出力に含めなかったタグ名・rep 名まで入っていて、
// ファイルの中に残っていれば読まれる。ここが落ちても画面は普通に出るので、
// 気づけるのはこのテストだけになる。

import (
	"encoding/json"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// structNames は木に残っている名前を深さ優先で集める。
func structNames(t *testing.T, raw *json.RawMessage, nameKey string) []string {
	t.Helper()
	if raw == nil {
		return nil
	}
	var root map[string]any
	if err := json.Unmarshal(*raw, &root); err != nil {
		t.Fatalf("unmarshal struct: %v", err)
	}
	names := []string{}
	var walk func(node map[string]any)
	walk = func(node map[string]any) {
		if name, ok := node[nameKey].(string); ok && name != "" {
			names = append(names, name)
		}
		children, ok := node["children"].([]any)
		if !ok {
			return
		}
		for _, child := range children {
			if childNode, ok := child.(map[string]any); ok {
				walk(childNode)
			}
		}
	}
	walk(root)
	return names
}

func rawJSON(t *testing.T, value any) *json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := json.RawMessage(encoded)
	return &raw
}

// TestPruneExportApplicationConfigStructs_RemovesUncheckedNodes は、
// チェックされていなかったタグが木から消えることを確認する。
// 入れ物（is_dir）は、中身が1つも残らなければ一緒に消える。
func TestPruneExportApplicationConfigStructs_RemovesUncheckedNodes(t *testing.T) {
	config := &user_config.ApplicationConfig{
		TagStruct: rawJSON(t, map[string]any{
			"key": "__root__", "is_dir": true,
			"children": []any{
				map[string]any{"tag_name": "TagA", "is_dir": false},
				map[string]any{"tag_name": "TagB", "is_dir": false},
				map[string]any{
					"name": "Folder", "is_dir": true,
					"children": []any{
						map[string]any{"tag_name": "TagC", "is_dir": false},
					},
				},
				map[string]any{
					"name": "EmptyFolder", "is_dir": true,
					"children": []any{
						map[string]any{"tag_name": "TagD", "is_dir": false},
					},
				},
			},
		}),
	}

	// チェックされていたのは TagA と TagC だけ
	columns := []find.FindQuery{{Tags: []string{"TagA", "TagC"}}}
	if err := pruneExportApplicationConfigStructs(config, collectExportAllowedNames(columns)); err != nil {
		t.Fatalf("prune: %v", err)
	}

	names := structNames(t, config.TagStruct, "tag_name")
	want := map[string]bool{"TagA": true, "TagC": true}
	for _, name := range names {
		if !want[name] {
			t.Errorf("チェックされていないタグが残っている: %s", name)
		}
		delete(want, name)
	}
	for name := range want {
		t.Errorf("チェックされていたタグが消えている: %s", name)
	}

	// 中身が残らない入れ物も消えること
	encoded, err := json.Marshal(config.TagStruct)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(encoded) == "" {
		t.Fatal("木が空になっている")
	}
	var root map[string]any
	if err := json.Unmarshal(*config.TagStruct, &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	children, _ := root["children"].([]any)
	for _, child := range children {
		node, _ := child.(map[string]any)
		if name, _ := node["name"].(string); name == "EmptyFolder" {
			t.Error("中身が残らない入れ物が消えていない")
		}
	}
}

// TestPruneExportApplicationConfigStructs_DropsParentWithNoCheckedChild は、
// 子が1つも残らない親が、**親自身がチェックされていても**消えることを確認する。
// 空の入れ物だけが残っても意味が無く、入れ物の名前そのものが
// 出力に含めなかった区分を伝えてしまう。
func TestPruneExportApplicationConfigStructs_DropsParentWithNoCheckedChild(t *testing.T) {
	config := &user_config.ApplicationConfig{
		TagStruct: rawJSON(t, map[string]any{
			"key": "__root__", "is_dir": true,
			"children": []any{
				map[string]any{"tag_name": "TagA", "is_dir": false},
				// 親自身はチェックされているが、子は1つもチェックされていない
				map[string]any{
					"tag_name": "ParentTag", "name": "ParentTag", "is_dir": true,
					"children": []any{
						map[string]any{"tag_name": "TagX", "is_dir": false},
						map[string]any{"tag_name": "TagY", "is_dir": false},
					},
				},
			},
		}),
	}

	columns := []find.FindQuery{{Tags: []string{"TagA", "ParentTag"}}}
	if err := pruneExportApplicationConfigStructs(config, collectExportAllowedNames(columns)); err != nil {
		t.Fatalf("prune: %v", err)
	}

	names := structNames(t, config.TagStruct, "tag_name")
	for _, name := range names {
		if name == "ParentTag" {
			t.Error("子が1つも残らない親が消えていない")
		}
		if name == "TagX" || name == "TagY" {
			t.Errorf("チェックされていない子が残っている: %s", name)
		}
	}
	found := false
	for _, name := range names {
		if name == "TagA" {
			found = true
		}
	}
	if !found {
		t.Error("チェックされていた葉が消えている")
	}
}

// TestPruneExportApplicationConfigStructs_KeepsAllWhenNotFiltered は、
// 絞っていない列（nil）が1つでもあれば木を刈らないことを確認する。
// nil は「フィルタ未使用」で、ここで削ると出力した本人の意図より狭くなる。
func TestPruneExportApplicationConfigStructs_KeepsAllWhenNotFiltered(t *testing.T) {
	config := &user_config.ApplicationConfig{
		TagStruct: rawJSON(t, map[string]any{
			"key": "__root__", "is_dir": true,
			"children": []any{
				map[string]any{"tag_name": "TagA", "is_dir": false},
				map[string]any{"tag_name": "TagB", "is_dir": false},
			},
		}),
	}

	columns := []find.FindQuery{{Tags: []string{"TagA"}}, {Tags: nil}}
	if err := pruneExportApplicationConfigStructs(config, collectExportAllowedNames(columns)); err != nil {
		t.Fatalf("prune: %v", err)
	}

	names := structNames(t, config.TagStruct, "tag_name")
	if len(names) != 2 {
		t.Errorf("絞らない列があるのに刈られている: %v", names)
	}
}

// TestCollectExportAllowedNames_DerivesDevicesFromReps は、端末が rep 名から導かれることを
// 確認する。検索条件に端末の欄が無いので、クライアントと同じ規則
// （`<種別>_<端末>_<日付>` の真ん中）で導く。ずれると端末の木だけ空になる。
func TestCollectExportAllowedNames_DerivesDevicesFromReps(t *testing.T) {
	allowed := collectExportAllowedNames([]find.FindQuery{{
		Reps: []string{"Box_TestPC_20240101", "Kmemo_TestPC_20240101", "NotARepName"},
		Tags: []string{},
	}})
	if allowed.Devices == nil {
		t.Fatal("端末が絞られていない")
	}
	if !allowed.Devices["TestPC"] {
		t.Error("rep 名から端末が導けていない")
	}
	if !allowed.Devices["なし"] {
		t.Error("端末を導けない rep のための「なし」が残っていない")
	}
}

// TestCollectExportAllowedNames_DerivesRepTypesFromReps は、記録種別が rep 名から
// 導かれることを確認する。**サイドバーは RepTypes を使っていない**
// （チェックはクライアント専用の rep_types_in_sidebar に入り、絞り込みは reps で効く）ので、
// RepTypes だけを見ていると記録種別の木が刈られず、外したはずの種別が残る。
func TestCollectExportAllowedNames_DerivesRepTypesFromReps(t *testing.T) {
	allowed := collectExportAllowedNames([]find.FindQuery{{
		Reps: []string{"Box_TestPC_20240101", "Kmemo_TestPC_20240101", "NotARepName"},
		Tags: []string{},
	}})
	if allowed.RepTypes == nil {
		t.Fatal("記録種別が絞られていない")
	}
	if !allowed.RepTypes["Box"] || !allowed.RepTypes["Kmemo"] {
		t.Errorf("rep 名から種別が導けていない: %v", allowed.RepTypes)
	}
	// 3つに割れない rep 名は、rep 名そのものが種別（クライアントと同じ規則）
	if !allowed.RepTypes["NotARepName"] {
		t.Error("3つに割れない rep 名が種別として扱われていない")
	}
}

// TestCollectExportAllowedNames_EmptyIsZeroHit は、非 nil の空配列が「0件指定」として
// 扱われることを確認する。「絞らない」と取り違えると、外したはずの名前が全部残る。
func TestCollectExportAllowedNames_EmptyIsZeroHit(t *testing.T) {
	allowed := collectExportAllowedNames([]find.FindQuery{{Tags: []string{}}})
	if allowed.Tags == nil {
		t.Fatal("非 nil の空配列が「絞らない」になっている")
	}
	if len(allowed.Tags) != 0 {
		t.Errorf("空であるべき: %v", allowed.Tags)
	}
}

// TestCollectExportAllowedMiBoards は、mi の出力で焼き込んでよい板名の集め方を確かめる。
//
// mi の列は1本につき板1つ（MiBoardName）で、nil が「すべて」を意味する。
// nil の列が1本でもあれば、その列は板をまたいで全部を出すと決めた列なので、
// 木を刃ると本人の意図より狭くなる。
func TestCollectExportAllowedMiBoards(t *testing.T) {
	boardA := "BoardA"
	boardB := "BoardB"

	t.Run("列の板名を集める（「すべて」の番兵は常に残す）", func(t *testing.T) {
		boards := collectExportAllowedMiBoards([]find.FindQuery{
			{MiBoardName: &boardA},
			{MiBoardName: &boardB},
		})
		if boards == nil {
			t.Fatal("綞らないになっている")
		}
		for _, want := range []string{boardA, boardB, exportMiAllBoardKey} {
			if !boards[want] {
				t.Errorf("%q が許可集合に無い: %v", want, boards)
			}
		}
	})

	t.Run("nil の列（すべて）が1本でもあれば綞らない", func(t *testing.T) {
		boards := collectExportAllowedMiBoards([]find.FindQuery{
			{MiBoardName: &boardA},
			{MiBoardName: nil},
		})
		if boards != nil {
			t.Errorf("「すべて」の列があるのに刃ろうとしている: %v", boards)
		}
	})
}

// TestPruneExportApplicationConfigStructs_PrunesMiBoardStruct は、出力に含めなかった
// 板名が木から消えることを確かめる。
//
// 板名は利用者が付けた名前で、それ自身が知られたくない情報になりうる。
// 表示を止めるだけでは足らない —— ファイルの中に残っていれば読まれる。
func TestPruneExportApplicationConfigStructs_PrunesMiBoardStruct(t *testing.T) {
	config := &user_config.ApplicationConfig{
		MiBoardStruct: rawJSON(t, map[string]any{
			"key": "__root__", "is_dir": true, "children": []any{
				map[string]any{"key": exportMiAllBoardKey, "board_name": exportMiAllBoardKey, "children": nil},
				map[string]any{"key": "BoardA", "board_name": "BoardA", "children": nil},
				map[string]any{"key": "BoardSecret", "board_name": "BoardSecret", "children": nil},
			},
		}),
	}

	allowed := &exportAllowedNames{MiBoards: map[string]bool{
		"BoardA": true, exportMiAllBoardKey: true,
	}}
	if err := pruneExportApplicationConfigStructs(config, allowed); err != nil {
		t.Fatalf("prune: %v", err)
	}

	names := structNames(t, config.MiBoardStruct, exportStructMiBoardNameKey)
	for _, want := range []string{"BoardA", exportMiAllBoardKey} {
		found := false
		for _, name := range names {
			if name == want {
				found = true
			}
		}
		if !found {
			t.Errorf("出力した板 %q が木から消えている: %v", want, names)
		}
	}
	for _, name := range names {
		if name == "BoardSecret" {
			t.Errorf("出力に含めなかった板名が木に残っている: %v", names)
		}
	}
}

// TestPruneExportApplicationConfigStructs_KeepsMiBoardStructWhenNotFiltered は、
// rykv の出力（MiBoards が nil）では板ツリーを触らないことを確かめる。
func TestPruneExportApplicationConfigStructs_KeepsMiBoardStructWhenNotFiltered(t *testing.T) {
	config := &user_config.ApplicationConfig{
		MiBoardStruct: rawJSON(t, map[string]any{
			"key": "__root__", "is_dir": true, "children": []any{
				map[string]any{"key": "BoardA", "board_name": "BoardA", "children": nil},
			},
		}),
	}

	if err := pruneExportApplicationConfigStructs(config, &exportAllowedNames{}); err != nil {
		t.Fatalf("prune: %v", err)
	}
	names := structNames(t, config.MiBoardStruct, exportStructMiBoardNameKey)
	if len(names) != 1 || names[0] != "BoardA" {
		t.Errorf("綞らないはずの板ツリーを刃っている: %v", names)
	}
}
