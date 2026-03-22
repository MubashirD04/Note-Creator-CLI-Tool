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
	Model  string
}

func NewGroqClient(apiKey, model string) *GroqClient {
	if model == "" {
		model = DefaultModel
	}
	return &GroqClient{APIKey: apiKey, Model: model}
}

func (c *GroqClient) GenerateNotes(transcript string) (string, error) {
	systemPrompt := `You are an expert technical assistant. Your task is to output highly detailed and structured JSON notes based *only* on the provided lecture transcript. 

You MUST respond strictly in pure JSON format, matching this exact structure:
{
  "summary": "A concise 1-2 sentence high-level overview of the primary topic covered.",
  "key_concepts": [
    {
      "term": "Technical term, Annotation, or Pattern (e.g., @Service, POJO, Facade)",
      "definition": "An exhaustive explanation of what this is and how it was used in the specific context of the lecture."
    }
  ],
  "detailed_notes": "An extremely thorough, granular, and chronological breakdown of the lecture. This section should be exhaustive, capturing step-by-step implementation details, architectural 'whys', pros/cons, comparisons, and any pro-tips or warnings from the instructor. This MUST be the most extensive part of the JSON.",
  "code_examples": [
    "Clean, formatted code snippets or configuration examples mentioned in the transcript."
  ],
  "action_items": [
    "Specific tasks, next steps, or exercises mentioned for the user to perform."
  ]
}

CRITICAL CONSTRAINTS:
1. Provide a minimum of 4 distinct key_concepts. If there are fewer than 4 obvious terms, expand on related architectural patterns or internal logic mentioned.
2. The detailed_notes must be dense and informative, avoiding broad generalizations in favor of specific details found in the transcript.`

	reqBody := GroqRequest{
		Model: c.Model,
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
