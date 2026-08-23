package api

// rep_types の正準語彙 (find.KyouRepTypes) と、実装の switch 群の対応を機械検査で固定する。
// どちらかに値を足し忘れても go build / go vet は通り、
// 「その rep 種別だけ検索から静かに消える」形で壊れるため、ここでしか気付けない。

import (
	"os"
	"regexp"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// find.KyouRepTypes の全値が RepsOfKyouRepType の switch で拾われ、
// 未知値には空が返ることを、型ごとに1要素入れたフィクスチャで固定する。
// RepsOfKyouRepType は要素のメソッドを呼ばず収集するだけなので、
// 要素は nil で足りる（見るのは件数だけ）。
func TestKyouRepTypesCoversRepsOfKyouRepType(t *testing.T) {
	repositories := &reps.GkillRepositories{
		KmemoReps:        reps.KmemoRepositories{nil},
		KCReps:           reps.KCRepositories{nil},
		URLogReps:        reps.URLogRepositories{nil},
		TimeIsReps:       reps.TimeIsRepositories{nil},
		MiReps:           reps.MiRepositories{nil},
		NlogReps:         reps.NlogRepositories{nil},
		LantanaReps:      reps.LantanaRepositories{nil},
		IDFKyouReps:      reps.IDFKyouRepositories{nil},
		GitCommitLogReps: reps.GitCommitLogRepositories{nil},
		ReKyouReps:       reps.ReKyouRepositories{ReKyouRepositories: []reps.ReKyouRepository{nil}},
		MiReKyouReps:     reps.MiReKyouRepositories{MiReKyouRepositories: []reps.MiReKyouRepository{nil}},
	}

	for _, repType := range find.KyouRepTypes {
		if got := RepsOfKyouRepType(repositories, repType); len(got) == 0 {
			t.Errorf("正準値 %q に対応する case が RepsOfKyouRepType に無い（この種別が rep_types 検索から静かに消える）", repType)
		}
	}
	for _, unknown := range []string{"", "Kmemo", "idf", "plugin", "tag", "text", "notification", "gpslog"} {
		if got := RepsOfKyouRepType(repositories, unknown); len(got) != 0 {
			t.Errorf("未知の値 %q が rep を返した（正準語彙 find.KyouRepTypes と switch がずれている）", unknown)
		}
	}
}

// gkill_dao_manager.go の rep 生成 switch の case 集合が
// find.KyouRepTypes ∪ {tag, text, notification, gpslog} と一致することをソース走査で固定する。
// 新しい rep 種別を片方にだけ足すと、生成されるのに検索できない
// （またはその逆の）種別ができる。
func TestGkillDAOManagerRepTypeSwitchMatchesCanonicalList(t *testing.T) {
	content, err := os.ReadFile("../dao/gkill_dao_manager.go")
	if err != nil {
		t.Fatalf("read gkill_dao_manager.go: %v", err)
	}

	// `switch rep.Type` 節の case "xxx": を拾う（文字列リテラルのcaseのみ）
	caseRe := regexp.MustCompile(`case "([a-z_]+)":`)
	found := map[string]bool{}
	for _, match := range caseRe.FindAllStringSubmatch(string(content), -1) {
		found[match[1]] = true
	}
	if len(found) == 0 {
		t.Fatal("gkill_dao_manager.go から case が1つも拾えなかった（走査の壊れ）")
	}

	want := map[string]bool{}
	for _, repType := range find.KyouRepTypes {
		want[repType] = true
	}
	// Kyou を返さない付随・GPS の4種は生成側にのみ存在する
	for _, attachedType := range []string{"tag", "text", "notification", "gpslog"} {
		want[attachedType] = true
	}

	for repType := range want {
		if !found[repType] {
			t.Errorf("gkill_dao_manager.go の switch に %q が無い（設定に書いても rep が生成されない）", repType)
		}
	}
	for repType := range found {
		if !want[repType] {
			t.Errorf("gkill_dao_manager.go の switch にある %q が正準語彙に無い（find/rep_types.go への追記漏れか、検索できない種別）", repType)
		}
	}
}
