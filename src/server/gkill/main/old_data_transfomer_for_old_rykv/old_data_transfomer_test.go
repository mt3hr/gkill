package main

import (
	"testing"
	"time"
)

func TestTimeLayoutConstant(t *testing.T) {
	if TimeLayout != time.RFC3339 {
		t.Errorf("TimeLayout = %q, want %q", TimeLayout, time.RFC3339)
	}
}

func TestDefaultVariables(t *testing.T) {
	if SrcKyouDir == "" {
		t.Error("SrcKyouDir should not be empty")
	}
	if TranserDestinationDir == "" {
		t.Error("TranserDestinationDir should not be empty")
	}
	if TempDBFile == "" {
		t.Error("TempDBFile should not be empty")
	}
}

func TestTempDBFileDefault(t *testing.T) {
	// TempDBFile defaults to ":memory:" for in-memory SQLite
	if TempDBFile != ":memory:" {
		t.Errorf("TempDBFile = %q, want %q", TempDBFile, ":memory:")
	}
}
