package client

import (
	"context"
	"fmt"

	questionpb "github.com/LimeOnTop/interverse-contracts/question/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type QuestionOption struct {
	Text      string
	IsCorrect bool
	SortOrder int32
}

type QuestionDetails struct {
	ID         string
	Text       string
	Category   string
	Difficulty string
	Technology string
	Answer     string
	Options    []QuestionOption
}

type QuestionClient struct {
	client questionpb.QuestionServiceClient
}

func NewQuestionClient(questionServiceURL string) (*QuestionClient, error) {
	conn, err := grpc.NewClient(questionServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect to question service: %w", err)
	}

	return &QuestionClient{
		client: questionpb.NewQuestionServiceClient(conn),
	}, nil
}

func (c *QuestionClient) GetByID(ctx context.Context, questionID string) (QuestionDetails, error) {
	resp, err := c.client.GetQuestion(ctx, &questionpb.GetQuestionRequest{
		QuestionId: questionID,
	})
	if err != nil {
		return QuestionDetails{}, fmt.Errorf("get question: %w", err)
	}

	if resp.Response != nil && !resp.Response.Success {
		return QuestionDetails{}, fmt.Errorf("get question: %s", resp.Response.Error)
	}

	question := resp.GetQuestion()
	if question == nil {
		return QuestionDetails{}, fmt.Errorf("get question: empty response")
	}

	options := make([]QuestionOption, 0, len(question.GetOptions()))
	for _, option := range question.GetOptions() {
		options = append(options, QuestionOption{
			Text:      option.GetText(),
			IsCorrect: option.GetIsCorrect(),
			SortOrder: option.GetSortOrder(),
		})
	}

	return QuestionDetails{
		ID:         question.GetId(),
		Text:       question.GetText(),
		Category:   question.GetCategory(),
		Difficulty: question.GetDifficulty(),
		Technology: question.GetTechnology(),
		Answer:     question.GetAnswer(),
		Options:    options,
	}, nil
}
