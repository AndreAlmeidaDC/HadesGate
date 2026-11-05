package protocol

import "strings"

// PlaceholderSystemInstruction instructs upstream LLMs to treat HadesGate
// privacy tokens as real, valid data values without lecturing the user.
const PlaceholderSystemInstruction = `[HADESGATE PRIVACY PROXY DIRECTIVE]
Tokens matching [HG_<id>_<TYPE>_<N>] or [[<TYPE>_<N>]] in this conversation are live, valid, concrete data values (such as real names, CPFs, PIX keys, bank accounts, emails, phone numbers, or credentials) that were deterministically tokenized by the HadesGate privacy gateway before reaching you and are automatically detokenized back to their original values in your response stream.
MANDATORY RULES:
1. Treat every [HG_...] token as the real, already-filled, valid data value itself. They are NOT placeholders, templates, generic codes, or missing fields.
2. Whenever you reference, repeat, format, extract, or use any of these values in your response (whether in prose, emails, receipts, summaries, JSON, or code), output the exact [HG_...] token unchanged, character-for-character. Never invent, alter, or strip a token.
3. NEVER mention, warn, explain, apologize, or add notes/footnotes/observations about placeholders, anonymization, masked tokens, or missing data.
4. If the user's message is phrased as an operational command or message containing these values (e.g., a payment instruction, PIX/bank transfer note, or email request), fulfill it helpfully by drafting the corresponding confirmation, message, structured summary, or template using the exact [HG_...] tokens instead of refusing or claiming the fields are unfilled.`

// InjectChatCompletionsSystemInstruction prepends a system directive to an
// OpenAI Chat Completions payload without overwriting existing system prompts.
func InjectChatCompletionsSystemInstruction(payload map[string]interface{}, instruction string) {
	if strings.TrimSpace(instruction) == "" {
		return
	}
	messages, ok := payload["messages"].([]interface{})
	if !ok {
		return
	}
	for _, item := range messages {
		msg, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if role != "system" && role != "developer" {
			continue
		}
		switch content := msg["content"].(type) {
		case string:
			if strings.TrimSpace(content) == "" {
				msg["content"] = instruction
			} else if !strings.Contains(content, instruction) {
				msg["content"] = instruction + "\n\n" + content
			}
			return
		case []interface{}:
			block := map[string]interface{}{"type": "text", "text": instruction}
			msg["content"] = append([]interface{}{block}, content...)
			return
		}
	}
	sysMsg := map[string]interface{}{
		"role":    "system",
		"content": instruction,
	}
	payload["messages"] = append([]interface{}{sysMsg}, messages...)
}

// InjectResponsesSystemInstruction prepends a system directive to an OpenAI
// Responses API payload via the top-level instructions field.
func InjectResponsesSystemInstruction(payload map[string]interface{}, instruction string) {
	if strings.TrimSpace(instruction) == "" {
		return
	}
	if existing, ok := payload["instructions"].(string); ok && strings.TrimSpace(existing) != "" {
		if !strings.Contains(existing, instruction) {
			payload["instructions"] = instruction + "\n\n" + existing
		}
		return
	}
	payload["instructions"] = instruction
}

// InjectAnthropicSystemInstruction prepends a system directive to an Anthropic
// Messages API payload via the top-level system field.
func InjectAnthropicSystemInstruction(payload map[string]interface{}, instruction string) {
	if strings.TrimSpace(instruction) == "" {
		return
	}
	switch system := payload["system"].(type) {
	case string:
		if strings.TrimSpace(system) == "" {
			payload["system"] = instruction
		} else if !strings.Contains(system, instruction) {
			payload["system"] = instruction + "\n\n" + system
		}
	case []interface{}:
		block := map[string]interface{}{"type": "text", "text": instruction}
		payload["system"] = append([]interface{}{block}, system...)
	default:
		payload["system"] = instruction
	}
}

// InjectGeminiSystemInstruction prepends a system directive to a Google Gemini
// generateContent / streamGenerateContent payload via systemInstruction.parts.
func InjectGeminiSystemInstruction(payload map[string]interface{}, instruction string) {
	if strings.TrimSpace(instruction) == "" {
		return
	}
	sysKey := "systemInstruction"
	if _, hasSnake := payload["system_instruction"]; hasSnake && payload["systemInstruction"] == nil {
		sysKey = "system_instruction"
	}

	switch existing := payload[sysKey].(type) {
	case map[string]interface{}:
		parts, _ := existing["parts"].([]interface{})
		for _, item := range parts {
			if part, ok := item.(map[string]interface{}); ok {
				if txt, _ := part["text"].(string); strings.Contains(txt, instruction) {
					return
				}
			}
		}
		block := map[string]interface{}{"text": instruction}
		existing["parts"] = append([]interface{}{block}, parts...)
	case string:
		if strings.Contains(existing, instruction) {
			return
		}
		parts := []interface{}{map[string]interface{}{"text": instruction}}
		if strings.TrimSpace(existing) != "" {
			parts = append(parts, map[string]interface{}{"text": existing})
		}
		payload[sysKey] = map[string]interface{}{"parts": parts}
	default:
		payload[sysKey] = map[string]interface{}{
			"parts": []interface{}{
				map[string]interface{}{"text": instruction},
			},
		}
	}
}

// TransformFunc receives the semantic role and a text-bearing value and returns
// its protected representation.
type TransformFunc func(role, text string) string

// TransformChatCompletions protects text in OpenAI-compatible chat payloads,
// including multimodal text parts and function/tool arguments.
func TransformChatCompletions(payload map[string]interface{}, transform TransformFunc) {
	messages, ok := payload["messages"].([]interface{})
	if !ok {
		return
	}

	for _, item := range messages {
		message, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := message["role"].(string)
		transformContent(message, "content", role, transform)
		transformCallArguments(message["tool_calls"], role, transform)
		if functionCall, ok := message["function_call"].(map[string]interface{}); ok {
			transformStringField(functionCall, "arguments", role, transform)
		}
	}
}

// TransformResponses protects the text-bearing fields used by the OpenAI
// Responses and Codex protocols while leaving identifiers and enum values alone.
func TransformResponses(payload map[string]interface{}, transform TransformFunc) {
	transformResponsesValue(payload, "user", "", transform)
}

var responsesTextKeys = map[string]bool{
	"arguments": true, "content": true, "delta": true, "input": true,
	"input_text": true, "instructions": true, "output": true,
	"output_text": true, "stderr": true, "stdout": true, "text": true,
}

func transformResponsesValue(value interface{}, inheritedRole, key string, transforms ...TransformFunc) {
	if len(transforms) == 0 {
		return
	}
	transform := transforms[0]
	switch current := value.(type) {
	case map[string]interface{}:
		role := inheritedRole
		if explicit, ok := current["role"].(string); ok {
			role = explicit
		} else if kind, _ := current["type"].(string); kind == "function_call_output" || kind == "local_shell_call_output" {
			role = "tool"
		}
		for childKey, child := range current {
			if text, ok := child.(string); ok && responsesTextKeys[childKey] {
				if childKey == "instructions" {
					current[childKey] = transform("system", text)
				} else {
					current[childKey] = transform(role, text)
				}
				continue
			}
			transformResponsesValue(child, role, childKey, transform)
		}
	case []interface{}:
		for _, child := range current {
			transformResponsesValue(child, inheritedRole, key, transform)
		}
	}
}

// TransformAnthropic protects system prompts, message text blocks and tool input.
func TransformAnthropic(payload map[string]interface{}, transform TransformFunc) {
	transformContent(payload, "system", "system", transform)
	messages, ok := payload["messages"].([]interface{})
	if !ok {
		return
	}
	for _, item := range messages {
		message, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := message["role"].(string)
		transformAnthropicContent(message, role, transform)
	}
}

// TransformGemini protects systemInstruction, contents text parts, and
// functionCall / functionResponse payloads in Google Gemini native requests
// while leaving inlineData and fileData untouched.
func TransformGemini(payload map[string]interface{}, transform TransformFunc) {
	transformGeminiSystem(payload, "systemInstruction", transform)
	transformGeminiSystem(payload, "system_instruction", transform)

	contents, ok := payload["contents"].([]interface{})
	if !ok {
		return
	}
	for _, item := range contents {
		content, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := content["role"].(string)
		semanticRole := role
		if semanticRole == "" {
			semanticRole = "user"
		} else if semanticRole == "model" {
			semanticRole = "assistant"
		}
		transformGeminiParts(content["parts"], semanticRole, transform)
	}
}

func transformGeminiSystem(payload map[string]interface{}, key string, transform TransformFunc) {
	switch sys := payload[key].(type) {
	case string:
		if sys != "" {
			payload[key] = transform("system", sys)
		}
	case map[string]interface{}:
		transformGeminiParts(sys["parts"], "system", transform)
	}
}

func transformGeminiParts(partsVal interface{}, role string, transform TransformFunc) {
	parts, ok := partsVal.([]interface{})
	if !ok {
		return
	}
	for _, item := range parts {
		part, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		transformStringField(part, "text", role, transform)
		if fc, ok := part["functionCall"].(map[string]interface{}); ok {
			transformAllStrings(fc["args"], role, transform)
		}
		if fc, ok := part["function_call"].(map[string]interface{}); ok {
			transformAllStrings(fc["args"], role, transform)
		}
		if fr, ok := part["functionResponse"].(map[string]interface{}); ok {
			transformAllStrings(fr["response"], "tool", transform)
		}
		if fr, ok := part["function_response"].(map[string]interface{}); ok {
			transformAllStrings(fr["response"], "tool", transform)
		}
	}
}

func transformContent(container map[string]interface{}, key, role string, transform TransformFunc) {
	switch content := container[key].(type) {
	case string:
		container[key] = transform(role, content)
	case []interface{}:
		for _, item := range content {
			part, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			kind, _ := part["type"].(string)
			if kind == "text" || kind == "input_text" || kind == "output_text" {
				transformStringField(part, "text", role, transform)
			}
		}
	}
}

func transformAnthropicContent(message map[string]interface{}, role string, transform TransformFunc) {
	switch content := message["content"].(type) {
	case string:
		message["content"] = transform(role, content)
	case []interface{}:
		for _, item := range content {
			part, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			switch kind, _ := part["type"].(string); kind {
			case "text":
				transformStringField(part, "text", role, transform)
			case "tool_use", "tool_result":
				transformAllStrings(part["input"], role, transform)
				transformContent(part, "content", role, transform)
			}
		}
	}
}

func transformCallArguments(value interface{}, role string, transform TransformFunc) {
	calls, ok := value.([]interface{})
	if !ok {
		return
	}
	for _, item := range calls {
		call, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if function, ok := call["function"].(map[string]interface{}); ok {
			transformStringField(function, "arguments", role, transform)
		}
	}
}

func transformAllStrings(value interface{}, role string, transform TransformFunc) {
	switch current := value.(type) {
	case map[string]interface{}:
		for key, child := range current {
			if text, ok := child.(string); ok {
				current[key] = transform(role, text)
			} else {
				transformAllStrings(child, role, transform)
			}
		}
	case []interface{}:
		for index, child := range current {
			if text, ok := child.(string); ok {
				current[index] = transform(role, text)
			} else {
				transformAllStrings(child, role, transform)
			}
		}
	}
}

func transformStringField(container map[string]interface{}, key, role string, transform TransformFunc) {
	if text, ok := container[key].(string); ok && text != "" {
		container[key] = transform(role, text)
	}
}

// TransformAllResponseStrings restores placeholders in a decoded JSON response
// without dropping provider-specific fields.
func TransformAllResponseStrings(value interface{}, transform func(string) string) {
	switch current := value.(type) {
	case map[string]interface{}:
		for key, child := range current {
			if text, ok := child.(string); ok {
				current[key] = transform(text)
			} else {
				TransformAllResponseStrings(child, transform)
			}
		}
	case []interface{}:
		for index, child := range current {
			if text, ok := child.(string); ok {
				current[index] = transform(text)
			} else {
				TransformAllResponseStrings(child, transform)
			}
		}
	}
}

var streamingTextKeys = map[string]bool{
	"arguments": true, "content": true, "delta": true, "partial_json": true,
	"text": true, "output_text": true,
}

// TransformStreamingEvent applies a stateful transformer only to fields that
// can carry streamed model text or serialized tool arguments.
func TransformStreamingEvent(value interface{}, transform func(string) string) {
	switch current := value.(type) {
	case map[string]interface{}:
		for key, child := range current {
			if text, ok := child.(string); ok && streamingTextKeys[key] {
				current[key] = transform(text)
			} else if key == "functionCall" || key == "function_call" {
				if fcMap, ok := child.(map[string]interface{}); ok {
					TransformAllResponseStrings(fcMap["args"], transform)
				}
			} else {
				TransformStreamingEvent(child, transform)
			}
		}
	case []interface{}:
		for _, child := range current {
			TransformStreamingEvent(child, transform)
		}
	}
}
