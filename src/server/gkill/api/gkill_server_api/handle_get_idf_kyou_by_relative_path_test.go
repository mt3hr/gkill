package gkill_server_api

import "testing"

func TestResolveIDFRelativePath(t *testing.T) {
	tests := []struct {
		name              string
		currentTargetFile string
		relativePath      string
		want              string
		wantErr           bool
	}{
		{
			name:              "rep直下の同階層ファイル",
			currentTargetFile: "桜陽.md",
			relativePath:      "index.md",
			want:              "index.md",
		},
		{
			name:              "サブディレクトリ内からの同階層ファイル",
			currentTargetFile: "docs/a.md",
			relativePath:      "b.md",
			want:              "docs/b.md",
		},
		{
			name:              "./つき相対パス",
			currentTargetFile: "docs/a.md",
			relativePath:      "./b.md",
			want:              "docs/b.md",
		},
		{
			name:              "親ディレクトリへの相対パス",
			currentTargetFile: "docs/a.md",
			relativePath:      "../index.md",
			want:              "index.md",
		},
		{
			name:              "フラグメントを除去する",
			currentTargetFile: "桜陽.md",
			relativePath:      "index.md#section",
			want:              "index.md",
		},
		{
			name:              "URLエンコードを解除する",
			currentTargetFile: "桜陽.md",
			relativePath:      "%E8%A8%98%E9%8C%B2%E5%93%B2%E5%AD%A6.md",
			want:              "記録哲学.md",
		},
		{
			name:              "バックスラッシュ区切りの基準ファイル",
			currentTargetFile: "docs\\a.md",
			relativePath:      "b.md",
			want:              "docs/b.md",
		},
		{
			name:              "rep外へのパストラバーサルは拒否",
			currentTargetFile: "a.md",
			relativePath:      "../outside.md",
			wantErr:           true,
		},
		{
			name:              "絶対パスは拒否",
			currentTargetFile: "a.md",
			relativePath:      "/etc/passwd",
			wantErr:           true,
		},
		{
			name:              "Windows絶対パスは拒否",
			currentTargetFile: "a.md",
			relativePath:      "C:\\windows\\system32",
			wantErr:           true,
		},
		{
			name:              "空文字は拒否",
			currentTargetFile: "a.md",
			relativePath:      "",
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveIDFRelativePath(tt.currentTargetFile, tt.relativePath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
