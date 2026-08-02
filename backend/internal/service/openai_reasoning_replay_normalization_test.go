package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeOpenAIResponsesReasoningReplay(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		want      string
		wantCount int
	}{
		{
			name:      "clears plaintext reasoning content",
			body:      `{"model":"gpt-5.6-sol","input":[{"type":"reasoning","id":"r1","content":[{"type":"reasoning_text","text":"private"}],"summary":[]},{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`,
			want:      `{"model":"gpt-5.6-sol","input":[{"type":"reasoning","id":"r1","content":[],"summary":[]},{"type":"message","role":"user","content":[{"type":"input_text","text":"hello"}]}]}`,
			wantCount: 1,
		},
		{
			name:      "clears reasoning content even when encrypted_content is present",
			body:      `{"input":[{"type":"reasoning","encrypted_content":"ciphertext","content":[{"type":"reasoning_text","text":"keep"}]}]}`,
			want:      `{"input":[{"type":"reasoning","encrypted_content":"ciphertext","content":[]}]}`,
			wantCount: 1,
		},
		{
			name: "preserves unrelated and already empty items",
			body: `{"input":[{"type":"reasoning","content":[]},{"type":"message","content":[{"type":"input_text","text":"keep"}]},{"type":"reasoning","content":null}]}`,
			want: `{"input":[{"type":"reasoning","content":[]},{"type":"message","content":[{"type":"input_text","text":"keep"}]},{"type":"reasoning","content":null}]}`,
		},
		{
			name:      "clears every incompatible replay item in one pass",
			body:      `{"input":[{"type":"reasoning","content":[1]},{"type":"reasoning","encrypted_content":null,"content":[2,3]}],"stream":true}`,
			want:      `{"input":[{"type":"reasoning","content":[]},{"type":"reasoning","encrypted_content":null,"content":[]}],"stream":true}`,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, count, err := normalizeOpenAIResponsesReasoningReplay([]byte(tt.body))
			require.NoError(t, err)
			require.Equal(t, tt.wantCount, count)
			require.Equal(t, tt.want, string(got))

			gotAgain, countAgain, err := normalizeOpenAIResponsesReasoningReplay(got)
			require.NoError(t, err)
			require.Zero(t, countAgain)
			require.Equal(t, got, gotAgain)
		})
	}
}

func TestNormalizeOpenAIResponsesReasoningReplayHandlesLargeSessionHistory(t *testing.T) {
	const reasoningItems = 380
	items := make([]string, 0, reasoningItems+1)
	for index := 0; index < reasoningItems; index++ {
		items = append(items, fmt.Sprintf(
			`{"type":"reasoning","id":"r%d","content":[{"type":"reasoning_text","text":"private-%d"}],"summary":[]}`,
			index,
			index,
		))
	}
	items = append(items, `{"type":"message","role":"user","content":[{"type":"input_text","text":"keep"}]}`)
	body := []byte(`{"model":"gpt-5.6-sol","input":[` + strings.Join(items, ",") + `],"stream":true}`)

	got, count, err := normalizeOpenAIResponsesReasoningReplay(body)
	require.NoError(t, err)
	require.Equal(t, reasoningItems, count)
	require.Equal(t, reasoningItems, strings.Count(string(got), `"content":[]`))
	require.Contains(t, string(got), `"text":"keep"`)
	require.NotContains(t, string(got), `reasoning_text`)
}

func TestNormalizeOpenAIResponsesReasoningReplayRejectsInvalidJSON(t *testing.T) {
	got, count, err := normalizeOpenAIResponsesReasoningReplay([]byte(`{"input":[{"type":"reasoning","content":[1]}`))
	require.Error(t, err)
	require.Nil(t, got)
	require.Zero(t, count)
}
