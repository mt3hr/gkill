package kftl

import (
	"testing"
)

func TestNewKFTLFactory(t *testing.T) {
	f := newKFTLFactory()
	if f == nil {
		t.Fatal("newKFTLFactory returned nil")
	}
	if f.prevLineIsMetaInfo {
		t.Error("new factory should have prevLineIsMetaInfo = false")
	}
}

func TestKFTLFactoryReset(t *testing.T) {
	f := newKFTLFactory()
	if f.prevLineIsMetaInfo {
		t.Error("before reset, prevLineIsMetaInfo should be false")
	}
	f.reset()
	if !f.prevLineIsMetaInfo {
		t.Error("after reset, prevLineIsMetaInfo should be true")
	}
}

func TestGenerateDefaultConstructor_TagPrefix(t *testing.T) {
	f := newKFTLFactory()
	f.reset()

	constructor := f.generateDefaultConstructor(splitterTag+"sometag", nil)
	if constructor == nil {
		t.Fatal("generateDefaultConstructor returned nil for tag prefix")
	}

	ctx := &KFTLStatementLineContext{
		ThisStatementLineText:     splitterTag + "sometag",
		ThisStatementLineTargetID: "test-id",
		ThisIsPrototype:           false,
	}
	line := constructor(splitterTag+"sometag", ctx)
	if line == nil {
		t.Fatal("tag constructor returned nil")
	}
	if line.GetLabelName() != "tag" {
		t.Errorf("label = %q, want %q", line.GetLabelName(), "tag")
	}
}

func TestGenerateDefaultConstructor_StartText(t *testing.T) {
	f := newKFTLFactory()
	f.reset()

	constructor := f.generateDefaultConstructor(splitterStartText, nil)
	if constructor == nil {
		t.Fatal("generateDefaultConstructor returned nil for start text")
	}

	ctx := &KFTLStatementLineContext{
		ThisStatementLineText:     splitterStartText,
		ThisStatementLineTargetID: "test-id",
		ThisIsPrototype:           false,
	}
	line := constructor(splitterStartText, ctx)
	if line == nil {
		t.Fatal("start text constructor returned nil")
	}
	if line.GetLabelName() != "startText" {
		t.Errorf("label = %q, want %q", line.GetLabelName(), "startText")
	}
}

func TestGenerateDefaultConstructor_Split(t *testing.T) {
	f := newKFTLFactory()
	f.reset()

	constructor := f.generateDefaultConstructor(splitterSplit, nil)
	if constructor == nil {
		t.Fatal("generateDefaultConstructor returned nil for split")
	}

	ctx := &KFTLStatementLineContext{
		ThisStatementLineText:     splitterSplit,
		ThisStatementLineTargetID: "test-id",
	}
	line := constructor(splitterSplit, ctx)
	if line == nil {
		t.Fatal("split constructor returned nil")
	}
}

func TestGenerateDefaultConstructor_SplitNextSecond(t *testing.T) {
	f := newKFTLFactory()
	f.reset()

	constructor := f.generateDefaultConstructor(splitterSplitNextSecond, nil)
	if constructor == nil {
		t.Fatal("generateDefaultConstructor returned nil for split next second")
	}

	ctx := &KFTLStatementLineContext{
		ThisStatementLineText:     splitterSplitNextSecond,
		ThisStatementLineTargetID: "test-id",
	}
	line := constructor(splitterSplitNextSecond, ctx)
	if line == nil {
		t.Fatal("split next second constructor returned nil")
	}
}

func TestGenerateDefaultConstructor_Fallback(t *testing.T) {
	f := newKFTLFactory()
	f.reset()

	fallbackCalled := false
	fallback := func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
		fallbackCalled = true
		return nil
	}

	constructor := f.generateDefaultConstructor("some random text", fallback)
	if constructor == nil {
		t.Fatal("generateDefaultConstructor returned nil for unknown text")
	}

	ctx := &KFTLStatementLineContext{
		ThisStatementLineText:     "some random text",
		ThisStatementLineTargetID: "test-id",
	}
	constructor("some random text", ctx)
	if !fallbackCalled {
		t.Error("expected fallback to be called for unrecognized text")
	}
}

func TestGenerateKmemoConstructor(t *testing.T) {
	f := newKFTLFactory()
	f.reset()
	if !f.prevLineIsMetaInfo {
		t.Fatal("after reset, prevLineIsMetaInfo should be true")
	}

	// With unrecognized next line text, should return kmemo constructor
	constructor := f.generateKmemoConstructor("hello world")
	if constructor == nil {
		t.Fatal("generateKmemoConstructor returned nil")
	}

	ctx := &KFTLStatementLineContext{
		ThisStatementLineText:     "hello world",
		ThisStatementLineTargetID: "test-id",
		ThisIsPrototype:           true,
	}
	line := constructor("hello world", ctx)
	if line == nil {
		t.Fatal("kmemo constructor returned nil")
	}
	if f.prevLineIsMetaInfo {
		t.Error("after kmemo line, prevLineIsMetaInfo should be false")
	}
}

func TestGenerateNoneConstructor(t *testing.T) {
	f := newKFTLFactory()
	f.prevLineIsMetaInfo = false

	constructor := f.generateNoneConstructor("unknown text")
	if constructor == nil {
		t.Fatal("generateNoneConstructor returned nil")
	}

	ctx := &KFTLStatementLineContext{
		ThisStatementLineText:     "unknown text",
		ThisStatementLineTargetID: "test-id",
	}
	line := constructor("unknown text", ctx)
	if line == nil {
		t.Fatal("none constructor returned nil")
	}
	if !f.prevLineIsMetaInfo {
		t.Error("after none line, prevLineIsMetaInfo should be true")
	}
}

func TestGenerateDefaultConstructor_AllSplitters(t *testing.T) {
	// Verify that all known splitters produce non-nil constructors
	splitters := []string{
		splitterKC,
		splitterMi,
		splitterLantana,
		splitterNlog,
		splitterTimeIsStart,
		splitterTimeIsEnd,
		splitterTimeIs,
		splitterTimeIsEndIfExist,
		splitterTimeIsEndByTag,
		splitterTimeIsEndByTagIfExist,
		splitterURLog,
	}

	for _, s := range splitters {
		f := newKFTLFactory()
		f.reset()
		constructor := f.generateDefaultConstructor(s, nil)
		if constructor == nil {
			t.Errorf("generateDefaultConstructor returned nil for splitter %q", s)
		}
	}
}

func TestGenerateDefaultConstructor_RelatedTime(t *testing.T) {
	f := newKFTLFactory()
	f.reset()

	constructor := f.generateDefaultConstructor(splitterRelatedTime+"2024-01-01", nil)
	if constructor == nil {
		t.Fatal("generateDefaultConstructor returned nil for related time prefix")
	}

	ctx := &KFTLStatementLineContext{
		ThisStatementLineText:     splitterRelatedTime + "2024-01-01",
		ThisStatementLineTargetID: "test-id",
	}
	line := constructor(splitterRelatedTime+"2024-01-01", ctx)
	if line == nil {
		t.Fatal("related time constructor returned nil")
	}
	if line.GetLabelName() != "relatedTime" {
		t.Errorf("label = %q, want %q", line.GetLabelName(), "relatedTime")
	}
}

func TestFactoryPrevLineIsMetaInfoStateTransitions(t *testing.T) {
	f := newKFTLFactory()

	// Initial state
	if f.prevLineIsMetaInfo {
		t.Error("initial state should be false")
	}

	// reset sets to true
	f.reset()
	if !f.prevLineIsMetaInfo {
		t.Error("after reset should be true")
	}

	// split sets to true
	f.prevLineIsMetaInfo = false
	ctx := &KFTLStatementLineContext{
		ThisStatementLineText:     splitterSplit,
		ThisStatementLineTargetID: "test-id",
	}
	constructor := f.generateDefaultConstructor(splitterSplit, nil)
	constructor(splitterSplit, ctx)
	if !f.prevLineIsMetaInfo {
		t.Error("after split, prevLineIsMetaInfo should be true")
	}

	// splitNextSecond also sets to true
	f.prevLineIsMetaInfo = false
	ctx2 := &KFTLStatementLineContext{
		ThisStatementLineText:     splitterSplitNextSecond,
		ThisStatementLineTargetID: "test-id",
	}
	constructor2 := f.generateDefaultConstructor(splitterSplitNextSecond, nil)
	constructor2(splitterSplitNextSecond, ctx2)
	if !f.prevLineIsMetaInfo {
		t.Error("after splitNextSecond, prevLineIsMetaInfo should be true")
	}
}
