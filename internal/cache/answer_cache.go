package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/LimeOnTop/interverse-report/internal/usecase"
	"github.com/redis/go-redis/v9"
)

const (
	answersKeyPrefix = "interview:answers:"
	answersTTL       = 7 * 24 * time.Hour
)

type storedAnswer struct {
	StepID              string `json:"step_id"`
	QuestionID          string `json:"question_id"`
	ItemType            string `json:"item_type"`
	SelectedOptionIndex *int32 `json:"selected_option_index,omitempty"`
	TaskAnswer          string `json:"task_answer,omitempty"`
}

type AnswerCache struct {
	client *redis.Client
}

func NewAnswerCache(client *redis.Client) *AnswerCache {
	return &AnswerCache{client: client}
}

func answersKey(interviewID string) string {
	return answersKeyPrefix + interviewID
}

func (c *AnswerCache) Save(ctx context.Context, interviewID string, answers []usecase.AnswerInput) error {
	if interviewID == "" || len(answers) == 0 {
		return nil
	}

	payload := make([]storedAnswer, 0, len(answers))
	for _, answer := range answers {
		payload = append(payload, storedAnswer{
			StepID:              answer.StepID,
			QuestionID:          answer.QuestionID,
			ItemType:            answer.ItemType,
			SelectedOptionIndex: answer.SelectedOptionIndex,
			TaskAnswer:          answer.TaskAnswer,
		})
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal interview answers: %w", err)
	}

	if err := c.client.Set(ctx, answersKey(interviewID), raw, answersTTL).Err(); err != nil {
		return fmt.Errorf("save interview answers: %w", err)
	}

	return nil
}

func (c *AnswerCache) Load(ctx context.Context, interviewID string) ([]usecase.AnswerInput, error) {
	if interviewID == "" {
		return nil, fmt.Errorf("load interview answers: interview id is required")
	}

	raw, err := c.client.Get(ctx, answersKey(interviewID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("load interview answers: not found")
		}
		return nil, fmt.Errorf("load interview answers: %w", err)
	}

	var payload []storedAnswer
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal interview answers: %w", err)
	}

	answers := make([]usecase.AnswerInput, 0, len(payload))
	for _, item := range payload {
		answers = append(answers, usecase.AnswerInput{
			StepID:              item.StepID,
			QuestionID:          item.QuestionID,
			ItemType:            item.ItemType,
			SelectedOptionIndex: item.SelectedOptionIndex,
			TaskAnswer:          item.TaskAnswer,
		})
	}

	return answers, nil
}

var _ usecase.AnswerCache = (*AnswerCache)(nil)
