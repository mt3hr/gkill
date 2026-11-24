package dao

// manifest.json の data_type と provides の食い違いの警告（plugin_manager.go の loadManifest）。
//
// 1本のプラグインが複数の型別の種類を返してよい（ADR-0317）。manifest の data_type はその代表の1種類なので、
// 「代表が provides のどの種類の接頭辞とも合わない」ときだけ警告する。種類ごとに見ると代表以外が必ず
// 食い違い、走査のたびに警告し続ける。

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifest_WarnsOnlyWhenDataTypeMatchesNoTypedKind(t *testing.T) {
	const warning = "data_type does not match any of them"
	cases := []struct {
		name     string
		dataType string
		provides []string
		warn     bool
	}{
		{"代表が種類の1つに合う（複数の種類）", "timeis_start", []string{"timeis", "urlog", "kmemo"}, false},
		{"代表が種類の1つに合う（1種類）", "kc", []string{"kc"}, false},
		{"代表がどの種類にも合わない", "vrchat_world", []string{"timeis", "urlog"}, true},
		{"型別の種類を宣言していない", "claude_conversation", nil, false},
		{"GPS だけ（型別の種類ではない）", "gpslog_only", []string{"gpslog"}, false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			logBuffer := &bytes.Buffer{}
			originalLogger := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(logBuffer, &slog.HandlerOptions{Level: slog.LevelDebug})))
			t.Cleanup(func() { slog.SetDefault(originalLogger) })

			pluginDir := t.TempDir()
			manifest := map[string]any{
				"protocol_version": "1",
				"name":             "manifest_test_plugin",
				"version":          "1.0.0",
				"rep_name":         "ManifestTest",
				"data_type":        testCase.dataType,
				"executable":       "no_such_binary",
			}
			if testCase.provides != nil {
				manifest["provides"] = testCase.provides
			}
			data, err := json.Marshal(manifest)
			if err != nil {
				t.Fatalf("marshal manifest: %v", err)
			}
			if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), data, 0o600); err != nil {
				t.Fatalf("write manifest: %v", err)
			}

			if _, err := (&PluginManager{}).loadManifest(pluginDir); err != nil {
				t.Fatalf("loadManifest: %v", err)
			}
			if got := strings.Contains(logBuffer.String(), warning); got != testCase.warn {
				t.Errorf("warn = %v, want %v（log: %s）", got, testCase.warn, logBuffer.String())
			}
		})
	}
}
