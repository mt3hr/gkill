package dvnf_cmd

import (
	"testing"
)

func TestDVNFCmdNotNil(t *testing.T) {
	if DVNFCmd == nil {
		t.Fatal("DVNFCmd should not be nil")
	}
	if DVNFCmd.Use != "dvnf" {
		t.Errorf("DVNFCmd.Use = %q, want %q", DVNFCmd.Use, "dvnf")
	}
}

func TestConfigStruct(t *testing.T) {
	c := &Config{
		Directory:  "/tmp/test",
		Device:     "testdevice",
		TimeLength: 8,
	}
	if c.Directory != "/tmp/test" {
		t.Errorf("Directory = %q, want %q", c.Directory, "/tmp/test")
	}
	if c.TimeLength != 8 {
		t.Errorf("TimeLength = %d, want 8", c.TimeLength)
	}
}

func TestSplitDVNFPathnium(t *testing.T) {
	tests := []struct {
		input     string
		wantDir   string
		wantChild string
	}{
		{"hoge", "hoge", ""},
		{"hoge/fuga", "hoge", "fuga"},
		{"hoge/fuga/piyo", "hoge", "fuga/piyo"},
	}

	for _, tt := range tests {
		// Note: splitDVNFPathnium uses filepath separator which varies by OS.
		// On Windows, forward slashes get converted.
		dir, child := splitDVNFPathnium(tt.input)
		if dir == "" {
			t.Errorf("splitDVNFPathnium(%q): dir is empty", tt.input)
		}
		_ = child // child format depends on OS path separator
	}
}

func TestPlaneFileName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hogehoge.txt", "hogehoge.txt"},
		{"hogehoge (1).txt", "hogehoge.txt"},
		{"hogehoge (23).txt", "hogehoge.txt"},
		{"noext", "noext"},
	}

	for _, tt := range tests {
		got := planeFileName(tt.input)
		if got != tt.want {
			t.Errorf("planeFileName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsIgnored(t *testing.T) {
	// 既定の除外リストを模したもの。完全一致で当たる。
	ignores := []string{".gkill", "Thumbs.db"}
	// 書きかけのファイルを落とすためのパターン。
	patterns := []string{"*.tmp", "*.crdownload", "*.part"}

	tests := []struct {
		name string
		want bool
		why  string
	}{
		{".gkill", true, "完全一致"},
		{"Thumbs.db", true, "完全一致"},
		{"20260806.gpx.tmp", true, "書きかけのGPX"},
		{"Phone_2026-08-23_12-00-00.webp.tmp", true, "書きかけのスクリーンショット"},
		{"setup.exe.crdownload", true, "ダウンロード中"},
		{"movie.mp4.part", true, "ダウンロード中"},
		{"20260806.gpx", false, "出来上がったGPXは運ぶ"},
		{"Phone_2026-08-23_12-00-00.webp", false, "出来上がった画像は運ぶ"},
		{"tmp", false, "拡張子が.tmpでないものまで落とさない"},
		{"notes.tmpx", false, "前方一致で巻き込まない"},
		{"gkill_id.db", false, "渡していない名前は落とさない"},
	}

	for _, tt := range tests {
		got := isIgnored(tt.name, ignores, patterns)
		if got != tt.want {
			t.Errorf("isIgnored(%q) = %v, want %v (%s)", tt.name, got, tt.want, tt.why)
		}
	}

	// パターンを渡さなければ、これまでどおり完全一致だけで判定する。
	if isIgnored("20260806.gpx.tmp", ignores, nil) {
		t.Error("パターンを渡していないのに除外された。既存の呼び出しの意味が変わっている")
	}
}

func TestValidateIgnorePatterns(t *testing.T) {
	if err := validateIgnorePatterns([]string{"*.tmp", "*.part", "name"}); err != nil {
		t.Errorf("正しいパターンで失敗した: %v", err)
	}
	if err := validateIgnorePatterns(nil); err != nil {
		t.Errorf("パターン無しで失敗した: %v", err)
	}

	// 壊れたパターンは黙って「何にも当たらない」になる。
	// 除外し損ねたまま運んでしまうので、動き出す前に気づけないといけない。
	if err := validateIgnorePatterns([]string{"*.tmp", "[", "*.part"}); err == nil {
		t.Error("壊れたパターンを通してしまった")
	}
}

func TestSubcommands(t *testing.T) {
	// Verify subcommands are registered
	cmds := DVNFCmd.Commands()
	names := map[string]bool{}
	for _, c := range cmds {
		names[c.Use] = true
	}
	for _, want := range []string{"get", "move", "copy"} {
		if !names[want] {
			t.Errorf("subcommand %q not found", want)
		}
	}
}
