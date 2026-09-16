package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractPelicanHTML(t *testing.T) {
	t.Run("extracts complete html document with doctype", func(t *testing.T) {
		html := "<!DOCTYPE html><html><body>鹈鹕骑自行车</body></html>"
		input := fmt.Sprintf("Here is the requested animation:\n```html\n%s\n```\nEnjoy!", html)
		extracted := ExtractPelicanHTML(input)
		assert.Equal(t, html, extracted)
	})

	t.Run("extracts standalone svg with boundary", func(t *testing.T) {
		svg := "<svg viewBox=\"0 0 100 100\"><circle cx=\"50\" cy=\"50\" r=\"40\"/></svg>"
		extracted := ExtractPelicanHTML(svg)
		assert.Equal(t, svg, extracted)
	})

	t.Run("extracts uppercase HTML tags", func(t *testing.T) {
		html := "<HTML><BODY>Pelican</BODY></HTML>"
		extracted := ExtractPelicanHTML(html)
		assert.Equal(t, html, extracted)
	})

	t.Run("returns empty when markdown fence is unclosed", func(t *testing.T) {
		input := "```html\n<html><body>incomplete"
		extracted := ExtractPelicanHTML(input)
		assert.Empty(t, extracted, "unclosed fence must not be accepted as finished")
	})

	t.Run("returns empty for plain text without document", func(t *testing.T) {
		input := "I cannot create an SVG animation for you right now."
		extracted := ExtractPelicanHTML(input)
		assert.Empty(t, extracted)
	})

	t.Run("returns empty for invalid tag boundary", func(t *testing.T) {
		input := "<htmlish>something</html>"
		extracted := ExtractPelicanHTML(input)
		assert.Empty(t, extracted)
	})
}

func TestEvaluatePelicanResult(t *testing.T) {
	t.Run("passes when full html is generated and models match", func(t *testing.T) {
		reply := "<!DOCTYPE html><html><body><svg><text>鹈鹕</text></svg></body></html>"
		res := EvaluatePelicanResult("gpt-5.4", "gpt-5.4", reply, "")
		assert.True(t, res.HasHTML)
		assert.False(t, res.Downgraded)
		assert.Contains(t, res.Reason, "未降智")
	})

	t.Run("detects downgrade when html is missing", func(t *testing.T) {
		reply := "Sorry, I am unable to draw a pelican."
		res := EvaluatePelicanResult("gpt-5.4", "gpt-5.4", reply, "")
		assert.False(t, res.HasHTML)
		assert.True(t, res.Downgraded)
		assert.Contains(t, res.Reason, "疑似降智")
	})

	t.Run("detects downgrade when upstream silently downgraded model family", func(t *testing.T) {
		reply := "<!DOCTYPE html><html><body><svg></svg></body></html>"
		res := EvaluatePelicanResult("gpt-5.4", "gpt-4o-mini", reply, "")
		assert.True(t, res.HasHTML)
		assert.True(t, res.Downgraded)
		assert.Contains(t, res.Reason, "模型被上游降级为: gpt-4o-mini")
	})

	t.Run("detects refusal", func(t *testing.T) {
		res := EvaluatePelicanResult("gpt-5.4", "gpt-5.4", "", "I cannot fulfill this request")
		assert.False(t, res.HasHTML)
		assert.True(t, res.Downgraded)
		assert.Contains(t, res.Reason, "模型拒绝响应")
	})
}

func TestCreateOpenAIPelicanProbePayload(t *testing.T) {
	payload := createOpenAIPelicanProbePayload("gpt-5.4", true, DefaultPelicanPrompt, 123)
	require.NotNil(t, payload)
	assert.Equal(t, "gpt-5.4", payload["model"])
	assert.Equal(t, DefaultPelicanInstructions, payload["instructions"])
	assert.Equal(t, false, payload["store"])
	assert.Equal(t, true, payload["stream"])
	assert.NotEmpty(t, payload["prompt_cache_key"])

	reasoning, ok := payload["reasoning"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "low", reasoning["effort"])

	customPayload := createOpenAIPelicanProbePayload("gpt-6-astra", true, DefaultPelicanPrompt, 123, "high")
	customReasoning, ok := customPayload["reasoning"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "high", customReasoning["effort"])

	meta, ok := payload["client_metadata"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, meta["x-codex-installation-id"])
}

func TestAccountTestService_TestAccountConnection_OpenAIPelican(t *testing.T) {
	account := Account{
		ID:          1,
		Name:        "openai-oauth",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
	}
	repo := &snapshotUpdateAccountRepo{
		stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}},
	}
	pelicanSSEResponse := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"<!DOCTYPE html><html><body><svg><text>Pelican Bike</text></svg></body></html>\"}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_pelican\",\"model\":\"gpt-5.4\",\"output\":[]}}\n\n"

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(pelicanSSEResponse)),
	}}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("POST", "/api/v1/admin/accounts/1/test", bytes.NewReader(nil))

	err := svc.TestAccountConnection(c, account.ID, "gpt-5.4", "", AccountTestModePelican)
	require.NoError(t, err)

	body := rec.Body.String()
	assert.Contains(t, body, "pelican_result")
	assert.Contains(t, body, "未降智")

	t.Run("defaults to DefaultPelicanModel when modelID is empty", func(t *testing.T) {
		upstream.resp = &http.Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(pelicanSSEResponse)),
		}
		rec2 := httptest.NewRecorder()
		c2, _ := gin.CreateTestContext(rec2)
		c2.Request = httptest.NewRequest("POST", "/api/v1/admin/accounts/1/test", bytes.NewReader(nil))

		err2 := svc.TestAccountConnection(c2, account.ID, "", "", AccountTestModePelican)
		require.NoError(t, err2)

		require.NotNil(t, upstream.lastBody)
		var sentBody map[string]any
		err := json.Unmarshal(upstream.lastBody, &sentBody)
		require.NoError(t, err)
		assert.Equal(t, DefaultPelicanModel, sentBody["model"])
	})
}

func TestAccountTestService_PelicanUsesSelectedModel(t *testing.T) {
	for _, accountType := range []string{AccountTypeAPIKey, AccountTypeOAuth} {
		for _, mappingKey := range []string{"gpt-6-astra", "*"} {
			for _, mode := range []string{AccountTestModePelican, ""} {
				t.Run(accountType+"/"+mappingKey+"/"+mode, func(t *testing.T) {
					account := &Account{
						ID: 95, Platform: PlatformOpenAI, Type: accountType, Concurrency: 1,
						Credentials: map[string]any{
							"api_key": "test-key", "access_token": "test-token",
							"model_mapping": map[string]any{mappingKey: "gpt-5.6-luna"},
						},
					}
					upstream := &httpUpstreamRecorder{resp: &http.Response{
						StatusCode: 200,
						Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
						Body:       io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\"}\n\n")),
					}}
					svc := &AccountTestService{httpUpstream: upstream, cfg: &config.Config{}}
					rec := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(rec)
					c.Request = httptest.NewRequest("POST", "/api/v1/admin/accounts/95/test", nil)
					require.NoError(t, svc.testOpenAIAccountConnection(c, account, "gpt-6-astra", "", mode))
					var sent map[string]any
					require.NoError(t, json.Unmarshal(upstream.lastBody, &sent))
					expected := "gpt-6-astra"
					if mode != AccountTestModePelican {
						expected = "gpt-5.6-luna"
					}
					assert.Equal(t, expected, sent["model"])
					assert.Contains(t, rec.Body.String(), "\"model\":\""+expected+"\"")
				})
			}
		}
	}
}
