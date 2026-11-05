package protocol

import (
	"strings"
	"testing"
)

func protect(_ string, text string) string { return "<" + text + ">" }

func TestTransformChatCompletions(t *testing.T) {
	payload := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{"type": "text", "text": "sensitive"},
					map[string]interface{}{"type": "image_url", "image_url": map[string]interface{}{"url": "data:image/png;base64,abc"}},
				},
			},
			map[string]interface{}{
				"role": "assistant",
				"tool_calls": []interface{}{
					map[string]interface{}{"function": map[string]interface{}{"arguments": `{"email":"a@example.com"}`}},
				},
			},
		},
	}

	TransformChatCompletions(payload, protect)
	messages := payload["messages"].([]interface{})
	parts := messages[0].(map[string]interface{})["content"].([]interface{})
	if got := parts[0].(map[string]interface{})["text"]; got != "<sensitive>" {
		t.Fatalf("text part not transformed: %v", got)
	}
	imageURL := parts[1].(map[string]interface{})["image_url"].(map[string]interface{})["url"]
	if imageURL != "data:image/png;base64,abc" {
		t.Fatal("non-text multimodal content was modified")
	}
	arguments := messages[1].(map[string]interface{})["tool_calls"].([]interface{})[0].(map[string]interface{})["function"].(map[string]interface{})["arguments"]
	if arguments != `<{"email":"a@example.com"}>` {
		t.Fatalf("tool arguments not transformed: %v", arguments)
	}
}

func TestTransformResponses(t *testing.T) {
	payload := map[string]interface{}{
		"instructions": "system secret",
		"input": []interface{}{
			map[string]interface{}{"role": "user", "content": []interface{}{map[string]interface{}{"type": "input_text", "text": "user secret"}}},
			map[string]interface{}{"type": "function_call_output", "output": "tool secret"},
		},
		"model": "gpt-4o",
	}

	TransformResponses(payload, protect)
	if payload["instructions"] != "<system secret>" {
		t.Fatalf("instructions not transformed: %v", payload["instructions"])
	}
	if payload["model"] != "gpt-4o" {
		t.Fatal("model identifier was modified")
	}
	input := payload["input"].([]interface{})
	text := input[0].(map[string]interface{})["content"].([]interface{})[0].(map[string]interface{})["text"]
	if text != "<user secret>" {
		t.Fatalf("input text not transformed: %v", text)
	}
	if input[1].(map[string]interface{})["output"] != "<tool secret>" {
		t.Fatal("tool output not transformed")
	}
}

func TestTransformAnthropic(t *testing.T) {
	payload := map[string]interface{}{
		"system": "system secret",
		"messages": []interface{}{
			map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{"type": "text", "text": "user secret"},
					map[string]interface{}{"type": "tool_use", "input": map[string]interface{}{"email": "a@example.com", "count": float64(2)}},
				},
			},
		},
	}

	TransformAnthropic(payload, protect)
	if payload["system"] != "<system secret>" {
		t.Fatal("system prompt not transformed")
	}
	parts := payload["messages"].([]interface{})[0].(map[string]interface{})["content"].([]interface{})
	if parts[0].(map[string]interface{})["text"] != "<user secret>" {
		t.Fatal("message text not transformed")
	}
	if parts[1].(map[string]interface{})["input"].(map[string]interface{})["email"] != "<a@example.com>" {
		t.Fatal("tool input not transformed")
	}
}

func TestInjectSystemInstructions(t *testing.T) {
	chatPayload := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "[HG_12345678_CPF_1]"},
		},
	}
	InjectChatCompletionsSystemInstruction(chatPayload, PlaceholderSystemInstruction)
	msgs := chatPayload["messages"].([]interface{})
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages after injection, got %d", len(msgs))
	}
	first := msgs[0].(map[string]interface{})
	if first["role"] != "system" || first["content"] != PlaceholderSystemInstruction {
		t.Fatalf("unexpected injected system message: %#v", first)
	}

	// Existing system prompt should be preserved
	chatWithSys := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "Be concise."},
			map[string]interface{}{"role": "user", "content": "[HG_12345678_CPF_1]"},
		},
	}
	InjectChatCompletionsSystemInstruction(chatWithSys, PlaceholderSystemInstruction)
	sysContent := chatWithSys["messages"].([]interface{})[0].(map[string]interface{})["content"].(string)
	if !strings.Contains(sysContent, PlaceholderSystemInstruction) || !strings.Contains(sysContent, "Be concise.") {
		t.Fatalf("expected combined system instruction, got: %s", sysContent)
	}

	// Responses API
	respPayload := map[string]interface{}{"input": "[HG_12345678_PIX_1]"}
	InjectResponsesSystemInstruction(respPayload, PlaceholderSystemInstruction)
	if respPayload["instructions"] != PlaceholderSystemInstruction {
		t.Fatalf("expected responses instructions to be injected, got: %v", respPayload["instructions"])
	}

	// Anthropic Messages API
	anthropicPayload := map[string]interface{}{
		"system": "Respond in Portuguese.",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "[HG_12345678_PIX_1]"},
		},
	}
	InjectAnthropicSystemInstruction(anthropicPayload, PlaceholderSystemInstruction)
	anthSys := anthropicPayload["system"].(string)
	if !strings.Contains(anthSys, PlaceholderSystemInstruction) || !strings.Contains(anthSys, "Respond in Portuguese.") {
		t.Fatalf("expected combined anthropic system prompt, got: %s", anthSys)
	}

	// Google Gemini API (fresh & existing systemInstruction)
	geminiPayload := map[string]interface{}{
		"contents": []interface{}{
			map[string]interface{}{
				"role":  "user",
				"parts": []interface{}{map[string]interface{}{"text": "[HG_12345678_PIX_1]"}},
			},
		},
	}
	InjectGeminiSystemInstruction(geminiPayload, PlaceholderSystemInstruction)
	gemSysParts := geminiPayload["systemInstruction"].(map[string]interface{})["parts"].([]interface{})
	if len(gemSysParts) != 1 || gemSysParts[0].(map[string]interface{})["text"] != PlaceholderSystemInstruction {
		t.Fatalf("unexpected injected gemini systemInstruction: %#v", geminiPayload["systemInstruction"])
	}

	geminiWithSys := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []interface{}{map[string]interface{}{"text": "Responda sempre em pt-BR."}},
		},
	}
	InjectGeminiSystemInstruction(geminiWithSys, PlaceholderSystemInstruction)
	gemSnakeParts := geminiWithSys["system_instruction"].(map[string]interface{})["parts"].([]interface{})
	if len(gemSnakeParts) != 2 {
		t.Fatalf("expected 2 system_instruction parts, got %d", len(gemSnakeParts))
	}
}

func TestTransformGemini(t *testing.T) {
	var seenRoles []string
	roleTrackingProtect := func(role, text string) string {
		seenRoles = append(seenRoles, role)
		return "<" + text + ">"
	}

	payload := map[string]interface{}{
		"systemInstruction": map[string]interface{}{
			"parts": []interface{}{
				map[string]interface{}{"text": "system secret"},
			},
		},
		"contents": []interface{}{
			map[string]interface{}{
				// omitted role should default to "user"
				"parts": []interface{}{
					map[string]interface{}{"text": "user secret"},
					map[string]interface{}{
						"inlineData": map[string]interface{}{
							"mimeType": "image/png",
							"data":     "iVBORw0KGgoAAAANSUhEUg==",
						},
					},
				},
			},
			map[string]interface{}{
				"role": "model",
				"parts": []interface{}{
					map[string]interface{}{
						"functionCall": map[string]interface{}{
							"name": "send_invoice",
							"args": map[string]interface{}{"email": "a@example.com"},
						},
					},
				},
			},
			map[string]interface{}{
				"role": "user",
				"parts": []interface{}{
					map[string]interface{}{
						"functionResponse": map[string]interface{}{
							"name":     "send_invoice",
							"response": map[string]interface{}{"status": "sent to a@example.com"},
						},
					},
				},
			},
		},
	}

	TransformGemini(payload, roleTrackingProtect)

	sysText := payload["systemInstruction"].(map[string]interface{})["parts"].([]interface{})[0].(map[string]interface{})["text"]
	if sysText != "<system secret>" {
		t.Fatalf("systemInstruction not transformed: %v", sysText)
	}

	contents := payload["contents"].([]interface{})
	firstParts := contents[0].(map[string]interface{})["parts"].([]interface{})
	if firstParts[0].(map[string]interface{})["text"] != "<user secret>" {
		t.Fatalf("user text part not transformed: %v", firstParts[0])
	}
	inlineData := firstParts[1].(map[string]interface{})["inlineData"].(map[string]interface{})["data"]
	if inlineData != "iVBORw0KGgoAAAANSUhEUg==" {
		t.Fatalf("inlineData binary blob was modified: %v", inlineData)
	}

	fcArgs := contents[1].(map[string]interface{})["parts"].([]interface{})[0].(map[string]interface{})["functionCall"].(map[string]interface{})["args"].(map[string]interface{})
	if fcArgs["email"] != "<a@example.com>" {
		t.Fatalf("functionCall args not transformed: %v", fcArgs["email"])
	}

	frResp := contents[2].(map[string]interface{})["parts"].([]interface{})[0].(map[string]interface{})["functionResponse"].(map[string]interface{})["response"].(map[string]interface{})
	if frResp["status"] != "<sent to a@example.com>" {
		t.Fatalf("functionResponse not transformed: %v", frResp["status"])
	}

	if len(seenRoles) != 4 || seenRoles[0] != "system" || seenRoles[1] != "user" || seenRoles[2] != "assistant" || seenRoles[3] != "tool" {
		t.Fatalf("unexpected semantic roles recorded: %v", seenRoles)
	}
}
