package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"

	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

// usecase.JSONCompleter is the LLM call used to explain weak points (DeepSeek).

// weakPoint is a wrong question or a failed task with an explanation of the
// right answer. Stored in report notes; the gateway hides it from Basic users.
type weakPoint struct {
	StepID        string `json:"step_id"`
	ItemType      string `json:"item_type"`
	Label         string `json:"label"`
	Prompt        string `json:"prompt"`
	Technology    string `json:"technology,omitempty"`
	UserAnswer    string `json:"user_answer"`
	CorrectAnswer string `json:"correct_answer"`
	Explanation   string `json:"explanation,omitempty"`
	Topic         string `json:"topic"`
	SourceURL     string `json:"source_url"`
}

type weakPointVerdict struct {
	ID          string `json:"id"`
	IsWeak      *bool  `json:"is_weak"`
	Explanation string `json:"explanation"`
	Topic       string `json:"topic"`
}

const weakPointBatchSize = 6

// weakPointCandidates returns questions not answered correctly and all tasks;
// whether a task is weak is decided by the LLM (or by the coding score).
func weakPointCandidates(reviews []answerReviewItem) []answerReviewItem {
	out := make([]answerReviewItem, 0, len(reviews))
	for _, item := range reviews {
		if item.ItemType == "task" || item.IsCorrect == nil || !*item.IsCorrect {
			out = append(out, item)
		}
	}
	return out
}

// explainWeakPoints asks the LLM for explanations in parallel batches so a
// long list does not push report generation past the gateway timeout.
func explainWeakPoints(
	ctx context.Context,
	completer usecase.JSONCompleter,
	interview usecase.InterviewSummary,
	candidates []answerReviewItem,
) map[string]weakPointVerdict {
	verdicts := make(map[string]weakPointVerdict, len(candidates))
	if completer == nil || len(candidates) == 0 {
		return verdicts
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for start := 0; start < len(candidates); start += weakPointBatchSize {
		end := start + weakPointBatchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		batch := candidates[start:end]

		wg.Add(1)
		go func() {
			defer wg.Done()
			text, err := completer.CompleteJSON(ctx, weakPointsSystemPrompt, buildWeakPointsPrompt(interview, batch))
			if err != nil {
				log.Printf("weak points explanation failed: %v", err)
				return
			}
			var parsed struct {
				Items []weakPointVerdict `json:"items"`
			}
			if err := json.Unmarshal([]byte(text), &parsed); err != nil {
				log.Printf("weak points explanation: parse json: %v", err)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, verdict := range parsed.Items {
				verdicts[verdict.ID] = verdict
			}
		}()
	}
	wg.Wait()

	return verdicts
}

// buildWeakPoints merges candidates with LLM verdicts. Without a verdict a task
// counts as weak when it is unanswered or the coding section failed.
func buildWeakPoints(candidates []answerReviewItem, verdicts map[string]weakPointVerdict, codingPassed bool) []weakPoint {
	points := make([]weakPoint, 0, len(candidates))
	for _, item := range candidates {
		verdict, hasVerdict := verdicts[item.StepID]
		if item.ItemType == "task" {
			unanswered := strings.TrimSpace(item.UserAnswer) == "" || item.UserAnswer == "Нет ответа"
			weak := unanswered || !codingPassed
			if hasVerdict && verdict.IsWeak != nil && !unanswered {
				weak = *verdict.IsWeak
			}
			if !weak {
				continue
			}
		}

		topic := strings.TrimSpace(verdict.Topic)
		if topic == "" {
			topic = fallbackTopic(item)
		}

		points = append(points, weakPoint{
			StepID:        item.StepID,
			ItemType:      item.ItemType,
			Label:         item.Label,
			Prompt:        item.Prompt,
			Technology:    item.Technology,
			UserAnswer:    item.UserAnswer,
			CorrectAnswer: item.CorrectAnswer,
			Explanation:   strings.TrimSpace(verdict.Explanation),
			Topic:         topic,
			SourceURL:     habrSearchURL(topic),
		})
	}
	return points
}

func fallbackTopic(item answerReviewItem) string {
	words := strings.Fields(item.Prompt)
	if len(words) > 8 {
		words = words[:8]
	}
	topic := strings.Trim(strings.Join(words, " "), " ?.:,")
	if item.Technology != "" && !strings.Contains(strings.ToLower(topic), strings.ToLower(item.Technology)) {
		topic = strings.TrimSpace(item.Technology + " " + topic)
	}
	return topic
}

// habrSearchURL links to a Habr search instead of an article URL: LLMs make up
// article links, a search on a real topic always resolves.
func habrSearchURL(topic string) string {
	return "https://habr.com/ru/search/?q=" + url.QueryEscape(topic) + "&target_type=posts&order=relevance"
}

const weakPointsSystemPrompt = "You are a senior technical interviewer and mentor. Return ONLY valid JSON matching the schema requested by the user."

func buildWeakPointsPrompt(interview usecase.InterviewSummary, items []answerReviewItem) string {
	type promptItem struct {
		ID              string   `json:"id"`
		Type            string   `json:"type"`
		Technology      string   `json:"technology,omitempty"`
		Question        string   `json:"question"`
		Options         []string `json:"options,omitempty"`
		CandidateAnswer string   `json:"candidate_answer"`
		CorrectAnswer   string   `json:"correct_answer"`
	}

	payload := make([]promptItem, 0, len(items))
	for _, item := range items {
		payload = append(payload, promptItem{
			ID:              item.StepID,
			Type:            item.ItemType,
			Technology:      item.Technology,
			Question:        truncateRunes(item.Prompt, 1500),
			Options:         item.Options,
			CandidateAnswer: truncateRunes(item.UserAnswer, 2000),
			CorrectAnswer:   truncateRunes(item.CorrectAnswer, 2000),
		})
	}
	itemsJSON, _ := json.Marshal(payload)

	return fmt.Sprintf(`A candidate finished a %s training (level %s). Below are questions they answered wrong and practical tasks they solved.

For every item:
- "explanation": 2-4 sentences in Russian that explain why the correct answer is right and, if the candidate answered, what is wrong in their answer. Plain text, no markdown headings; short inline code only if essential.
- "is_weak": for "task" items decide whether the candidate's solution correctly solves the task compared with the reference (false = solved, true = not solved or incomplete). For "question" items always true.
- "topic": a 2-5 word search query naming the concept to study, the way Russian IT articles name it (for example "замыкания JavaScript", "индексы PostgreSQL", "горутины и каналы Go"). Never output URLs.

Return ONLY JSON in this schema:
{"items":[{"id":"<item id>","is_weak":true,"explanation":"...","topic":"..."}]}

Items:
%s`, interview.Specialization, interview.Level, string(itemsJSON))
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}
