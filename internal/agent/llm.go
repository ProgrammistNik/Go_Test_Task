package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type OpenAIClient struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

type completionRequest struct {
	Model          string              `json:"model"`
	Messages       []completionMessage `json:"messages"`
	Temperature    int                 `json:"temperature"`
	ResponseFormat map[string]string   `json:"response_format"`
}

type completionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type completionResponse struct {
	Choices []struct {
		Message completionMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func NewOpenAIClient(baseURL, apiKey, model string) *OpenAIClient {
	return &OpenAIClient{
		baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, model: model,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (client *OpenAIClient) Complete(ctx context.Context, system, message string) (string, error) {
	payload := completionRequest{
		Model: client.model,
		Messages: []completionMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: message},
		},
		Temperature:    0,
		ResponseFormat: map[string]string{"type": "json_object"},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	response, err := client.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("llm request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	var result completionResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode llm response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		if result.Error != nil {
			return "", errors.New(result.Error.Message)
		}
		return "", fmt.Errorf("llm returned status %d", response.StatusCode)
	}
	if len(result.Choices) == 0 {
		return "", errors.New("llm returned no choices")
	}
	return result.Choices[0].Message.Content, nil
}
