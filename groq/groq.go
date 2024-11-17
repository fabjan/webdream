package groq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/fabjan/webdream/dream"
	"github.com/fabjan/webdream/metrics"
)

type GroqChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqRequest struct {
	Model    string            `json:"model"`
	Messages []GroqChatMessage `json:"messages"`
}

type GroqResponse struct {
	Choices []struct {
		Message GroqChatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		QueueTime        float64 `json:"queue_time"`
		PromptTokens     int     `json:"prompt_tokens"`
		PromptTime       float64 `json:"prompt_time"`
		CompletionTokens int     `json:"completion_tokens"`
		CompletionTime   float64 `json:"completion_time"`
		TotalTokens      int     `json:"total_tokens"`
		TotalTime        float64 `json:"total_time"`
	} `json:"usage"`
}

func Dream(reqPath string) (*dream.Response, error) {
	apiKey := os.Getenv("GROQ_API_KEY")

	reqData := GroqRequest{
		Model: "llama3-8b-8192",
		Messages: []GroqChatMessage{
			{
				Role:    "system",
				Content: dream.SystemPrompt,
			},
			{
				Role: "user",
				Content: `{
					"headers": {
						"Content-Type": "text/html"
					},
					"path": "` + reqPath + `",
				}`,
			},
		},
	}

	bodyBytes := new(bytes.Buffer)
	json.NewEncoder(bodyBytes).Encode(reqData)

	chatCompletionsURL := "https://api.groq.com/openai/v1/chat/completions"
	req, err := http.NewRequest("POST", chatCompletionsURL, bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// read the response body
	var respData GroqResponse
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode response: %w", err)
	}

	slog.Debug("Received response",
		"path", reqPath,
		"choices", len(respData.Choices),
		"promptTokens", respData.Usage.PromptTokens,
		"completionTokens", respData.Usage.CompletionTokens,
		"totalTokens", respData.Usage.TotalTokens,
		"totalTime", respData.Usage.TotalTime,
	)
	metrics.RecordRequest()
	metrics.RecordTokens(respData.Usage.TotalTokens)

	if len(respData.Choices) == 0 {
		return nil, fmt.Errorf("No choices in response")
	}

	// parse the wrapped JSON inside
	firstChoice := respData.Choices[0].Message.Content
	var result dream.Response
	err = json.Unmarshal([]byte(firstChoice), &result)
	if err != nil {
		slog.Error("Failed to decode inner response",
			"err", err,
			"firstChoice", firstChoice,
		)

		return nil, fmt.Errorf("Failed to decode inner response: %w", err)
	}

	return &result, nil
}
