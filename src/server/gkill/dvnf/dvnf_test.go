package dvnf

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSortDVNFs(t *testing.T) {
	dvnfs := []string{"a_20200101", "a_20230315", "a_20210601"}
	SortDVNFs(dvnfs)
	// SortDVNFs sorts in reverse (descending) string order.
	if dvnfs[0] != "a_20230315" {
		t.Errorf("expected first element a_20230315, got %s", dvnfs[0])
	}
	if dvnfs[2] != "a_20200101" {
		t.Errorf("expected last element a_20200101, got %s", dvnfs[2])
	}
}

func TestSortDVNFsEmpty(t *testing.T) {
	dvnfs := []string{}
	SortDVNFs(dvnfs) // should not panic
	if len(dvnfs) != 0 {
		t.Error("expected empty slice")
	}
}

func TestNewDVNF(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "test",
		Device:     "dev1",
		TimeLength: 8,
		Extension:  ".txt",
	}

	result, err := NewDVNF(opt)
	if err != nil {
		t.Fatalf("NewDVNF returned error: %v", err)
	}

	// Should contain the directory prefix.
	if filepath.Dir(result) != tmpDir {
		t.Errorf("expected directory %s, got %s", tmpDir, filepath.Dir(result))
	}

	// Should contain today's date in YYYYMMDD format.
	today := time.Now().Format("20060102")
	base := filepath.Base(result)
	if len(base) == 0 {
		t.Fatal("base name is empty")
	}
	// The format is: Name_Device_YYYYMMDD.ext
	expected := "test_dev1_" + today + ".txt"
	if base != expected {
		t.Errorf("expected base name %q, got %q", expected, base)
	}
}

func TestNewDVNFTimeLength6(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "log",
		TimeLength: 6,
	}
	result, err := NewDVNF(opt)
	if err != nil {
		t.Fatalf("NewDVNF error: %v", err)
	}
	month := time.Now().Format("200601")
	base := filepath.Base(result)
	expected := "log_" + month
	if base != expected {
		t.Errorf("expected %q, got %q", expected, base)
	}
}

func TestNewDVNFTimeLength4(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "log",
		TimeLength: 4,
	}
	result, err := NewDVNF(opt)
	if err != nil {
		t.Fatalf("NewDVNF error: %v", err)
	}
	year := time.Now().Format("2006")
	base := filepath.Base(result)
	expected := "log_" + year
	if base != expected {
		t.Errorf("expected %q, got %q", expected, base)
	}
}

func TestCreateNewDVNFDir(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "backup",
		Device:     "pc",
		TimeLength: 8,
	}

	path, err := CreateNewDVNF(opt, true)
	if err != nil {
		t.Fatalf("CreateNewDVNF(dir) error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory, got file")
	}
}

func TestCreateNewDVNFFile(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "data",
		Device:     "pc",
		TimeLength: 8,
		Extension:  ".db",
	}

	path, err := CreateNewDVNF(opt, false)
	if err != nil {
		t.Fatalf("CreateNewDVNF(file) error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.IsDir() {
		t.Error("expected file, got directory")
	}
}

func TestGetDVNFs(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "item",
		Device:     "d",
		TimeLength: 8,
		Extension:  ".log",
	}

	// Create a DVNF file first.
	_, err := CreateNewDVNF(opt, false)
	if err != nil {
		t.Fatalf("CreateNewDVNF error: %v", err)
	}

	dvnfs, err := GetDVNFs(opt)
	if err != nil {
		t.Fatalf("GetDVNFs error: %v", err)
	}
	if len(dvnfs) != 1 {
		t.Fatalf("expected 1 dvnf, got %d", len(dvnfs))
	}
}

func TestGetDVNFsEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "nothing",
		TimeLength: 8,
	}

	dvnfs, err := GetDVNFs(opt)
	if err != nil {
		t.Fatalf("GetDVNFs error: %v", err)
	}
	if len(dvnfs) != 0 {
		t.Errorf("expected 0 dvnfs, got %d", len(dvnfs))
	}
}

func TestGetDVNFsNonExistentDirectory(t *testing.T) {
	opt := &Option{
		Directory:  filepath.Join(t.TempDir(), "nonexistent"),
		Name:       "x",
		TimeLength: 8,
	}

	dvnfs, err := GetDVNFs(opt)
	if err != nil {
		t.Fatalf("GetDVNFs error: %v", err)
	}
	if len(dvnfs) != 0 {
		t.Errorf("expected 0 dvnfs for non-existent dir, got %d", len(dvnfs))
	}
}

func TestGetLatestDVNF(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "latest",
		Device:     "dev",
		TimeLength: 8,
		Extension:  ".dat",
	}

	// Create a DVNF.
	created, err := CreateNewDVNF(opt, false)
	if err != nil {
		t.Fatalf("CreateNewDVNF error: %v", err)
	}

	latest, err := GetLatestDVNF(opt)
	if err != nil {
		t.Fatalf("GetLatestDVNF error: %v", err)
	}
	if latest != created {
		t.Errorf("expected %q, got %q", created, latest)
	}
}

func TestGetLatestDVNFEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "empty",
		TimeLength: 8,
	}

	latest, err := GetLatestDVNF(opt)
	if err != nil {
		t.Fatalf("GetLatestDVNF error: %v", err)
	}
	if latest != "" {
		t.Errorf("expected empty string for no DVNFs, got %q", latest)
	}
}

func TestGetOrCreateLatestDVNFDir(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "dirtest",
		Device:     "dev",
		TimeLength: 8,
	}

	path, err := GetOrCreateLatestDVNFDir(opt)
	if err != nil {
		t.Fatalf("GetOrCreateLatestDVNFDir error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}

	// Call again; should return the same path without creating a new one.
	path2, err := GetOrCreateLatestDVNFDir(opt)
	if err != nil {
		t.Fatalf("second GetOrCreateLatestDVNFDir error: %v", err)
	}
	if path2 != path {
		t.Errorf("expected same path %q, got %q", path, path2)
	}
}

func TestGetOrCreateLatestDVNFFile(t *testing.T) {
	tmpDir := t.TempDir()
	opt := &Option{
		Directory:  tmpDir,
		Name:       "filetest",
		Device:     "dev",
		TimeLength: 8,
		Extension:  ".txt",
	}

	path, err := GetOrCreateLatestDVNFFile(opt)
	if err != nil {
		t.Fatalf("GetOrCreateLatestDVNFFile error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat error: %v", err)
	}
	if info.IsDir() {
		t.Error("expected file, got directory")
	}

	// Call again; should return the same path.
	path2, err := GetOrCreateLatestDVNFFile(opt)
	if err != nil {
		t.Fatalf("second GetOrCreateLatestDVNFFile error: %v", err)
	}
	if path2 != path {
		t.Errorf("expected same path %q, got %q", path, path2)
	}
}

func TestInvalidTimeLengthOption(t *testing.T) {
	opt := &Option{
		Directory:  t.TempDir(),
		Name:       "bad",
		TimeLength: 5, // invalid: must be 0, 4, 6, or 8
	}

	_, err := NewDVNF(opt)
	if err == nil {
		t.Error("expected error for invalid TimeLength, got nil")
	}

	_, err = GetDVNFs(opt)
	if err == nil {
		t.Error("expected error for invalid TimeLength in GetDVNFs, got nil")
	}
}

func TestEmptyOption(t *testing.T) {
	opt := &Option{}

	_, err := NewDVNF(opt)
	if err == nil {
		t.Error("expected error for empty option, got nil")
	}
}
