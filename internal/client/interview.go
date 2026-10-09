package client

import (
	"context"
	"fmt"
	"strconv"

	interviewpb "github.com/LimeOnTop/interverse-contracts/interview/gen"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type SessionItem = usecase.SessionItem

type InterviewSummary = usecase.InterviewSummary

type InterviewClient struct {
	client interviewpb.InterviewServiceClient
}

func NewInterviewClient(interviewServiceURL string) (*InterviewClient, error) {
	conn, err := grpc.NewClient(interviewServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect to interview service: %w", err)
	}

	return &InterviewClient{
		client: interviewpb.NewInterviewServiceClient(conn),
	}, nil
}

func (c *InterviewClient) GetSessionContent(ctx context.Context, interviewID string, userID int64) (InterviewSummary, []SessionItem, []SessionItem, error) {
	resp, err := c.client.GetSessionContent(ctx, &interviewpb.GetSessionContentRequest{
		InterviewId: interviewID,
		UserId:      strconv.FormatInt(userID, 10),
	})
	if err != nil {
		return InterviewSummary{}, nil, nil, fmt.Errorf("get session content: %w", err)
	}

	if resp.Response != nil && !resp.Response.Success {
		return InterviewSummary{}, nil, nil, fmt.Errorf("get session content: %s", resp.Response.Error)
	}

	interview := resp.GetInterview()
	if interview == nil {
		return InterviewSummary{}, nil, nil, fmt.Errorf("get session content: empty interview")
	}

	parsedUserID, err := strconv.ParseInt(interview.GetUserId(), 10, 64)
	if err != nil {
		return InterviewSummary{}, nil, nil, fmt.Errorf("get session content: invalid user id: %w", err)
	}

	summary := InterviewSummary{
		ID:             interview.GetId(),
		UserID:         parsedUserID,
		Title:          interview.GetTitle(),
		Description:    interview.GetDescription(),
		Status:         interview.GetStatus(),
		ScheduledAt:    interview.GetScheduledAt(),
		Level:          interview.GetLevel(),
		Specialization: interview.GetSpecialization(),
	}

	return summary, mapSessionItems(resp.GetQuestions()), mapSessionItems(resp.GetTasks()), nil
}

func (c *InterviewClient) CompleteInterview(ctx context.Context, interview InterviewSummary) error {
	resp, err := c.client.UpdateInterview(ctx, &interviewpb.UpdateInterviewRequest{
		InterviewId:    interview.ID,
		Title:          interview.Title,
		Description:    interview.Description,
		Status:         "completed",
		ScheduledAt:    interview.ScheduledAt,
		Level:          interview.Level,
		Specialization: interview.Specialization,
	})
	if err != nil {
		return fmt.Errorf("complete interview: %w", err)
	}

	if resp.Response != nil && !resp.Response.Success {
		return fmt.Errorf("complete interview: %s", resp.Response.Error)
	}

	return nil
}

func mapSessionItems(items []*interviewpb.SessionItem) []SessionItem {
	result := make([]SessionItem, 0, len(items))
	for _, item := range items {
		result = append(result, SessionItem{
			ID:         item.GetId(),
			QuestionID: item.GetQuestionId(),
			ItemType:   item.GetItemType(),
			SortOrder:  item.GetSortOrder(),
			Text:       item.GetText(),
			Technology: item.GetTechnology(),
			Difficulty: item.GetDifficulty(),
			Category:   item.GetCategory(),
			Options:    item.GetOptions(),
		})
	}
	return result
}

var _ usecase.InterviewReader = (*InterviewClient)(nil)
