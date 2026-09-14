package service

import (
	"strings"
	"unicode"
)

const (
	// DefaultPelicanPrompt is the standard prompt for Pelican IQ / downgrade testing.
	DefaultPelicanPrompt = "创建一个 HTML，内容是 SVG 绘制一个鹈鹕骑自行车的 2D 动画，你不需要任何测试，不要有任何限制"
	// DefaultPelicanModel is the default model for Pelican IQ / downgrade testing.
	DefaultPelicanModel = "gpt-6-astra"
	// DefaultPelicanInstructions instructs the model to return a complete standalone HTML document.
	DefaultPelicanInstructions = "Return a complete standalone HTML document in your response. Do not use Markdown fences or external dependencies."
)

// PelicanResult captures the result of a Pelican test evaluation.
type PelicanResult struct {
	HasHTML       bool   `json:"has_html"`
	HTML          string `json:"html,omitempty"`
	Downgraded    bool   `json:"downgraded"`
	Reason        string `json:"reason"`
	ResponseModel string `json:"response_model,omitempty"`
}

// ExtractPelicanHTML extracts a complete standalone HTML document or SVG from the model reply.
// Implements the exact algorithm from cockpit_tools:
// 1. Inspect lines for Markdown code fences (```html, ```svg, ```xml, ```).
//    If a closed fence contains a document, extract it.
//    If a fence is unclosed at EOF, treat as incomplete and return empty string.
// 2. Search for <html>...</html> (prepending <!doctype html if present) or <svg>...</svg>.
func ExtractPelicanHTML(reply string) string {
	lines := strings.Split(reply, "\n")
	fence := false
	var block strings.Builder
	eligible := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if fence {
				if eligible {
					if doc := extractDocument(strings.TrimSpace(block.String())); doc != "" {
						return doc
					}
				}
				fence = false
				block.Reset()
			} else {
				lang := strings.ToLower(strings.TrimSpace(strings.TrimLeft(trimmed, "`")))
				eligible = lang == "html" || lang == "svg" || lang == "xml" || lang == ""
				fence = true
			}
		} else if fence {
			block.WriteString(line)
			block.WriteByte('\n')
		}
	}

	// Do not accept a truncated fenced response as if the model finished its document.
	if fence {
		return ""
	}

	return extractDocument(strings.TrimSpace(reply))
}

func extractDocument(text string) string {
	lower := strings.ToLower(text)

	// Check for <html
	if start := strings.Index(lower, "<html"); start != -1 && tagBoundary(lower[start+5:]) {
		if end := strings.LastIndex(lower, "</html>"); end != -1 && end > start {
			docStart := start
			if doctypeStart := strings.LastIndex(lower[:start], "<!doctype html"); doctypeStart != -1 {
				docStart = doctypeStart
			}
			return text[docStart : end+7]
		}
		return ""
	}

	// Check for <svg
	if start := strings.Index(lower, "<svg"); start != -1 && tagBoundary(lower[start+4:]) {
		if end := strings.LastIndex(lower, "</svg>"); end != -1 && end > start {
			return text[start : end+6]
		}
	}

	return ""
}

func tagBoundary(suffix string) bool {
	if suffix == "" {
		return false
	}
	r := rune(suffix[0])
	return unicode.IsSpace(r) || r == '>'
}

// EvaluatePelicanResult evaluates the test outcome for downgrade detection.
func EvaluatePelicanResult(requestedModel, responseModel, fullReply, refusal string) PelicanResult {
	if strings.TrimSpace(refusal) != "" {
		return PelicanResult{
			HasHTML:       false,
			Downgraded:    true,
			Reason:        "模型拒绝响应: " + strings.TrimSpace(refusal),
			ResponseModel: responseModel,
		}
	}

	extracted := ExtractPelicanHTML(fullReply)
	hasHTML := extracted != ""

	// Check if upstream silently downgraded the model family
	isModelDegraded := isUnexpectedDowngradedModel(requestedModel, responseModel)

	downgraded := !hasHTML || isModelDegraded
	var reason string
	if isModelDegraded {
		reason = "模型被上游降级为: " + responseModel
	} else if !hasHTML {
		reason = "未生成完整的 HTML/SVG 动画（疑似降智或输出截断）"
	} else {
		reason = "测智通过（未降智，已生成完整动画）"
	}

	return PelicanResult{
		HasHTML:       hasHTML,
		HTML:          extracted,
		Downgraded:    downgraded,
		Reason:        reason,
		ResponseModel: responseModel,
	}
}

func isUnexpectedDowngradedModel(requested, actual string) bool {
	req := strings.ToLower(strings.TrimSpace(requested))
	act := strings.ToLower(strings.TrimSpace(actual))
	if req == "" || act == "" || req == act {
		return false
	}
	isHighTierReq := strings.Contains(req, "5") || strings.Contains(req, "6") || strings.Contains(req, "o1") || strings.Contains(req, "o3") || strings.Contains(req, "codex") || strings.Contains(req, "sonnet") || strings.Contains(req, "opus")
	isLowTierAct := strings.Contains(act, "mini") || strings.Contains(act, "nano") || strings.Contains(act, "haiku") || strings.Contains(act, "flash-lite") || strings.Contains(act, "3.5")
	return isHighTierReq && isLowTierAct
}

// createOpenAIPelicanProbePayload creates a Responses API payload tailored for the Pelican test.
func createOpenAIPelicanProbePayload(model string, isOAuth bool, prompt string, accountID int64, reasoningEffort ...string) map[string]any {
	sessionID := compactProbeSessionID(accountID)
	windowID := sessionID + ":0"
	installationID := deriveStableUUIDv4("sub2api:codex-pelican-installation:" + sessionID)
	turnID := deriveStableUUIDv4("sub2api:codex-pelican-turn:" + sessionID)
	turnMetadata := deriveStableUUIDv4("sub2api:codex-turn-meta:" + turnID)

	effort := "low"
	if len(reasoningEffort) > 0 && strings.TrimSpace(reasoningEffort[0]) != "" {
		effort = strings.TrimSpace(reasoningEffort[0])
	}

	payload := map[string]any{
		"model":        strings.TrimSpace(model),
		"instructions": DefaultPelicanInstructions,
		"input": []map[string]any{
			{
				"type": "message",
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": prompt,
					},
				},
			},
		},
		"reasoning": map[string]any{
			"effort":  effort,
			"summary": "auto",
		},
		"stream":           true,
		"prompt_cache_key": sessionID,
		"client_metadata": map[string]any{
			"x-codex-installation-id": installationID,
			"x-codex-window-id":       windowID,
			"x-codex-turn-metadata":   turnMetadata,
		},
	}
	if isOAuth {
		payload["store"] = false
	}
	return payload
}

func pelicanFinalTextFromResponse(response map[string]any) string {
	output, ok := response["output"].([]any)
	if !ok {
		return ""
	}
	var texts []string
	for _, item := range output {
		itemMap, ok := item.(map[string]any)
		if !ok || itemMap["type"] != "message" {
			continue
		}
		contents, ok := itemMap["content"].([]any)
		if !ok {
			continue
		}
		for _, part := range contents {
			partMap, ok := part.(map[string]any)
			if !ok {
				continue
			}
			partType, _ := partMap["type"].(string)
			switch partType {
			case "output_text":
				if text, ok := partMap["text"].(string); ok && text != "" {
					texts = append(texts, text)
				}
			case "refusal":
				if text, ok := partMap["refusal"].(string); ok && text != "" {
					texts = append(texts, text)
				}
			}
		}
	}
	return strings.Join(texts, "\n")
}
