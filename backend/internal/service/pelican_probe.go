package service

import (
	"fmt"
	"regexp"
	"strconv"
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
//  1. Inspect lines for Markdown code fences (```html, ```svg, ```xml, ```).
//     If a closed fence contains a document, extract it.
//     If a fence is unclosed at EOF, treat as incomplete and return empty string.
//  2. Search for <html>...</html> (prepending <!doctype html if present) or <svg>...</svg>.
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

// DefaultCandyPrompt is the capability probe appended to scheduled pelican
// runs: a bounded combinatorial worst-case question with a fixed numeric
// answer. Strong models and downgraded models differ most on multi-step
// reasoning, and the exact-answer format is objectively gradable. The leading
// preamble steers models away from brute-force enumeration toward the
// intended logical answer (operator-specified CandyExpectedAnswer).
const DefaultCandyPrompt = "只回答数字。这是一道逻辑推理题，不是穷举题：请直接依据题目的逻辑关系推理出唯一确定的结果，不要穷举、不要解释、不要分步计算，最终只输出一个数字。\n" +
	"在一个黑色的袋子里放有三种口味的糖果，每种糖果有两种不同的形状（圆形和五角星形，不同的形状靠手感可以分辨）：\n" +
	"辣椒味：圆形 7 个，五角星形 7 个（共 14 个）\n" +
	"空心菜味：圆形 9 个，五角星形 6 个（共 15 个）\n" +
	"金属味：圆形 8 个，五角星形 4 个（共 12 个，作为干扰项）\n" +
	"问题：为了保证能够拿到不同形状的辣椒味和空心菜味糖果（即：一个圆形辣椒味 + 一个五角星形空心菜味，或者一个五角星形辣椒味 + 一个圆形空心菜味），最少需要从袋子中取出多少颗糖果？"

// CandyExpectedAnswer is the required numeric answer for DefaultCandyPrompt.
// Operator-specified: the capability probe passes only when the model's reply
// evaluates to this number.
const CandyExpectedAnswer = 21

var candyNumberPattern = regexp.MustCompile(`-?\d+`)

// EvaluateCandyResult extracts the model's numeric answer from the reply and
// compares it with the configured expected answer. The last number in the
// reply is taken as the final answer (models put the conclusion last; the
// prompt demands a bare number).
func EvaluateCandyResult(reply string, expected int) (answer int, correct bool) {
	matches := candyNumberPattern.FindAllString(strings.ReplaceAll(reply, ",", ""), -1)
	if len(matches) == 0 {
		return 0, false
	}
	answer, err := strconv.Atoi(matches[len(matches)-1])
	if err != nil {
		return 0, false
	}
	return answer, answer == expected
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

// createOpenAIRuleProbePayload builds a Responses payload for a user-defined
// degradation rule: the rule prompt is sent verbatim with no instructions and
// no codex session metadata, so text-question rules are not steered toward
// HTML output.
func createOpenAIRuleProbePayload(model string, isOAuth bool, prompt string, accountID int64, reasoningEffort ...string) map[string]any {
	effort := "low"
	if len(reasoningEffort) > 0 && strings.TrimSpace(reasoningEffort[0]) != "" {
		effort = strings.TrimSpace(reasoningEffort[0])
	}
	payload := map[string]any{
		"model": strings.TrimSpace(model),
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
		"stream":           true,
		"prompt_cache_key": compactProbeSessionID(accountID),
	}
	if effort != "" && effort != "none" {
		// Request the readable reasoning summary so a failed answer keeps the
		// model's thinking for later inspection; OpenAI models omit it otherwise.
		payload["reasoning"] = map[string]any{"effort": effort, "summary": "auto"}
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

// Only animation prompts default to HTML validation. Other custom prompts need
// an explicit answer key before their correctness can be judged.
func resolveDirectProbeEvaluator(prompt, evaluator string) string {
	switch evaluator {
	case "html", "reply", "numeric":
		return evaluator
	}
	lower := strings.ToLower(strings.TrimSpace(prompt))
	if lower == "" || strings.Contains(lower, "html") || strings.Contains(lower, "svg") || (strings.Contains(lower, "鹈鹕") && (strings.Contains(lower, "绘制") || strings.Contains(lower, "动画"))) {
		return "html"
	}
	return "reply"
}

func evaluateDirectProbeResult(opts AccountTestOptions, requestedModel, responseModel, reply, refusal string) PelicanResult {
	if opts.Evaluator == "" || opts.Evaluator == "html" {
		return EvaluatePelicanResult(requestedModel, responseModel, reply, refusal)
	}
	result := PelicanResult{ResponseModel: responseModel, Reason: "回复完整（未配置答案判定，尚未校验正确性）"}
	switch {
	case strings.TrimSpace(refusal) != "":
		result.Downgraded = true
		result.Reason = "模型拒绝响应: " + strings.TrimSpace(refusal)
	case strings.TrimSpace(reply) == "":
		result.Downgraded = true
		result.Reason = "上游返回空回复"
	case isUnexpectedDowngradedModel(requestedModel, responseModel):
		result.Downgraded = true
		result.Reason = "模型被上游降级为: " + responseModel
	case opts.Evaluator == "numeric":
		if opts.ExpectedAnswer == nil {
			result.Reason = "未配置预期数字答案，尚未校验正确性"
			break
		}
		answer, correct := EvaluateCandyResult(reply, *opts.ExpectedAnswer)
		result.Downgraded = !correct
		if correct {
			result.Reason = "数字答案判定通过"
		} else {
			result.Reason = fmt.Sprintf("数字答案不符：期望 %d，实际 %s", *opts.ExpectedAnswer, formatCandyAnswer(answer))
		}
	}
	return result
}
