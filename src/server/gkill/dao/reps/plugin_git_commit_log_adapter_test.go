package reps

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
)

// gitIndexSource は provides に git_commit_log を書いたプラグインの索引の材料。
// 申告 rep 名（リポジトリ名）を持ち、Kyou ごとに別の rep_name を返す。
type gitIndexSource struct {
	kyous []gkill_plugin.PluginKyou
}

func (s *gitIndexSource) indexRepName() string { return "ArchivedGit" }
func (s *gitIndexSource) indexIsDeclaredRepName(name string) bool {
	return name == "racoonboard" || name == "ocha"
}
func (s *gitIndexSource) indexPluginName() string { return "gkill_plugin_archived_git_commit_log" }
func (s *gitIndexSource) indexProvidedKinds() map[gkill_plugin.PluginProvidedKind]struct{} {
	return map[gkill_plugin.PluginProvidedKind]struct{}{gkill_plugin.PluginProvidesGitCommitLog: {}}
}
func (s *gitIndexSource) indexFetchAll(_ context.Context) ([]gkill_plugin.PluginKyou, error) {
	return s.kyous, nil
}

// gitStubPlugin はアダプタが manifest と申告名を引くためだけの PluginRepository スタブ。
type gitStubPlugin struct {
	PluginRepository
	index *PluginTypedIndex
}

func (s *gitStubPlugin) GetManifest() gkill_plugin.PluginManifest {
	return gkill_plugin.PluginManifest{
		Name:     "gkill_plugin_archived_git_commit_log",
		RepName:  "ArchivedGit",
		DataType: "git_commit_log",
		Provides: []gkill_plugin.PluginProvidedKind{gkill_plugin.PluginProvidesGitCommitLog},
	}
}
func (s *gitStubPlugin) GetRepNames(_ context.Context) ([]string, error) {
	return []string{"racoonboard", "ocha"}, nil
}
func (s *gitStubPlugin) TypedIndex() *PluginTypedIndex { return s.index }

// gitCommitKyou は native の git rep と同じ形（ID=ハッシュ・時刻=コミッタ日時・author=CreateUser）の PluginKyou。
func gitCommitKyou(hash string, repName string, committedAt time.Time, message string, addition int, deletion int) gkill_plugin.PluginKyou {
	return gkill_plugin.PluginKyou{
		ID: hash, RepName: repName, DataType: "git_commit_log",
		RelatedTime: committedAt, CreateTime: committedAt, UpdateTime: committedAt,
		CreateApp: "git", UpdateApp: "git", CreateUser: "author", UpdateUser: "author",
		Typed: &gkill_plugin.PluginTypedData{GitCommitLog: &gkill_plugin.PluginGitCommitLog{
			CommitMessage: message, Addition: addition, Deletion: deletion,
		}},
	}
}

func newGitAdapterForTest(t *testing.T, kyous []gkill_plugin.PluginKyou) (*pluginGitCommitLogRepositoryImpl, GitCommitLogRepository) {
	t.Helper()
	index := newPluginTypedIndex(&gitIndexSource{kyous: kyous})
	if err := index.build(context.Background()); err != nil {
		t.Fatalf("index build: %v", err)
	}
	plugin := &gitStubPlugin{index: index}
	adapters := NewPluginTypedRepositories(plugin)
	if adapters.GitCommitLog == nil {
		t.Fatal("provides に git_commit_log があるのにアダプタが作られていない")
	}
	impl, ok := adapters.GitCommitLog.(*pluginGitCommitLogRepositoryImpl)
	if !ok {
		t.Fatalf("アダプタの型が違う: %T", adapters.GitCommitLog)
	}
	return impl, adapters.GitCommitLog
}

var (
	gitTestTimeJST = time.Date(2021, 3, 8, 1, 58, 55, 0, time.FixedZone("JST", 9*3600))
	gitTestTimeUTC = time.Date(2020, 10, 12, 9, 5, 23, 0, time.UTC)
)

func gitTestKyous() []gkill_plugin.PluginKyou {
	return []gkill_plugin.PluginKyou{
		gitCommitKyou("2aa9a829e303ad135ca6c32575a8ab678c9eb268", "racoonboard", gitTestTimeJST, "to github\n", 12, 3),
		gitCommitKyou("b66fb9148bb13ba73a1271c1de4e5431ef448acc", "racoonboard", gitTestTimeUTC, "Initial commit", 1, 0),
		gitCommitKyou("747bb1c24add3fea53c835be9f8626c7ec97ebed", "ocha", gitTestTimeJST.Add(time.Hour), "update go mod\n", 0, 870),
	}
}

// provides に git_commit_log を書いたプラグインの記録が、native の GitCommitLog と同じ列で
// 型別リポジトリから引けること。クライアントの get_git_commit_log と MCP の payload はここを通る。
func TestPluginGitCommitLogAdapter_GetGitCommitLog(t *testing.T) {
	_, rep := newGitAdapterForTest(t, gitTestKyous())
	ctx := context.Background()

	got, err := rep.GetGitCommitLog(ctx, "2aa9a829e303ad135ca6c32575a8ab678c9eb268", nil)
	if err != nil {
		t.Fatalf("GetGitCommitLog: %v", err)
	}
	if got == nil {
		t.Fatal("GetGitCommitLog = nil, want 1件")
	}
	if got.RepName != "racoonboard" {
		t.Errorf("RepName = %q, want racoonboard（Kyou ごとの rep 名を保つ）", got.RepName)
	}
	if got.DataType != "git_commit_log" {
		t.Errorf("DataType = %q, want git_commit_log", got.DataType)
	}
	if got.CommitMessage != "to github\n" || got.Addition != 12 || got.Deletion != 3 {
		t.Errorf("列が写っていない: message=%q +%d -%d", got.CommitMessage, got.Addition, got.Deletion)
	}
	if !got.RelatedTime.Equal(gitTestTimeJST) || !got.UpdateTime.Equal(gitTestTimeJST) {
		t.Errorf("時刻が写っていない: related=%v update=%v", got.RelatedTime, got.UpdateTime)
	}
	if got.CreateUser != "author" {
		t.Errorf("CreateUser = %q, want author", got.CreateUser)
	}

	// updateTime はコミッタ日時と秒一致で絞る
	if found, _ := rep.GetGitCommitLog(ctx, "2aa9a829e303ad135ca6c32575a8ab678c9eb268", &gitTestTimeJST); found == nil {
		t.Error("同じ updateTime で引けない")
	}
	other := gitTestTimeJST.Add(time.Minute)
	if found, _ := rep.GetGitCommitLog(ctx, "2aa9a829e303ad135ca6c32575a8ab678c9eb268", &other); found != nil {
		t.Error("違う updateTime で引けてしまう")
	}
	if found, _ := rep.GetGitCommitLog(ctx, "0000000000000000000000000000000000000000", nil); found != nil {
		t.Error("無いハッシュで引けてしまう")
	}
}

// FindGitCommitLog / FindGitCommitLogByIDs が期間・ID・ワードで絞れること。
// ワードの対象列は native と同じくコミットメッセージだけ（rep 名や author では当たらない）。
func TestPluginGitCommitLogAdapter_FindGitCommitLog(t *testing.T) {
	_, rep := newGitAdapterForTest(t, gitTestKyous())
	ctx := context.Background()

	all, err := rep.FindGitCommitLog(ctx, &find.FindQuery{})
	if err != nil {
		t.Fatalf("FindGitCommitLog: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("FindGitCommitLog(条件なし) = %d件, want 3", len(all))
	}

	hashes := func(logs []GitCommitLog) map[string]bool {
		set := map[string]bool{}
		for _, log := range logs {
			set[log.ID] = true
		}
		return set
	}

	byWord, _ := rep.FindGitCommitLog(ctx, &find.FindQuery{Words: []string{"github"}})
	if got := hashes(byWord); len(got) != 1 || !got["2aa9a829e303ad135ca6c32575a8ab678c9eb268"] {
		t.Errorf("FindGitCommitLog(github) = %v, want メッセージ一致の1件", got)
	}
	byRepName, _ := rep.FindGitCommitLog(ctx, &find.FindQuery{Words: []string{"racoonboard"}})
	if len(byRepName) != 0 {
		t.Errorf("rep 名で当たってはいけない（native はメッセージだけ）: %d件", len(byRepName))
	}
	byNotWord, _ := rep.FindGitCommitLog(ctx, &find.FindQuery{Words: []string{}, NotWords: []string{"github"}})
	if got := hashes(byNotWord); len(got) != 2 || got["2aa9a829e303ad135ca6c32575a8ab678c9eb268"] {
		t.Errorf("FindGitCommitLog(-github) = %v, want github 以外の2件", got)
	}
	byIDPrefix, _ := rep.FindGitCommitLog(ctx, &find.FindQuery{Words: []string{"b66fb914"}})
	if got := hashes(byIDPrefix); len(got) != 1 || !got["b66fb9148bb13ba73a1271c1de4e5431ef448acc"] {
		t.Errorf("FindGitCommitLog(ハッシュ前方一致) = %v, want 1件", got)
	}

	start := gitTestTimeJST.Add(30 * time.Minute)
	end := gitTestTimeJST.Add(2 * time.Hour)
	byCalendar, _ := rep.FindGitCommitLog(ctx, &find.FindQuery{CalendarStartDate: &start, CalendarEndDate: &end})
	if got := hashes(byCalendar); len(got) != 1 || !got["747bb1c24add3fea53c835be9f8626c7ec97ebed"] {
		t.Errorf("FindGitCommitLog(期間) = %v, want ocha の1件", got)
	}

	none, err := rep.FindGitCommitLog(ctx, &find.FindQuery{Words: []string{"no-such-word"}})
	if err != nil || none != nil {
		t.Errorf("0件は (nil, nil) を返す契約: got %v, %v", none, err)
	}

	byIDs, _ := rep.FindGitCommitLogByIDs(ctx, []string{"747bb1c24add3fea53c835be9f8626c7ec97ebed", "0000000000000000000000000000000000000000"})
	if got := hashes(byIDs); len(got) != 1 || !got["747bb1c24add3fea53c835be9f8626c7ec97ebed"] {
		t.Errorf("FindGitCommitLogByIDs = %v, want 見つかった1件だけ（無い id は黙って落ちる）", got)
	}
	if empty, _ := rep.FindGitCommitLogByIDs(ctx, nil); empty != nil {
		t.Errorf("FindGitCommitLogByIDs(空) は (nil, nil): got %v", empty)
	}
}

// FindKyous（型別アダプタ経由の Kyou 検索）が Kyou ごとの rep 名を保ち、ワードで絞れること。
// rep_types=["git_commit_log"] の検索はこの経路で来る。
func TestPluginGitCommitLogAdapter_FindKyous(t *testing.T) {
	_, rep := newGitAdapterForTest(t, gitTestKyous())
	ctx := context.Background()

	got, err := rep.FindKyous(ctx, &find.FindQuery{Words: []string{"update"}})
	if err != nil {
		t.Fatalf("FindKyous: %v", err)
	}
	kyous := got["ArchivedGit"]
	if len(kyous) != 1 {
		t.Fatalf("FindKyous(update) = %d件, want 1 (%v)", len(kyous), got)
	}
	if kyous[0].ID != "747bb1c24add3fea53c835be9f8626c7ec97ebed" || kyous[0].RepName != "ocha" || kyous[0].DataType != "git_commit_log" {
		t.Errorf("Kyou = %+v, want ocha のコミット", kyous[0])
	}

	// 申告済みの rep 名も GetRepNames で引ける（Step4 の rep 名照合が使う）
	names, err := RepNamesOf(ctx, rep)
	if err != nil || len(names) != 2 {
		t.Errorf("RepNamesOf(adapter) = %v, %v, want 申告名2つ", names, err)
	}
}

// GitCommitLog を含む PluginTypedData が2つ以上入っていたら、先に並ぶ種別を採用して警告すること
// （既存の採用順の末尾に GitCommitLog が付く）。
func TestPluginTypedIndex_GitCommitLogIsLastInPrecedence(t *testing.T) {
	kyou := gitCommitKyou("2aa9a829e303ad135ca6c32575a8ab678c9eb268", "racoonboard", gitTestTimeJST, "msg", 1, 1)
	kyou.Typed.Kmemo = &gkill_plugin.PluginKmemo{Content: "memo"}
	source := &gitIndexSource{kyous: []gkill_plugin.PluginKyou{kyou}}
	index := newPluginTypedIndex(source)
	if err := index.build(context.Background()); err != nil {
		t.Fatalf("index build: %v", err)
	}
	record := index.Snapshot().byID[kyou.ID]
	if record == nil {
		t.Fatal("レコードが無い")
	}
	// provides に kmemo が無いので Kmemo は捨てられ、GitCommitLog も「2つ目」として警告扱いになる
	if record.GitCommitLog != nil {
		t.Error("採用順で先に Kmemo が来るので GitCommitLog は採用されない")
	}
	if record.Kmemo != nil {
		t.Error("provides に無い Kmemo が採用されている")
	}
}
