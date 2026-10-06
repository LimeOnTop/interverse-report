package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultDeepSeekBaseURL = "https://api.deepseek.com"
	defaultDeepSeekModel   = "deepseek-chat"
)

// DeepSeekClient talks to DeepSeek's OpenAI-compatible Chat Completions API.
// Works from RU IPs (unlike Google Gemini).
type DeepSeekClient struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewDeepSeekClient(apiKey, model, baseURL string) *DeepSeekClient {
	if model == "" {
		model = defaultDeepSeekModel
	}
	if baseURL == "" {
		baseURL = defaultDeepSeekBaseURL
	}
	return &DeepSeekClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (c *DeepSeekClient) Enabled() bool {
	return c != nil && strings.TrimSpace(c.apiKey) != ""
}

func (c *DeepSeekClient) Analyze(ctx context.Context, prompt string) (GeminiAnalysis, error) {
	// Single attempt here — multi-attempt + Gemini failover is owned by FallbackAnalyzer.
	text, err := c.CompleteJSON(
		ctx,
		"You are a senior technical interviewer. Return ONLY valid JSON matching the schema requested by the user. Include the word JSON in the response schema.",
		prompt,
	)
	if err != nil {
		return GeminiAnalysis{}, err
	}

	var analysis GeminiAnalysis
	if err := json.Unmarshal([]byte(text), &analysis); err != nil {
		return GeminiAnalysis{}, fmt.Errorf("parse deepseek json: %w; raw=%s", err, text)
	}
	normalizeScores(&analysis)
	return analysis, nil
}

// CompleteJSON sends one chat request in JSON mode and returns the raw JSON text.
func (c *DeepSeekClient) CompleteJSON(ctx context.Context, system, prompt string) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("deepseek api key is not configured")
	}

	payload := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal deepseek request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create deepseek request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call deepseek api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read deepseek response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("deepseek api status %d: %s", resp.StatusCode, string(respBody))
	}

	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return "", fmt.Errorf("parse deepseek envelope: %w", err)
	}
	if envelope.Error != nil && envelope.Error.Message != "" {
		return "", fmt.Errorf("deepseek error: %s", envelope.Error.Message)
	}
	if len(envelope.Choices) == 0 || strings.TrimSpace(envelope.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("deepseek returned empty response")
	}

	return sanitizeJSON(envelope.Choices[0].Message.Content), nil
}
