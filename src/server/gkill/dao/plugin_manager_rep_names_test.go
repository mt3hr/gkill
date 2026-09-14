package dao

import (
	"context"
	"errors"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// repNamesStubPlugin は manifest と申告 rep 名だけを持つ PluginRepository のスタブ。
// 埋め込みが nil なので、テストで呼ばない他のメソッドを実装する必要がない。
type repNamesStubPlugin struct {
	reps.PluginRepository
	manifest gkill_plugin.PluginManifest
	repNames []string
	err      error
	// calls は GetRepNames が呼ばれた回数。manifest 名で引けるときは stdio へ行かないことを見る。
	calls int
}

func (s *repNamesStubPlugin) GetManifest() gkill_plugin.PluginManifest { return s.manifest }
func (s *repNamesStubPlugin) GetRepNames(_ context.Context) ([]string, error) {
	s.calls++
	return s.repNames, s.err
}

// GetPluginByRepName が manifest の rep_name でも、get_rep_name で申告した rep 名でも引けること。
//
// zip の Git リポジトリを束ねるプラグインの Kyou はリポジトリ名を rep_name に持ち、
// クライアントはその rep_name で本文（get_plugin_content_html）を取りに来る。
// manifest 名だけで引くと「プラグインが見つかりません」になる。
func TestPluginManagerGetPluginByRepName_ResolvesDeclaredRepNames(t *testing.T) {
	archived := &repNamesStubPlugin{
		manifest: gkill_plugin.PluginManifest{Name: "gkill_plugin_archived_git_commit_log", RepName: "ArchivedGit"},
		repNames: []string{"racoonboard", "ocha"},
	}
	plain := &repNamesStubPlugin{
		manifest: gkill_plugin.PluginManifest{Name: "gkill_plugin_codex", RepName: "Codex"},
		repNames: []string{"Codex"},
	}
	broken := &repNamesStubPlugin{
		manifest: gkill_plugin.PluginManifest{Name: "gkill_plugin_broken", RepName: "Broken"},
		err:      errors.New("plugin is dead"),
	}
	pm := &PluginManager{plugins: []reps.PluginRepository{plain, archived, broken}, discovered: true}
	ctx := context.Background()

	// manifest 名は1段目で引け、申告名を取りに行かない
	if got := pm.GetPluginByRepName(ctx, "Codex"); got != reps.PluginRepository(plain) {
		t.Errorf("manifest 名で引けない: %v", got)
	}
	if plain.calls != 0 || archived.calls != 0 {
		t.Errorf("manifest 名で引けるのに申告名を取りに行った: plain=%d archived=%d", plain.calls, archived.calls)
	}

	// 申告名は2段目で引ける（失敗するプラグインが混ざっていても止まらない）
	if got := pm.GetPluginByRepName(ctx, "ocha"); got != reps.PluginRepository(archived) {
		t.Errorf("申告名で引けない: %v", got)
	}

	// どこにも無い名前は nil
	if got := pm.GetPluginByRepName(ctx, "nowhere"); got != nil {
		t.Errorf("存在しない名前で引けてしまった: %v", got)
	}
}
