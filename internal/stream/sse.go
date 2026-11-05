package stream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ChatCompletionChunk represents the standard OpenAI streaming response payload.
type ChatCompletionChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Choice represents a choice element in the chunk.
type Choice struct {
	Index        int                    `json:"index"`
	Delta        Delta                  `json:"delta"`
	FinishReason *string                `json:"finish_reason"`
	Extra        map[string]interface{} `json:"-"`
}

// Delta represents the incremental token payload.
type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// FragmentFilter processes a logical streamed text field while retaining any
// boundary state required to inspect values split across multiple SSE events.
type FragmentFilter interface {
	Feed(string) string
	Flush() string
}

// FragmentFilterFactory creates isolated filters for content and tool streams.
type FragmentFilterFactory func() FragmentFilter

// SSEProcessor intercepts and detokenizes OpenAI-compatible SSE event streams.
type SSEProcessor struct {
	detok          *Detokenizer
	id             string
	model          string
	contentStreams map[int]*Detokenizer
	toolStreams    map[string]*Detokenizer
	legacyStreams  map[int]*Detokenizer
	filterFactory  FragmentFilterFactory
	contentFilters map[int]FragmentFilter
	toolFilters    map[string]FragmentFilter
	legacyFilters  map[int]FragmentFilter
}

// NewSSEProcessor creates a new SSEProcessor.
func NewSSEProcessor(detok *Detokenizer) *SSEProcessor {
	return NewSSEProcessorWithFilter(detok, nil)
}

// NewSSEProcessorWithFilter creates a processor with an optional post-
// detokenization filter, isolated for every content and tool-call stream.
func NewSSEProcessorWithFilter(detok *Detokenizer, filterFactory FragmentFilterFactory) *SSEProcessor {
	return &SSEProcessor{
		detok:          detok,
		contentStreams: make(map[int]*Detokenizer),
		toolStreams:    make(map[string]*Detokenizer),
		legacyStreams:  make(map[int]*Detokenizer),
		filterFactory:  filterFactory,
		contentFilters: make(map[int]FragmentFilter),
		toolFilters:    make(map[string]FragmentFilter),
		legacyFilters:  make(map[int]FragmentFilter),
	}
}

// ProcessLine processes a single line from an SSE stream.
// Returns one or more formatted SSE lines to emit, or empty if withheld/buffered.
func (p *SSEProcessor) ProcessLine(line string) []string {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return []string{""}
	}

	if !strings.HasPrefix(line, "data:") {
		// Pass through headers, comments, or event types
		return []string{line}
	}

	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

	// Handle stream termination
	if payload == "[DONE]" {
		lines := p.flushStreams()
		lines = append(lines, "data: [DONE]")
		return lines
	}

	var event map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&event); err != nil {
		// If it's not valid JSON, just pass it through
		return []string{line}
	}

	if id, _ := event["id"].(string); p.id == "" && id != "" {
		p.id = id
	}
	if model, _ := event["model"].(string); p.model == "" && model != "" {
		p.model = model
	}

	choices, ok := event["choices"].([]interface{})
	if !ok {
		return []string{line}
	}

	modified := false
	for position, rawChoice := range choices {
		choice, ok := rawChoice.(map[string]interface{})
		if !ok {
			continue
		}
		choiceIndex := numberAsInt(choice["index"], position)
		delta, ok := choice["delta"].(map[string]interface{})
		if !ok {
			continue
		}

		if content, ok := delta["content"].(string); ok && content != "" {
			restored := p.contentStream(choiceIndex).Feed(content)
			delta["content"] = p.filterFeed(p.contentFilter(choiceIndex), restored)
			modified = true
		} else if parts, ok := delta["content"].([]interface{}); ok {
			for _, rawPart := range parts {
				part, ok := rawPart.(map[string]interface{})
				if !ok {
					continue
				}
				if text, ok := part["text"].(string); ok && text != "" {
					restored := p.contentStream(choiceIndex).Feed(text)
					part["text"] = p.filterFeed(p.contentFilter(choiceIndex), restored)
					modified = true
				}
			}
		}

		if calls, ok := delta["tool_calls"].([]interface{}); ok {
			for callPosition, rawCall := range calls {
				call, ok := rawCall.(map[string]interface{})
				if !ok {
					continue
				}
				toolIndex := numberAsInt(call["index"], callPosition)
				function, ok := call["function"].(map[string]interface{})
				if !ok {
					continue
				}
				if arguments, ok := function["arguments"].(string); ok && arguments != "" {
					restored := p.toolStream(choiceIndex, toolIndex).Feed(arguments)
					function["arguments"] = p.filterFeed(p.toolFilter(choiceIndex, toolIndex), restored)
					modified = true
				}
			}
		}

		if function, ok := delta["function_call"].(map[string]interface{}); ok {
			if arguments, ok := function["arguments"].(string); ok && arguments != "" {
				restored := p.legacyStream(choiceIndex).Feed(arguments)
				function["arguments"] = p.filterFeed(p.legacyFilter(choiceIndex), restored)
				modified = true
			}
		}
	}

	if !modified {
		return []string{line}
	}
	modifiedJSON, err := json.Marshal(event)
	if err != nil {
		return []string{line}
	}

	return []string{fmt.Sprintf("data: %s", string(modifiedJSON))}
}

// Flush emits any text retained at end-of-stream when a provider closes the
// connection without sending the optional [DONE] sentinel.
func (p *SSEProcessor) Flush() []string {
	return p.flushStreams()
}

func (p *SSEProcessor) contentStream(choice int) *Detokenizer {
	if current := p.contentStreams[choice]; current != nil {
		return current
	}
	current := p.detok.Clone()
	p.contentStreams[choice] = current
	return current
}

func (p *SSEProcessor) toolStream(choice, tool int) *Detokenizer {
	key := fmt.Sprintf("%d:%d", choice, tool)
	if current := p.toolStreams[key]; current != nil {
		return current
	}
	current := p.detok.Clone()
	p.toolStreams[key] = current
	return current
}

func (p *SSEProcessor) legacyStream(choice int) *Detokenizer {
	if current := p.legacyStreams[choice]; current != nil {
		return current
	}
	current := p.detok.Clone()
	p.legacyStreams[choice] = current
	return current
}

func (p *SSEProcessor) contentFilter(choice int) FragmentFilter {
	if current := p.contentFilters[choice]; current != nil {
		return current
	}
	if p.filterFactory == nil {
		return nil
	}
	current := p.filterFactory()
	p.contentFilters[choice] = current
	return current
}

func (p *SSEProcessor) toolFilter(choice, tool int) FragmentFilter {
	key := fmt.Sprintf("%d:%d", choice, tool)
	if current := p.toolFilters[key]; current != nil {
		return current
	}
	if p.filterFactory == nil {
		return nil
	}
	current := p.filterFactory()
	p.toolFilters[key] = current
	return current
}

func (p *SSEProcessor) legacyFilter(choice int) FragmentFilter {
	if current := p.legacyFilters[choice]; current != nil {
		return current
	}
	if p.filterFactory == nil {
		return nil
	}
	current := p.filterFactory()
	p.legacyFilters[choice] = current
	return current
}

func (p *SSEProcessor) filterFeed(filter FragmentFilter, text string) string {
	if filter == nil {
		return text
	}
	return filter.Feed(text)
}

func flushFiltered(detok *Detokenizer, filter FragmentFilter) string {
	restored := detok.Flush()
	if filter == nil {
		return restored
	}
	return filter.Feed(restored) + filter.Flush()
}

func (p *SSEProcessor) flushStreams() []string {
	var lines []string
	for choice, current := range p.contentStreams {
		if text := flushFiltered(current, p.contentFilters[choice]); text != "" {
			lines = append(lines, fmt.Sprintf("data: %s", p.createChunkJSONForChoice(choice, text)))
		}
	}
	for key, current := range p.toolStreams {
		if text := flushFiltered(current, p.toolFilters[key]); text != "" {
			var choice, tool int
			_, _ = fmt.Sscanf(key, "%d:%d", &choice, &tool)
			event := map[string]interface{}{
				"id": p.id, "object": "chat.completion.chunk", "model": p.model,
				"choices": []interface{}{map[string]interface{}{
					"index": choice, "finish_reason": nil,
					"delta": map[string]interface{}{"tool_calls": []interface{}{map[string]interface{}{
						"index": tool, "function": map[string]interface{}{"arguments": text},
					}}},
				}},
			}
			encoded, _ := json.Marshal(event)
			lines = append(lines, "data: "+string(encoded))
		}
	}
	for choice, current := range p.legacyStreams {
		if text := flushFiltered(current, p.legacyFilters[choice]); text != "" {
			event := map[string]interface{}{
				"id": p.id, "object": "chat.completion.chunk", "model": p.model,
				"choices": []interface{}{map[string]interface{}{
					"index": choice, "finish_reason": nil,
					"delta": map[string]interface{}{"function_call": map[string]interface{}{"arguments": text}},
				}},
			}
			encoded, _ := json.Marshal(event)
			lines = append(lines, "data: "+string(encoded))
		}
	}
	return lines
}

func numberAsInt(value interface{}, fallback int) int {
	switch number := value.(type) {
	case json.Number:
		if parsed, err := number.Int64(); err == nil {
			return int(parsed)
		}
	case float64:
		return int(number)
	case int:
		return number
	}
	return fallback
}

func (p *SSEProcessor) createChunkJSONForChoice(choice int, content string) string {
	event := map[string]interface{}{
		"id": p.id, "object": "chat.completion.chunk", "model": p.model,
		"choices": []interface{}{map[string]interface{}{
			"index": choice, "delta": map[string]interface{}{"content": content}, "finish_reason": nil,
		}},
	}
	encoded, _ := json.Marshal(event)
	return string(encoded)
}

// FormatSSELine encodes a data payload into standard SSE wire format.
func FormatSSELine(data string) []byte {
	var buf bytes.Buffer
	buf.WriteString("data: ")
	buf.WriteString(data)
	buf.WriteString("\n\n")
	return buf.Bytes()
}
