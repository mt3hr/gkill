package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// runOnce は構築が失敗したら ERROR 行を stderr（last_error）へ出して戻り、完了の Info は出さない。
// 成功したら ERROR 行は出ない。stdout には何も書かない（プロトコルのチャネル）。
func TestRunOnceLogsBuildFailureAndSkipsDoneOnError(t *testing.T) {
	original := buildCacheFn
	t.Cleanup(func() { buildCacheFn = original })
	var logs bytes.Buffer
	setPluginLogWriterForTest(t, &logs)

	buildCacheFn = func(string, pluginConfig) error { return errors.New("boom") }
	globalBuilder.runOnce(t.TempDir(), pluginConfig{})
	if !strings.Contains(logs.String(), "ERROR: gkill_plugin_fitbit: build error: boom") {
		t.Errorf("失敗の ERROR 行が無い: %q", logs.String())
	}

	logs.Reset()
	buildCacheFn = func(string, pluginConfig) error { return nil }
	globalBuilder.runOnce(t.TempDir(), pluginConfig{})
	if strings.Contains(logs.String(), "ERROR:") {
		t.Errorf("成功なのに ERROR 行が出ている: %q", logs.String())
	}
}
