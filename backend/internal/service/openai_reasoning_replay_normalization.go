package service

import (
	"bytes"
	"fmt"

	"github.com/tidwall/gjson"
)

type openAIJSONReplacement struct {
	start int
	end   int
}

// normalizeOpenAIResponsesReasoningReplay clears reasoning content that the
// Responses API does not accept when historical output items are replayed.
// Upstream requires reasoning item content to be an empty array, even when
// encrypted_content is present. All unrelated request bytes remain unchanged.
func normalizeOpenAIResponsesReasoningReplay(body []byte) ([]byte, int, error) {
	if !bytes.Contains(body, []byte(`"reasoning"`)) || !bytes.Contains(body, []byte(`"content"`)) {
		return body, 0, nil
	}
	if !gjson.ValidBytes(body) {
		return nil, 0, fmt.Errorf("invalid OpenAI Responses request JSON")
	}

	input := gjson.GetBytes(body, "input")
	if !input.IsArray() {
		return body, 0, nil
	}

	replacements := make([]openAIJSONReplacement, 0)
	for _, item := range input.Array() {
		if !item.IsObject() || item.Get("type").String() != "reasoning" {
			continue
		}
		content := item.Get("content")
		if !content.Exists() || content.Type == gjson.Null || (content.IsArray() && len(content.Array()) == 0) {
			continue
		}

		start := content.Index
		end := start + len(content.Raw)
		if start < 0 || end > len(body) || !bytes.Equal(body[start:end], []byte(content.Raw)) {
			return nil, 0, fmt.Errorf("locate OpenAI reasoning content in request body")
		}
		replacements = append(replacements, openAIJSONReplacement{start: start, end: end})
	}
	if len(replacements) == 0 {
		return body, 0, nil
	}

	result := make([]byte, 0, len(body))
	previousEnd := 0
	for _, replacement := range replacements {
		if replacement.start < previousEnd {
			return nil, 0, fmt.Errorf("overlapping OpenAI reasoning content replacements")
		}
		result = append(result, body[previousEnd:replacement.start]...)
		result = append(result, '[', ']')
		previousEnd = replacement.end
	}
	result = append(result, body[previousEnd:]...)
	return result, len(replacements), nil
}
