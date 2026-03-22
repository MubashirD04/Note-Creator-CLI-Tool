package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Constants
const GroqAPIURL = "https://api.groq.com/openai/v1/chat/completions"
const DefaultModel = "llama-3.3-70b-versatile"

type GroqRequest struct {
	Model          string          `json:"model"`
	Messages       []GroqMessage   `json:"messages"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type GroqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ResponseFormat struct {
	Type string `json:"type"`
}

type GroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type GroqClient struct {
	APIKey string
}

func NewGroqClient(apiKey string) *GroqClient {
	return &GroqClient{APIKey: apiKey}
}

func (c *GroqClient) GenerateNotes(transcript string) (string, error) {
	systemPrompt := `You are an expert technical assistant. Your task is to output structured JSON notes based *only* on the provided lecture transcript. Do not add any conversational text.

You MUST respond strictly in pure JSON format, matching this exact structure:
{
  "summary": "A brief 2-3 sentence overview of the lecture.",
  "key_concepts": [
    {
      "term": "Name of the concept",
      "definition": "Clear, concise definition of this concept based on the transcript"
    }
  ],
  "detailed_notes": "Detailed explanations outlining the core ideas...",
  "code_examples": [
    "Example code block 1"
  ],
  "action_items": [
    "Action item 1"
  ]
}`

	reqBody := GroqRequest{
		Model: DefaultModel,
		Messages: []GroqMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: transcript},
		},
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", GroqAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var groqResp GroqResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", err
	}

	if len(groqResp.Choices) == 0 {
		return "", errors.New("no completion choices returned from Groq")
	}

	return groqResp.Choices[0].Message.Content, nil
}
