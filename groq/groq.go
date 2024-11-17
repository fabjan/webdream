// Copyright 2024 Fabian Bergström
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

var apiKey string

func init() {
	apiKey = os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		panic("GROQ_API_KEY not set")
	}
}

type GroqChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GroqResponseFormat struct {
	Type string `json:"type"`
}

type GroqRequest struct {
	Model          string             `json:"model"`
	Messages       []GroqChatMessage  `json:"messages"`
	ResponseFormat GroqResponseFormat `json:"response_format"`
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

type GroqError struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	}
}

func Dream(reqPath string) (*dream.Response, error) {

	reqData := GroqRequest{
		Model: "llama3-8b-8192",
		ResponseFormat: GroqResponseFormat{
			Type: "json_object",
		},
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
	if resp.StatusCode == http.StatusUnauthorized {
		panic("Unauthorized, fix your API key!")
	}
	if resp.StatusCode != http.StatusOK {
		var respErr GroqError
		err = json.NewDecoder(resp.Body).Decode(&respErr)
		if err == nil {
			code := respErr.Error.Code
			message := respErr.Error.Message
			typ := respErr.Error.Type
			slog.Error("Groq error", "code", code, "message", message, "type", typ)
			return nil, fmt.Errorf("Groq error: %s %s", code, typ)
		}
		return nil, fmt.Errorf("Unexpected status in API response: %d", resp.StatusCode)
	}

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
	var result dream.Response
	for _, choice := range respData.Choices {
		err = json.Unmarshal([]byte(choice.Message.Content), &result)
		if err == nil {
			break
		}
	}
	if err != nil {
		slog.Error("Failed to decode LLM generated JSON",
			"err", err,
			"choices", len(respData.Choices),
		)
		return nil, fmt.Errorf("Failed to decode LLM generated JSON: %w", err)
	}

	return &result, nil
}
