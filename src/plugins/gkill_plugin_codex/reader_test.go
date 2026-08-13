package main

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestClassifyHeadOnEveryOuterKind(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		outer   string
		payload string
	}{
		{
			name:    "payload.typeを持つevent_msg",
			line:    `{"timestamp":"2026-01-02T01:00:02.000Z","type":"event_msg","payload":{"type":"user_message","message":"x"}}`,
			outer:   outerEventMsg,
			payload: "user_message",
		},
		{
			name:    "payload.typeを持つresponse_item",
			line:    `{"timestamp":"2026-01-02T01:00:06.000Z","type":"response_item","payload":{"type":"function_call","name":"shell_command"}}`,
			outer:   outerResponseItem,
			payload: "function_call",
		},
		{
			name:  "session_metaはpayload.typeを持たない",
			line:  `{"timestamp":"2026-01-02T01:00:00.000Z","type":"session_meta","payload":{"id":"x","cwd":"c:\\a"}}`,
			outer: outerSessionMeta,
		},
		{
			name:  "turn_contextはpayload.typeを持たない",
			line:  `{"timestamp":"2026-01-02T01:00:01.000Z","type":"turn_context","payload":{"turn_id":"t","model":"gpt-5.3-codex"}}`,
			outer: outerTurnContext,
		},
		{
			name:  "compactedはpayload.typeを持たない",
			line:  `{"timestamp":"2026-01-02T01:01:22.000Z","type":"compacted","payload":{"message":"","replacement_history":[]}}`,
			outer: "compacted",
		},
		{
			name:  "world_stateはpayload.typeを持たない",
			line:  `{"timestamp":"2026-01-02T01:01:23.000Z","type":"world_state","payload":{"full":true}}`,
			outer: "world_state",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			head := classifyHead([]byte(c.line))
			if head.Outer != c.outer {
				t.Errorf("Outer = %q, want %q", head.Outer, c.outer)
			}
			if head.Payload != c.payload {
				t.Errorf("Payload = %q, want %q", head.Payload, c.payload)
			}
		})
	}
}

func TestClassifyHeadSurvivesKeyReorder(t *testing.T) {
	// 外側のキーが入れ替わっても外側のtypeは取れる
	head := classifyHead([]byte(`{"type":"event_msg","timestamp":"2026-01-02T01:00:02.000Z","payload":{"type":"agent_message","message":"x"}}`))
	if head.Outer != outerEventMsg || head.Payload != "agent_message" {
		t.Fatalf("got %+v", head)
	}

	// payload の中で type が先頭でなくても取れる
	head = classifyHead([]byte(`{"timestamp":"2026-01-02T01:00:07.000Z","type":"response_item","payload":{"call_id":"call_1","id":"fco_1","type":"function_call_output"}}`))
	if head.Outer != outerResponseItem || head.Payload != "function_call_output" {
		t.Fatalf("got %+v", head)
	}

	// payload が先に来ると外側は確定できない。そのときは空にして「拾う」側へ倒す
	head = classifyHead([]byte(`{"payload":{"type":"user_message"},"type":"event_msg"}`))
	if head.Outer != "" {
		t.Errorf("Outer = %q, want empty (payloadより後ろのtypeは外側と断定できない)", head.Outer)
	}
}

func TestClassifyHeadUnknownPayloadIsKept(t *testing.T) {
	// 分からなかったものを捨てると、キー順や種別名が変わった日に会話が静かに消える。
	// 判定不能は必ず拾うこと。
	if !keepForBuild(recordHead{Outer: outerResponseItem, Payload: ""}) {
		t.Error("payload.typeが分からない response_item は拾うべき")
	}
	if !keepForBuild(recordHead{Outer: outerEventMsg, Payload: ""}) {
		t.Error("payload.typeが分からない event_msg は拾うべき")
	}
	if !keepForBuild(recordHead{Outer: "", Payload: ""}) {
		t.Error("外側すら分からない行は拾うべき")
	}
	if !keepForBuild(recordHead{Outer: "brand_new_kind"}) {
		t.Error("知らない外側の種別は拾うべき(Codexが増やしたものかもしれない)")
	}

	// 明示的に捨ててよいものだけ捨てる
	for _, kind := range []string{"compacted", "world_state", "inter_agent_communication_metadata"} {
		if keepForBuild(recordHead{Outer: kind}) {
			t.Errorf("%s は捨てるべき", kind)
		}
	}
	if keepForBuild(recordHead{Outer: outerResponseItem, Payload: "function_call_output"}) {
		t.Error("ツールの実行結果は捨てるべき")
	}
	if keepForBuild(recordHead{Outer: outerEventMsg, Payload: "mcp_tool_call_end"}) {
		t.Error("mcp_tool_call_end は捨てるべき")
	}
}

// giantSkippedLine は「捨てる種別の巨大な1行」を作る。
// 実データには19,912,604バイトの custom_tool_call_output が実在する。
func giantSkippedLine(size int) string {
	return `{"timestamp":"2026-01-02T01:00:07.000Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"call_1","output":[{"type":"input_text","text":"` +
		strings.Repeat("x", size) + `"}]}}`
}

func TestLineReaderDrainsGiantSkippedLine(t *testing.T) {
	const giant = 3 * 1024 * 1024
	input := giantSkippedLine(giant) + "\n" +
		`{"timestamp":"2026-01-02T01:00:08.000Z","type":"event_msg","payload":{"type":"agent_message","message":"後続は普通に読める"}}` + "\n"

	reader := newLineReader(strings.NewReader(input))

	head, data, dropped, err := reader.next(keepForBuild)
	if err != nil {
		t.Fatalf("1行目 err = %v", err)
	}
	if data != nil {
		t.Errorf("捨てる行なのに %d バイト保持している", len(data))
	}
	if dropped {
		t.Error("読み捨てた行に dropped を立ててはいけない(上限超過とは別)")
	}
	if head.Payload != "custom_tool_call_output" {
		t.Errorf("Payload = %q", head.Payload)
	}
	if got := reader.bufferCap(); got > readBufBytes {
		t.Errorf("捨てる行のためにバッファを %d バイト確保している(上限 %d)", got, readBufBytes)
	}

	head, data, _, err = reader.next(keepForBuild)
	if err != nil {
		t.Fatalf("2行目 err = %v", err)
	}
	if head.Payload != "agent_message" {
		t.Errorf("2行目の Payload = %q", head.Payload)
	}
	if !strings.Contains(string(data), "後続は普通に読める") {
		t.Errorf("2行目が読めていない: %q", string(data))
	}

	if _, _, _, err = reader.next(keepForBuild); !errors.Is(err, io.EOF) {
		t.Errorf("末尾で EOF を返すべき: %v", err)
	}
}

func TestLineReaderDropsOversizeKeptLine(t *testing.T) {
	// 保持対象なのに上限を超える行。実データの保持対象最大は260,614バイトなので
	// 通常は起きないが、起きたら数えて設定画面に出す。
	huge := `{"timestamp":"2026-01-02T01:00:02.000Z","type":"event_msg","payload":{"type":"user_message","message":"` +
		strings.Repeat("あ", maxRecordBytes) + `"}}`
	input := huge + "\n" +
		`{"timestamp":"2026-01-02T01:00:03.000Z","type":"event_msg","payload":{"type":"agent_message","message":"次の行"}}` + "\n"

	reader := newLineReader(strings.NewReader(input))
	head, _, dropped, err := reader.next(keepForBuild)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !dropped {
		t.Fatal("上限を超えた行には dropped を立てるべき")
	}
	if head.Kind() != outerEventMsg+"/user_message" {
		t.Errorf("Kind = %q (捨てた行でも head は有効であるべき)", head.Kind())
	}

	// 次の行が壊れずに読めること(読み飛ばしが行の途中で止まっていない)
	_, data, _, err := reader.next(keepForBuild)
	if err != nil {
		t.Fatalf("次の行 err = %v", err)
	}
	if !strings.Contains(string(data), "次の行") {
		t.Errorf("次の行が読めていない: %q", string(data))
	}
}

func TestLineReaderHandlesTerminators(t *testing.T) {
	t.Run("末尾に改行が無い", func(t *testing.T) {
		input := `{"timestamp":"2026-01-02T01:00:03.000Z","type":"event_msg","payload":{"type":"agent_message","message":"末尾"}}`
		reader := newLineReader(strings.NewReader(input))
		_, data, _, err := reader.next(keepForBuild)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if !strings.Contains(string(data), "末尾") {
			t.Errorf("data = %q", string(data))
		}
		if _, _, _, err := reader.next(keepForBuild); !errors.Is(err, io.EOF) {
			t.Errorf("次は EOF であるべき: %v", err)
		}
	})

	t.Run("CRLF", func(t *testing.T) {
		input := `{"timestamp":"2026-01-02T01:00:03.000Z","type":"event_msg","payload":{"type":"agent_message","message":"crlf"}}` + "\r\n"
		reader := newLineReader(strings.NewReader(input))
		_, data, _, err := reader.next(keepForBuild)
		if err != nil {
			t.Fatalf("err = %v", err)
		}
		if strings.HasSuffix(string(data), "\r") {
			t.Error("末尾の CR が落ちていない")
		}
		if !strings.HasSuffix(string(data), "}") {
			t.Errorf("data = %q", string(data))
		}
	})

	t.Run("空ファイル", func(t *testing.T) {
		reader := newLineReader(strings.NewReader(""))
		if _, _, _, err := reader.next(keepForBuild); !errors.Is(err, io.EOF) {
			t.Errorf("空ファイルは即 EOF であるべき: %v", err)
		}
	})

	t.Run("空行が混ざる", func(t *testing.T) {
		input := "\n\n" + `{"timestamp":"2026-01-02T01:00:03.000Z","type":"event_msg","payload":{"type":"agent_message","message":"本体"}}` + "\n"
		reader := newLineReader(strings.NewReader(input))
		found := false
		for range 5 {
			_, data, _, err := reader.next(keepForBuild)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if strings.Contains(string(data), "本体") {
				found = true
			}
		}
		if !found {
			t.Error("空行のあとの行が読めていない")
		}
	})
}
