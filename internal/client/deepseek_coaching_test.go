package client

import (
	"context"
	"encoding/json"
	"github.com/LimeOnTop/interverse-report/internal/usecase"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeepSeekSendsCoachingSystemMessage(t *testing.T) {
	var system string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct{ Role, Content string }
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if len(body.Messages) > 0 {
			system = body.Messages[0].Content
		}
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"comments":"Теория у тебя получается хорошо."}`}}}})
	}))
	defer server.Close()
	result, err := NewDeepSeekClient("fixture-key", "", server.URL).Analyze(context.Background(), "fixture prompt")
	if err != nil {
		t.Fatal(err)
	}
	if system != usecase.ReportCoachingInstructions || result.Comments != "Теория у тебя получается хорошо." {
		t.Fatalf("coaching policy not sent")
	}
}
