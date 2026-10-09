package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LimeOnTop/interverse-report/internal/usecase"
	"golang.org/x/net/proxy"
)

const defaultGeminiModel = "gemini-flash-latest"

type GeminiAnalysis = usecase.GeminiAnalysis

type GeminiClient struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewGeminiClient(apiKey, model, proxyURL, baseURL string) *GeminiClient {
	if model == "" {
		model = defaultGeminiModel
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}

	return &GeminiClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   90 * time.Second,
			Transport: newGeminiTransport(proxyURL),
		},
	}
}

func newGeminiTransport(proxyURL string) http.RoundTripper {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		base = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}
	transport := base.Clone()

	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return transport
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return transport
	}

	switch strings.ToLower(parsed.Scheme) {
	case "socks5", "socks5h":
		dialer, err := proxy.FromURL(parsed, proxy.Direct)
		if err != nil {
			return transport
		}
		if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = contextDialer.DialContext
		} else {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		}
		transport.Proxy = nil
	default:
		transport.Proxy = http.ProxyURL(parsed)
	}

	return transport
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
		"%s/v1beta/models/%s:generateContent",
		c.baseURL,
		c.model,
	)

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		analysis, err := c.analyzeOnce(ctx, url, body)
		if err == nil {
			return analysis, nil
		}

		lastErr = err
		if attempt == 3 || !isRetryableGeminiError(err) {
			break
		}

		select {
		case <-ctx.Done():
			return GeminiAnalysis{}, ctx.Err()
		case <-time.After(time.Duration(attempt) * 2 * time.Second):
		}
	}

	return GeminiAnalysis{}, lastErr
}

func (c *GeminiClient) analyzeOnce(ctx context.Context, url string, body []byte) (GeminiAnalysis, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return GeminiAnalysis{}, fmt.Errorf("create gemini request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-goog-api-key", c.apiKey)
	req.Header.Set("User-Agent", "interverse-report/1.0")

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

	text = sanitizeJSON(text)

	var analysis GeminiAnalysis
	if err := json.Unmarshal([]byte(text), &analysis); err != nil {
		return GeminiAnalysis{}, fmt.Errorf("parse gemini json: %w; raw=%s", err, text)
	}

	normalizeScores(&analysis)
	return analysis, nil
}

func isRetryableGeminiError(err error) bool {
	if err == nil {
		return false
	}

	message := err.Error()
	return strings.Contains(message, "503") ||
		strings.Contains(message, "429") ||
		strings.Contains(message, "500") ||
		strings.Contains(message, "502") ||
		strings.Contains(message, "504")
}

func sanitizeJSON(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
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
