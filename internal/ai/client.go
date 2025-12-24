package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/zhangdongyu29-blip/mchat/internal/config"
)

const defaultModel = "xiaomi/mimo-v2-flash:free"

// Message represents an OpenRouter message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionResponse holds minimal response fields.
type CompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Chat calls the OpenRouter API.
func Chat(ctx context.Context, cfg config.Config, messages []Message) (string, error) {
	if cfg.OpenRouterAPIKey == "" {
		return "", errors.New("OPENROUTER_API_KEY 未配置")
	}

	body := map[string]any{
		"model":    defaultModel,
		"messages": messages,
		"reasoning": map[string]bool{
			"enabled": false,
		},
	}

	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.OpenRouterURL, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.OpenRouterAPIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/zhangdongyu29-blip/mchat")
	req.Header.Set("X-Title", "mchat")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Error != nil {
		return "", errors.New(out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}
	return out.Choices[0].Message.Content, nil
}
