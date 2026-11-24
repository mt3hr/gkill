package gkill_server_api

// 「HTML で出力」したファイルへ焼き込む ApplicationConfig の木（タグ・記録保管場所・
// 端末・記録種別）から、**チェックされていなかったノードを取り除く**。
//
// 木は利用者が設定画面で並べたもので、出力に含めなかったタグ名・rep 名まで入っている。
// そのまま焼くと、渡した相手に「ほかにこういうタグがある」と伝わってしまう。
// **表示を止めるだけでは足りない** —— ファイルの中に残っていれば読まれる。

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// exportStructNameKeys は木ごとの「葉が持つ名前の欄」。
// クライアントのデータクラス（tag-struct-element-data.ts ほか）と対。
const (
	exportStructTagNameKey     = "tag_name"
	exportStructRepNameKey     = "rep_name"
	exportStructDeviceNameKey  = "device_name"
	exportStructRepTypeNameKey = "rep_type_name"
	exportStructMiBoardNameKey = "board_name"
)

// exportMiAllBoardKey は板ツリーの「すべて」の番兵。
// クライアントの MI_ALL_BOARD_KEY（mi-board-names.ts）と同じ値であること ——
// ずれると、出力した mi のサイドバーから「すべて」の列を選べなくなる。
// ロケール非依存の固定値で、表示名（name）だけが訳語になる。
const exportMiAllBoardKey = "すべて"

// exportAllowedNames は焼き込んでよい名前の集合。
// nil は「絞らない」（その列は全部を出すと決めた列なので、ここで削ると本人の意図より狭くなる）。
type exportAllowedNames struct {
	Tags     map[string]bool
	Reps     map[string]bool
	RepTypes map[string]bool
	Devices  map[string]bool
	// MiBoards は mi の板名。列の MiBoardName から集める（1列1板）。
	// **nil のときは絞らない** —— mi_board_name が nil の列（「すべて」）が
	// 1本でもあれば、その列は板をまたいで全部を出すと決めた列なので、
	// 木を刈ると本人の意図より狭くなる。rykv の出力では常に nil（板ツリーを使わない）。
	MiBoards map[string]bool
}

// collectExportAllowedNames は列の検索条件から、チェックされていた名前を集める。
//
// **非 nil の空配列は「0件指定」**なので、空の集合としてそのまま扱う（絞らないではない）。
// 1列でも nil なら、その欄は絞らない。
func collectExportAllowedNames(columns []find.FindQuery) *exportAllowedNames {
	allowed := &exportAllowedNames{
		Tags:     map[string]bool{},
		Reps:     map[string]bool{},
		RepTypes: map[string]bool{},
		Devices:  map[string]bool{},
	}
	for i := range columns {
		if columns[i].Tags == nil {
			allowed.Tags = nil
		}
		if columns[i].Reps == nil {
			allowed.Reps = nil
		}
		if columns[i].RepTypes == nil {
			allowed.RepTypes = nil
		}
	}
	for i := range columns {
		if allowed.Tags != nil {
			for _, tag := range columns[i].Tags {
				allowed.Tags[tag] = true
			}
		}
		if allowed.Reps != nil {
			for _, repName := range columns[i].Reps {
				allowed.Reps[repName] = true
			}
		}
		if allowed.RepTypes != nil {
			for _, repType := range columns[i].RepTypes {
				allowed.RepTypes[repType] = true
			}
		}
	}

	// 端末と記録種別は rep 名から導く。
	//
	// **端末には検索条件の欄が無く、記録種別の欄（RepTypes）はサイドバーが使っていない**
	// （チェックはクライアント専用の rep_types_in_sidebar に入り、実際の絞り込みは reps で効く）。
	// そのため、どちらも許された rep 名から導く。規則はクライアントと同じ
	// （application-config.ts の get_device_from_rep_name / get_rep_type_from_rep_name。
	// rep 名は `<種別>_<端末>_<日付>`）。ずれるとその木だけ空になるか、外したはずの名前が残る。
	if allowed.Reps == nil {
		allowed.Devices = nil
	} else {
		for repName := range allowed.Reps {
			if device := exportDeviceFromRepName(repName); device != "" {
				allowed.Devices[device] = true
			} else {
				// 3つに割れない rep 名は端末を名乗れない。その受け皿が「なし」
				allowed.Devices["なし"] = true
			}
		}
	}

	// 記録種別。RepTypes が明示されていればそれを優先し、無ければ rep 名から導く。
	// （最初のループで RepTypes が nil なら「絞らない」になっているので、
	//  rep から導けるときだけ作り直す）
	if len(allowed.RepTypes) == 0 && allowed.Reps != nil {
		allowed.RepTypes = map[string]bool{}
		for repName := range allowed.Reps {
			if repType := exportRepTypeFromRepName(repName); repType != "" {
				allowed.RepTypes[repType] = true
			}
		}
	}
	return allowed
}

// collectExportAllowedMiBoards は列の検索条件から、焼き込んでよい板名を集める。
//
// mi の列は1本につき板1つ（`mi_board_name`）で、nil が「すべて」を意味する。
// nil の列が1本でもあれば絞らない（nil を返す）。
// 「すべて」の番兵ノードは実在の板ではないので、常に残す。
//
// rykv の出力ではこの関数を呼ばない（板ツリーは使わないので刈る必要が無い）。
func collectExportAllowedMiBoards(columns []find.FindQuery) map[string]bool {
	boards := map[string]bool{}
	for i := range columns {
		if columns[i].MiBoardName == nil {
			return nil
		}
		boards[*columns[i].MiBoardName] = true
	}
	// 番兵は実在の板ではないので、刈られないように入れておく
	boards[exportMiAllBoardKey] = true
	return boards
}

// exportDeviceFromRepName は rep 名から端末名を導く。
// クライアントの get_device_from_rep_name と同じ規則にすること。
func exportDeviceFromRepName(repName string) string {
	splited := strings.Split(repName, "_")
	if len(splited) != 3 {
		return ""
	}
	return splited[1]
}

// exportRepTypeFromRepName は rep 名から記録種別を導く。
// クライアントの get_rep_type_from_rep_name と同じ規則にすること
// （**3つに割れないときは rep 名そのものが種別**。端末と違って null にはならない）。
func exportRepTypeFromRepName(repName string) string {
	splited := strings.Split(repName, "_")
	if len(splited) != 3 {
		return repName
	}
	return splited[0]
}

// pruneExportApplicationConfigStructs は焼き込む前に木を刈る。
//
// 木は *json.RawMessage（設定 DB に入っていた JSON そのまま）なので、
// 一般的な map として辿る。葉は名前が許されているものだけ、
// 入れ物（is_dir）は中身が残ったものだけを残す。根は必ず残す。
func pruneExportApplicationConfigStructs(config *user_config.ApplicationConfig, allowed *exportAllowedNames) error {
	targets := []struct {
		raw     **json.RawMessage
		nameKey string
		names   map[string]bool
	}{
		{&config.TagStruct, exportStructTagNameKey, allowed.Tags},
		{&config.RepStruct, exportStructRepNameKey, allowed.Reps},
		{&config.DeviceStruct, exportStructDeviceNameKey, allowed.Devices},
		{&config.RepTypeStruct, exportStructRepTypeNameKey, allowed.RepTypes},
		// 板ツリー。mi の出力だけ非 nil になる（rykv は板ツリーを使わない）。
		// 刈らないと、出力に含めなかった板名が渡した相手に読まれる
		{&config.MiBoardStruct, exportStructMiBoardNameKey, allowed.MiBoards},
	}
	for _, target := range targets {
		if target.names == nil || *target.raw == nil {
			// 絞らない欄、または木そのものが無い（設定画面を一度も触っていない利用者）
			continue
		}
		pruned, err := pruneExportStructJSON(**target.raw, target.nameKey, target.names)
		if err != nil {
			return fmt.Errorf("error at prune export struct %s: %w", target.nameKey, err)
		}
		*target.raw = &pruned
	}
	return nil
}

// pruneExportStructJSON は木1本を刈って JSON へ戻す。
func pruneExportStructJSON(raw json.RawMessage, nameKey string, names map[string]bool) (json.RawMessage, error) {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("error at parse struct json: %w", err)
	}
	node, ok := root.(map[string]any)
	if !ok {
		// 木として読めない（null など）。触らずそのまま返す
		return raw, nil
	}
	// 根は必ず残す。中身だけ刈る
	node["children"] = pruneExportStructChildren(node["children"], nameKey, names)
	pruned, err := json.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("error at marshal pruned struct: %w", err)
	}
	return json.RawMessage(pruned), nil
}

// pruneExportStructChildren は子の並びを刈る。残るものが無ければ空の配列を返す
// （null にすると、クライアントが「まだ読み込んでいない」と見分けられない）。
func pruneExportStructChildren(children any, nameKey string, names map[string]bool) []any {
	list, ok := children.([]any)
	if !ok {
		return []any{}
	}
	kept := []any{}
	for _, child := range list {
		node, ok := child.(map[string]any)
		if !ok {
			continue
		}
		originalChildren, hasChildren := node["children"].([]any)
		survivors := pruneExportStructChildren(node["children"], nameKey, names)
		name, _ := node[nameKey].(string)
		isDir, _ := node["is_dir"].(bool)

		// **子を持つノードは、残る子が1つも無ければ親ごと消す。**
		// 親自身がチェックされていても消す（中身が空の入れ物だけが残っても意味が無く、
		// 入れ物の名前そのものが出力に含めなかった区分を伝えてしまう）。
		// 入れ物の名前はタグ名の欄に入ることがある（木の編集画面が folder_name を写す）ので、
		// 「名前が許されているか」だけで判断すると空の親が残る
		if hasChildren && len(originalChildren) != 0 {
			if len(survivors) == 0 {
				continue
			}
			node["children"] = survivors
			kept = append(kept, node)
			continue
		}

		// 子を持たない葉。名前が許されているときだけ残す
		if name == "" || !names[name] {
			continue
		}
		if isDir {
			node["children"] = survivors
		}
		kept = append(kept, node)
	}
	return kept
}
