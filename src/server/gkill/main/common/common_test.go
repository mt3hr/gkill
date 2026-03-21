package common

import (
	"testing"
)

func TestAppNameDefault(t *testing.T) {
	if AppName == "" {
		t.Error("AppName should not be empty")
	}
}

func TestIDFCmdNotNil(t *testing.T) {
	if IDFCmd == nil {
		t.Fatal("IDFCmd should not be nil")
	}
	if IDFCmd.Use != "idf" {
		t.Errorf("IDFCmd.Use = %q, want %q", IDFCmd.Use, "idf")
	}
}

func TestDVNFCmdNotNil(t *testing.T) {
	if DVNFCmd == nil {
		t.Fatal("DVNFCmd should not be nil")
	}
}

func TestVersionCommandNotNil(t *testing.T) {
	if VersionCommand == nil {
		t.Fatal("VersionCommand should not be nil")
	}
	if VersionCommand.Use != "version" {
		t.Errorf("VersionCommand.Use = %q, want %q", VersionCommand.Use, "version")
	}
}

func TestGenerateThumbCacheCmdNotNil(t *testing.T) {
	if GenerateThumbCacheCmd == nil {
		t.Fatal("GenerateThumbCacheCmd should not be nil")
	}
}

func TestGenerateVideoCacheCmdNotNil(t *testing.T) {
	if GenerateVideoCacheCmd == nil {
		t.Fatal("GenerateVideoCacheCmd should not be nil")
	}
}

func TestOptimizeCmdNotNil(t *testing.T) {
	if OptimizeCmd == nil {
		t.Fatal("OptimizeCmd should not be nil")
	}
}

func TestUpdateCacheCmdNotNil(t *testing.T) {
	if UpdateCacheCmd == nil {
		t.Fatal("UpdateCacheCmd should not be nil")
	}
}
