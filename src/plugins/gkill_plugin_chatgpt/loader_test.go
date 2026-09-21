package main

import (
	"strings"
	"testing"
)

// decodeConversationArray はトップレベルの配列を要素ごとに読む。配列でない JSON・途中で切れた JSON・
// 配列の後ろに続きがある JSON はエラーにする（黙って 0 件や途中までにしない）。
func TestDecodeConversationArrayRejectsNonArrayAndTruncatedJSON(t *testing.T) {
	cases := map[string]string{
		"object instead of array": `{"not":"array"}`,
		"truncated array":         `[{}`,
		"truncated element":       `[{"title":`,
		"garbage after element":   `[{} garbage]`,
		"empty input":             ``,
	}
	for name, input := range cases {
		if _, err := decodeConversationArray(strings.NewReader(input)); err == nil {
			t.Errorf("%s: エラーにならない", name)
		}
	}
	if _, err := decodeConversationArray(strings.NewReader(`{"not":"array"}`)); err == nil || !strings.Contains(err.Error(), "unexpected token") {
		t.Errorf("配列でない JSON のエラー文言 = %v, want unexpected token", err)
	}

	convs, err := decodeConversationArray(strings.NewReader(`[]`))
	if err != nil || len(convs) != 0 {
		t.Errorf("空配列 = %v, %v; want 0 件", convs, err)
	}
	convs, err = decodeConversationArray(strings.NewReader(` [ {} , {} ] `))
	if err != nil || len(convs) != 2 {
		t.Errorf("2要素 = %d 件, %v; want 2", len(convs), err)
	}
}
