package stream

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hadesgate/hadesgate/internal/anonymizer/vault"
)

func TestSSEProcessorPreservesFieldsAndAllChoices(t *testing.T) {
	v := vault.NewMemoryVault(time.Minute)
	defer v.Close()
	v.Put("sse-preserve", "[PERSON_1]", "João", "PERSON", time.Minute)
	processor := NewSSEProcessor(NewDetokenizer("sse-preserve", v))

	line := `data: {"id":"chunk-1","model":"gpt-test","service_tier":"priority","usage":{"completion_tokens":2},"choices":[{"index":0,"delta":{"content":"Olá [PERSON_1]"},"finish_reason":null},{"index":1,"delta":{"content":"Oi [PERSON_1]"},"finish_reason":null}]}`
	output := processor.ProcessLine(line)
	if len(output) != 1 {
		t.Fatalf("expected one output line, got %v", output)
	}

	var event map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(output[0], "data: ")), &event); err != nil {
		t.Fatalf("invalid output JSON: %v", err)
	}
	if event["service_tier"] != "priority" || event["usage"] == nil {
		t.Fatalf("metadata fields were lost: %+v", event)
	}
	choices := event["choices"].([]interface{})
	if len(choices) != 2 {
		t.Fatalf("choices were lost: %+v", choices)
	}
	for _, rawChoice := range choices {
		content := rawChoice.(map[string]interface{})["delta"].(map[string]interface{})["content"].(string)
		if !strings.Contains(content, "João") {
			t.Fatalf("choice was not restored: %q", content)
		}
	}
}

func TestSSEProcessorRestoresFragmentedToolArguments(t *testing.T) {
	v := vault.NewMemoryVault(time.Minute)
	defer v.Close()
	v.Put("sse-tools", "[PERSON_1]", `João "Admin"`, "PERSON", time.Minute)
	processor := NewSSEProcessor(NewDetokenizer("sse-tools", v))

	first := processor.ProcessLine(`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"[PER"}}]}}]}`)
	second := processor.ProcessLine(`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"SON_1]"}}]}}]}`)
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected outputs: first=%v second=%v", first, second)
	}
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(second[0], "data: ")), &event); err != nil {
		t.Fatalf("invalid output JSON: %v", err)
	}
	choices := event["choices"].([]interface{})
	arguments := choices[0].(map[string]interface{})["delta"].(map[string]interface{})["tool_calls"].([]interface{})[0].(map[string]interface{})["function"].(map[string]interface{})["arguments"]
	if arguments != `João "Admin"` {
		t.Fatalf("tool arguments were not restored safely: %v", arguments)
	}
}
