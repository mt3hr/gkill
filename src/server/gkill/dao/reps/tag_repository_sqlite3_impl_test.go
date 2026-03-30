package reps

import (
	"context"
	"testing"
	"time"
)

func TestTagAddAndGet(t *testing.T) {
	repo := newTempTagRepo(t)
	ctx := context.Background()

	tag := makeTag("tag-001", "target-001", "日記")
	if err := repo.AddTagInfo(ctx, tag); err != nil {
		t.Fatalf("AddTagInfo failed: %v", err)
	}

	got, err := repo.GetTag(ctx, "tag-001", nil)
	if err != nil {
		t.Fatalf("GetTag failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetTag returned nil")
	}
	if got.ID != "tag-001" {
		t.Errorf("ID = %q, want %q", got.ID, "tag-001")
	}
	if got.Tag != "日記" {
		t.Errorf("Tag = %q, want %q", got.Tag, "日記")
	}
	if got.TargetID != "target-001" {
		t.Errorf("TargetID = %q, want %q", got.TargetID, "target-001")
	}
}

func TestTagGetByTargetID(t *testing.T) {
	repo := newTempTagRepo(t)
	ctx := context.Background()

	tag1 := makeTag("tag-a", "target-x", "仕事")
	tag2 := makeTag("tag-b", "target-x", "重要")
	tag3 := makeTag("tag-c", "target-y", "個人")

	for _, tag := range []Tag{tag1, tag2, tag3} {
		if err := repo.AddTagInfo(ctx, tag); err != nil {
			t.Fatalf("AddTagInfo failed: %v", err)
		}
	}

	tags, err := repo.GetTagsByTargetID(ctx, "target-x")
	if err != nil {
		t.Fatalf("GetTagsByTargetID failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags for target-x, got %d", len(tags))
	}
}

func TestTagGetByTagName(t *testing.T) {
	repo := newTempTagRepo(t)
	ctx := context.Background()

	tag1 := makeTag("tag-1", "target-a", "食事")
	tag2 := makeTag("tag-2", "target-b", "食事")
	tag3 := makeTag("tag-3", "target-c", "運動")

	for _, tag := range []Tag{tag1, tag2, tag3} {
		if err := repo.AddTagInfo(ctx, tag); err != nil {
			t.Fatalf("AddTagInfo failed: %v", err)
		}
	}

	tags, err := repo.GetTagsByTagName(ctx, "食事")
	if err != nil {
		t.Fatalf("GetTagsByTagName failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags named '食事', got %d", len(tags))
	}
}

func TestTagGetAllTagNames(t *testing.T) {
	repo := newTempTagRepo(t)
	ctx := context.Background()

	tags := []Tag{
		makeTag("tag-1", "t-a", "食事"),
		makeTag("tag-2", "t-b", "運動"),
		makeTag("tag-3", "t-c", "食事"),
	}
	for _, tag := range tags {
		if err := repo.AddTagInfo(ctx, tag); err != nil {
			t.Fatalf("AddTagInfo failed: %v", err)
		}
	}

	names, err := repo.GetAllTagNames(ctx)
	if err != nil {
		t.Fatalf("GetAllTagNames failed: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("expected 2 unique tag names, got %d: %v", len(names), names)
	}
}

func TestTagGetAllTags(t *testing.T) {
	repo := newTempTagRepo(t)
	ctx := context.Background()

	for i := range 3 {
		tag := makeTag("tag-all-"+string(rune('a'+i)), "target-"+string(rune('a'+i)), "タグ")
		tag.UpdateTime = tag.UpdateTime.Add(time.Duration(i) * time.Second)
		if err := repo.AddTagInfo(ctx, tag); err != nil {
			t.Fatalf("AddTagInfo failed: %v", err)
		}
	}

	allTags, err := repo.GetAllTags(ctx)
	if err != nil {
		t.Fatalf("GetAllTags failed: %v", err)
	}
	if len(allTags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(allTags))
	}
}

func TestTagFindTags(t *testing.T) {
	repo := newTempTagRepo(t)
	ctx := context.Background()

	tag := makeTag("tag-find", "target-find", "検索テスト")
	if err := repo.AddTagInfo(ctx, tag); err != nil {
		t.Fatalf("AddTagInfo failed: %v", err)
	}

	query := makeDefaultFindQuery()
	found, err := repo.FindTags(ctx, query)
	if err != nil {
		t.Fatalf("FindTags failed: %v", err)
	}
	if len(found) != 1 {
		t.Errorf("expected 1 tag, got %d", len(found))
	}
}

func TestTagGetHistories(t *testing.T) {
	repo := newTempTagRepo(t)
	ctx := context.Background()

	tag1 := makeTag("tag-hist", "target-hist", "v1")
	if err := repo.AddTagInfo(ctx, tag1); err != nil {
		t.Fatalf("AddTagInfo failed: %v", err)
	}

	tag2 := makeTag("tag-hist", "target-hist", "v2")
	tag2.UpdateTime = tag2.UpdateTime.Add(time.Hour)
	if err := repo.AddTagInfo(ctx, tag2); err != nil {
		t.Fatalf("AddTagInfo failed: %v", err)
	}

	histories, err := repo.GetTagHistories(ctx, "tag-hist")
	if err != nil {
		t.Fatalf("GetTagHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}
