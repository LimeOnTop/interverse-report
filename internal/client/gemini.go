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

const defaultGeminiModel = "gemini-flash-latest"

type GeminiAnalysis struct {
	OverallScore      int    `json:"overall_score"`
	AlgorithmScore    int    `json:"algorithm_score"`
	ArchitectureScore int    `json:"architecture_score"`
	CodingScore       int    `json:"coding_score"`
	SoftSkillsScore   int    `json:"soft_skills_score"`
	Comments          string `json:"comments"`
	Strengths         string `json:"strengths"`
	Weaknesses        string `json:"weaknesses"`
	Recommendations   string `json:"recommendations"`
}

type GeminiClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = defaultGeminiModel
	}

	return &GeminiClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (c *GeminiClient) Analyze(ctx context.Context, prompt string) (GeminiAnalysis, error) {
	if c.apiKey == "" {
		return GeminiAnalysis{}, fmt.Errorf("gemini api key is not configured")
	}

	payload := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]string{
			"responseMimeType": "application/json",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return GeminiAnalysis{}, fmt.Errorf("marshal gemini request: %w", err)
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent",
		c.model,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return GeminiAnalysis{}, fmt.Errorf("create gemini request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GeminiAnalysis{}, fmt.Errorf("call gemini api: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return GeminiAnalysis{}, fmt.Errorf("read gemini response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return GeminiAnalysis{}, fmt.Errorf("gemini api status %d: %s", resp.StatusCode, string(respBody))
	}

	text, err := extractGeminiText(respBody)
	if err != nil {
		return GeminiAnalysis{}, err
	}

	var analysis GeminiAnalysis
	if err := json.Unmarshal([]byte(text), &analysis); err != nil {
		return GeminiAnalysis{}, fmt.Errorf("parse gemini json: %w; raw=%s", err, text)
	}

	normalizeScores(&analysis)
	return analysis, nil
}

func extractGeminiText(body []byte) (string, error) {
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("parse gemini response envelope: %w", err)
	}

	if parsed.Error.Message != "" {
		return "", fmt.Errorf("gemini error: %s", parsed.Error.Message)
	}

	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned empty response")
	}

	text := strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text)
	if text == "" {
		return "", fmt.Errorf("gemini returned empty text")
	}

	return text, nil
}

func normalizeScores(analysis *GeminiAnalysis) {
	analysis.OverallScore = clampScore(analysis.OverallScore)
	analysis.AlgorithmScore = clampScore(analysis.AlgorithmScore)
	analysis.ArchitectureScore = clampScore(analysis.ArchitectureScore)
	analysis.CodingScore = clampScore(analysis.CodingScore)
	analysis.SoftSkillsScore = clampScore(analysis.SoftSkillsScore)
}

func clampScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
