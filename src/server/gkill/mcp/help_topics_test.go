package mcp

// gkill_get_mcp_help（help_topics.go）の検査。
//
// ツール一覧の説明文は要約にとどめ、本文はここへ移した（ADR-0622）。守るのは
//   - 全 topic に本文があり、index が全 topic を列挙すること
//   - 説明文から移した知識が本文に実在すること（移した瞬間に消えたら要約化は劣化）
//   - 本文が名指しするツール名が実在すること（本文は tool_handlers_test.go の走査対象外）
//   - 3サーバ全部に載り、gkill へ往復せずに応答すること

import (
	"regexp"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

func allToolNames() *StringSet {
	return NewStringSet(toolNames(concatTools(ReadTools, WriteTools, PluginTools))...)
}

func helpTopicNamesOnly() []string {
	out := []string{}
	for _, topic := range HelpTopics {
		out = append(out, topic.Name)
	}
	return out
}

func mustHelpTopic(t testing.TB, name string) HelpTopic {
	t.Helper()
	topic, ok := HelpTopicByName(name)
	if !ok {
		t.Fatalf("help topic %q is not defined", name)
	}
	return topic
}

func TestHelpTopics(t *testing.T) {
	t.Run("every topic has a title and a substantial body", func(t *testing.T) {
		for _, entry := range HelpTopics {
			expectTrue(t, len(entry.Title) > 10, "%s: title too short", entry.Name)
			expectTrue(t, len(entry.Text) > 300, "%s: text too short", entry.Name)
		}
	})

	t.Run("HELP_TOPIC_NAMES is index plus every topic, and matches the tool's enum", func(t *testing.T) {
		expectTrue(t, HelpTopicNames[0] == HelpIndexTopic, "first name is %q", HelpTopicNames[0])
		expectEqual(t, HelpTopicNames[1:], helpTopicNamesOnly())
		tool := findTool(ReadTools, "gkill_get_mcp_help")
		expectTrue(t, tool != nil, "gkill_get_mcp_help is not defined")
		expectEqual(t, objAt(t, tool, "inputSchema", "properties", "topic").Value("enum"), jsonobj.Strings(HelpTopicNames...))
	})

	t.Run("the index lists every topic", func(t *testing.T) {
		index := BuildHelpPayload("")
		expectTrue(t, strAt(t, index, "topic") == HelpIndexTopic, "topic is %q", strAt(t, index, "topic"))
		expectEqual(t, index.Value("topics"), ListHelpTopics())
		for _, topic := range HelpTopics {
			mustContain(t, strAt(t, index, "text"), "- "+topic.Name+": ")
		}
		expectEqual(t, BuildHelpPayload("index"), index)
		expectEqual(t, BuildHelpPayload(""), index)
	})

	t.Run("a topic payload carries its title and text", func(t *testing.T) {
		payload := BuildHelpPayload("kftl")
		kftl := mustHelpTopic(t, "kftl")
		expectTrue(t, strAt(t, payload, "topic") == "kftl", "topic is %q", strAt(t, payload, "topic"))
		expectTrue(t, strAt(t, payload, "title") == kftl.Title, "title differs")
		expectTrue(t, strAt(t, payload, "text") == kftl.Text, "text differs")
		expectTrue(t, !payload.Has("topics"), "topic payload must not carry topics")
	})

	// 説明文から移した知識。要約化で消えていないことを、移した先で固定する。
	t.Run("knowledge moved out of the tool descriptions lives in a topic", func(t *testing.T) {
		kftl := mustHelpTopic(t, "kftl").Text
		for _, phrase := range []string{"~~", "??", "/endt?", "/end?", "/expense", "monthly 31 skips February", "idempotency_key", "打刻終了タイトルを指定してください"} {
			mustContain(t, kftl, phrase)
		}
		mi := mustHelpTopic(t, "mi").Text
		for _, phrase := range []string{"mi_create", "mi_check", "for_mi", "include_create_mi", "mi_sort_type", "collapse"} {
			mustContain(t, mi, phrase)
		}
		search := mustHelpTopic(t, "search").Text
		for _, phrase := range []string{"even when partial is false", "do not put a repository named by that warning back into query.reps", "is_include_timeis", "payload.kind"} {
			mustContain(t, search, phrase)
		}
		pagination := mustHelpTopic(t, "pagination").Text
		for _, phrase := range []string{"remaining_count", "total_count", "count_only", "group_by", "cannot be combined"} {
			mustContain(t, pagination, phrase)
		}
		dataTypes := mustHelpTopic(t, "data_types").Text
		for _, phrase := range []string{"timeis_start", "expand", "num_min", "create_apps", "gkill_kftl"} {
			mustContain(t, dataTypes, phrase)
		}
		idf := mustHelpTopic(t, "idf").Text
		for _, phrase := range []string{"file_path", "file_url", "gkill_get_idf_file", "thumb", "indexed_at"} {
			mustContain(t, idf, phrase)
		}
		deleted := mustHelpTopic(t, "deleted").Text
		for _, phrase := range []string{"include_deleted_data", "gkill_get_kyou_history", "gkill_restore_kyou", "one-second"} {
			mustContain(t, deleted, phrase)
		}
		rep := mustHelpTopic(t, "rep").Text
		for _, phrase := range []string{"canonical_rep_types", "attached_data_reps", "use_to_write", "writable_only", "directory"} {
			mustContain(t, rep, phrase)
		}
		plugin := mustHelpTopic(t, "plugin").Text
		for _, phrase := range []string{"include_plugin_content", "content_status", "plugin_content_max_text_length", "gkill_get_plugin_list"} {
			mustContain(t, plugin, phrase)
		}
	})

	// 本文は説明文の走査（tool_handlers_test.go）の対象外なので、綴り違いのツール名はここで落とす。
	t.Run("tool names mentioned in the bodies exist on some server", func(t *testing.T) {
		names := allToolNames()
		toolNameRe := regexp.MustCompile(`gkill_[a-z_]+`)
		for _, entry := range HelpTopics {
			for _, name := range toolNameRe.FindAllString(entry.Text, -1) {
				// create_app の値（gkill_kftl / gkill_mcp_readwrite / gkill_mcp_write / gkill_wear / gkill_autolog）は素通し
				switch name {
				case "gkill_kftl", "gkill_mcp_readwrite", "gkill_mcp_write", "gkill_wear", "gkill_autolog":
					continue
				}
				expectTrue(t, names.Has(name), "%s: %s", entry.Name, name)
			}
		}
	})
}

func TestGkillGetMcpHelpTool(t *testing.T) {
	t.Run("is carried by all three servers", func(t *testing.T) {
		expectTrue(t, findTool(ReadTools, "gkill_get_mcp_help") != nil, "missing from ReadTools")
		expectTrue(t, WriteServerReadToolNames.Has("gkill_get_mcp_help"), "missing from WriteServerReadToolNames")
	})

	t.Run("normalizeMcpHelpArgs accepts a known topic, index by omission, and rejects the rest", func(t *testing.T) {
		got, err := NormalizeMcpHelpArgs(obj())
		expectNoError(t, err)
		expectEqual(t, got, obj())
		got, err = NormalizeMcpHelpArgs(jsonobj.Undefined)
		expectNoError(t, err)
		expectEqual(t, got, obj())
		got, err = NormalizeMcpHelpArgs(obj("topic", "mi"))
		expectNoError(t, err)
		expectEqual(t, got, obj("topic", "mi"))
		got, err = NormalizeMcpHelpArgs(obj("topic", "index"))
		expectNoError(t, err)
		expectEqual(t, got, obj("topic", "index"))
		_, err = NormalizeMcpHelpArgs(obj("topic", "tasks"))
		expectErrorMatches(t, err, "must be one of")
		_, err = NormalizeMcpHelpArgs(obj("topic", 3))
		expectGkillApiError(t, err)
		_, err = NormalizeMcpHelpArgs(obj("subject", "mi"))
		expectErrorMatches(t, err, "is not supported")
	})

	t.Run("dispatch returns the topic without calling gkill", func(t *testing.T) {
		calls := 0
		ctx := &CallContext{
			Client: &mockClient{
				callApi: func(_ string, _ *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
					calls++
					return obj(), nil
				},
			},
			SID:              "sid-1",
			IsLocalTransport: true,
		}
		payload, err := HandleReadToolCall(ctx, "gkill_get_mcp_help", obj("topic", "pagination"))
		expectNoError(t, err)
		expectTrue(t, calls == 0, "gkill was called %d times", calls)
		expectTrue(t, strAt(t, payload, "topic") == "pagination", "topic is %q", strAt(t, payload, "topic"))
		expectTrue(t, strAt(t, payload, "text") == mustHelpTopic(t, "pagination").Text, "text differs")
		index, err := HandleReadToolCall(ctx, "gkill_get_mcp_help", obj())
		expectNoError(t, err)
		expectTrue(t, strAt(t, index, "topic") == "index", "topic is %q", strAt(t, index, "topic"))
		expectTrue(t, len(arrAt(t, index, "topics")) == len(HelpTopics), "topics has %d entries", len(arrAt(t, index, "topics")))
	})

	t.Run("summary names the topic", func(t *testing.T) {
		summary, _ := SummarizeReadToolPayload("gkill_get_mcp_help", BuildHelpPayload("kftl"))
		mustContain(t, summary, `Help topic "kftl"`)
		mustContain(t, summary, mustHelpTopic(t, "kftl").Title)
	})
}
