package client

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Analyzer is the shared LLM contract used by report analysis.
type Analyzer interface {
	Analyze(ctx context.Context, prompt string) (GeminiAnalysis, error)
}

const defaultPrimaryAttempts = 3

// DefaultPrimaryAttempts is the number of DeepSeek tries before Gemini failover.
func DefaultPrimaryAttempts() int {
	return defaultPrimaryAttempts
}

// FallbackAnalyzer tries primary several times, then secondary.
type FallbackAnalyzer struct {
	Primary         Analyzer
	Secondary       Analyzer
	NamePrim        string
	NameSec         string
	PrimaryAttempts int
}

func (a *FallbackAnalyzer) Analyze(ctx context.Context, prompt string) (GeminiAnalysis, error) {
	if a.Primary == nil && a.Secondary == nil {
		return GeminiAnalysis{}, fmt.Errorf("no llm analyzer configured")
	}

	attempts := a.PrimaryAttempts
	if attempts <= 0 {
		attempts = defaultPrimaryAttempts
	}

	var lastErr error
	if a.Primary != nil {
		for attempt := 1; attempt <= attempts; attempt++ {
			out, err := a.Primary.Analyze(ctx, prompt)
			if err == nil {
				if attempt > 1 {
					log.Printf("%s analysis succeeded on attempt %d/%d", a.NamePrim, attempt, attempts)
				}
				return out, nil
			}
			lastErr = err
			log.Printf("%s analysis failed (attempt %d/%d): %v", a.NamePrim, attempt, attempts, err)

			if attempt == attempts {
				break
			}
			if ctx.Err() != nil {
				return GeminiAnalysis{}, ctx.Err()
			}
			select {
			case <-ctx.Done():
				return GeminiAnalysis{}, ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
		}
	}

	if a.Secondary == nil {
		if lastErr != nil {
			return GeminiAnalysis{}, lastErr
		}
		return GeminiAnalysis{}, fmt.Errorf("primary llm unavailable and no secondary configured")
	}

	log.Printf("failing over to %s after %d %s attempt(s)", a.NameSec, attempts, a.NamePrim)
	out, err := a.Secondary.Analyze(ctx, prompt)
	if err != nil {
		log.Printf("%s analysis failed: %v", a.NameSec, err)
		if lastErr != nil {
			return GeminiAnalysis{}, fmt.Errorf("%s failed after retries (%v); %s also failed: %w", a.NamePrim, lastErr, a.NameSec, err)
		}
	}
	return out, err
}
