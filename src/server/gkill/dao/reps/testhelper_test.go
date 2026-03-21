package reps

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

// newTempKmemoRepo creates a KmemoRepository backed by a temp SQLite3 file.
func newTempKmemoRepo(t *testing.T) KmemoRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewKmemoRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "kmemo.db"), true)
	if err != nil {
		t.Fatalf("failed to create kmemo repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// newTempTagRepo creates a TagRepository backed by a temp SQLite3 file.
func newTempTagRepo(t *testing.T) TagRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewTagRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "tag.db"), true)
	if err != nil {
		t.Fatalf("failed to create tag repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// newTempTextRepo creates a TextRepository backed by a temp SQLite3 file.
func newTempTextRepo(t *testing.T) TextRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewTextRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "text.db"), true)
	if err != nil {
		t.Fatalf("failed to create text repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// newTempMiRepo creates a MiRepository backed by a temp SQLite3 file.
func newTempMiRepo(t *testing.T) MiRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewMiRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "mi.db"), true)
	if err != nil {
		t.Fatalf("failed to create mi repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// newTempTimeIsRepo creates a TimeIsRepository backed by a temp SQLite3 file.
func newTempTimeIsRepo(t *testing.T) TimeIsRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewTimeIsRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "timeis.db"), true)
	if err != nil {
		t.Fatalf("failed to create timeis repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// newTempLantanaRepo creates a LantanaRepository backed by a temp SQLite3 file.
func newTempLantanaRepo(t *testing.T) LantanaRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewLantanaRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "lantana.db"), true)
	if err != nil {
		t.Fatalf("failed to create lantana repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// newTempKCRepo creates a KCRepository backed by a temp SQLite3 file.
func newTempKCRepo(t *testing.T) KCRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewKCRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "kc.db"), true)
	if err != nil {
		t.Fatalf("failed to create kc repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// newTempNlogRepo creates a NlogRepository backed by a temp SQLite3 file.
func newTempNlogRepo(t *testing.T) NlogRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewNlogRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "nlog.db"), true)
	if err != nil {
		t.Fatalf("failed to create nlog repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// newTempURLogRepo creates a URLogRepository backed by a temp SQLite3 file.
func newTempURLogRepo(t *testing.T) URLogRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewURLogRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "urlog.db"), true)
	if err != nil {
		t.Fatalf("failed to create urlog repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// newTempNotificationRepo creates a NotificationRepository backed by a temp SQLite3 file.
func newTempNotificationRepo(t *testing.T) NotificationRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewNotificationRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "notification.db"), true)
	if err != nil {
		t.Fatalf("failed to create notification repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// newTempReKyouRepo creates a ReKyouRepository backed by a temp SQLite3 file.
// Note: ReKyou requires a *GkillRepositories reference; pass nil for basic tests.
func newTempReKyouRepo(t *testing.T, reps *GkillRepositories) ReKyouRepository {
	t.Helper()
	dir := t.TempDir()
	repo, err := NewReKyouRepositorySQLite3Impl(context.Background(), filepath.Join(dir, "rekyou.db"), true, reps)
	if err != nil {
		t.Fatalf("failed to create rekyou repo: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// Note: IDFKyouRepository uses NewIDFDirRep with complex dependencies (mux.Router, etc.)
// and is not easily unit-testable with a simple temp file approach.
// IDFKyou tests are deferred to integration tests.

// testTime returns a fixed time for testing.
func testTime() time.Time {
	t, _ := time.Parse(sqlite3impl.TimeLayout, "2025-01-15T10:30:00+09:00")
	return t
}

// testTime2 returns a second fixed time for testing.
func testTime2() time.Time {
	t, _ := time.Parse(sqlite3impl.TimeLayout, "2025-02-20T14:00:00+09:00")
	return t
}

// makeKmemo creates a Kmemo with sensible test defaults.
func makeKmemo(id, content string) Kmemo {
	now := testTime()
	return Kmemo{
		IsDeleted:    false,
		ID:           id,
		Content:      content,
		RelatedTime:  now,
		DataType:     "kmemo",
		CreateTime:   now,
		CreateApp:    "test_app",
		CreateDevice: "test_device",
		CreateUser:   "test_user",
		UpdateTime:   now,
		UpdateApp:    "test_app",
		UpdateUser:   "test_user",
		UpdateDevice: "test_device",
	}
}

// makeTag creates a Tag with sensible test defaults.
func makeTag(id, targetID, tagName string) Tag {
	now := testTime()
	return Tag{
		IsDeleted:    false,
		ID:           id,
		TargetID:     targetID,
		Tag:          tagName,
		RelatedTime:  now,
		CreateTime:   now,
		CreateApp:    "test_app",
		CreateDevice: "test_device",
		CreateUser:   "test_user",
		UpdateTime:   now,
		UpdateApp:    "test_app",
		UpdateUser:   "test_user",
		UpdateDevice: "test_device",
		RepName:      "",
	}
}

// makeText creates a Text with sensible test defaults.
func makeText(id, targetID, textContent string) Text {
	now := testTime()
	return Text{
		IsDeleted:    false,
		ID:           id,
		TargetID:     targetID,
		Text:         textContent,
		RelatedTime:  now,
		CreateTime:   now,
		CreateApp:    "test_app",
		CreateDevice: "test_device",
		CreateUser:   "test_user",
		UpdateTime:   now,
		UpdateApp:    "test_app",
		UpdateUser:   "test_user",
		UpdateDevice: "test_device",
		RepName:      "",
	}
}

// makeMi creates a Mi with sensible test defaults.
func makeMi(id, title string) Mi {
	now := testTime()
	return Mi{
		IsDeleted:    false,
		ID:           id,
		Title:        title,
		DataType:     "mi",
		IsChecked:    false,
		BoardName:    "default",
		CreateTime:   now,
		CreateApp:    "test_app",
		CreateDevice: "test_device",
		CreateUser:   "test_user",
		UpdateTime:   now,
		UpdateApp:    "test_app",
		UpdateUser:   "test_user",
		UpdateDevice: "test_device",
	}
}

// makeDefaultFindQuery creates a FindQuery with all filters off.
func makeDefaultFindQuery() *find.FindQuery {
	return &find.FindQuery{
		OnlyLatestData: true,
	}
}

// makeCalendarFindQuery creates a FindQuery with calendar filter.
func makeCalendarFindQuery(start, end time.Time) *find.FindQuery {
	return &find.FindQuery{
		UseCalendar:       true,
		CalendarStartDate: &start,
		CalendarEndDate:   &end,
		OnlyLatestData:    true,
	}
}

// makeWordFindQuery creates a FindQuery with word filter.
func makeWordFindQuery(words []string) *find.FindQuery {
	return &find.FindQuery{
		UseWords:       true,
		Words:          words,
		WordsAnd:       false,
		OnlyLatestData: true,
	}
}
