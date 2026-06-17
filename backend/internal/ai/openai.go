package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAIClient struct {
	apiKey string
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		apiKey: apiKey,
	}
}

func (c *OpenAIClient) GenerateAutoReply(emailBody string, customerName string, companyName string, ticketID uint) (string, error) {
	if c.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key is missing")
	}

	prompt := fmt.Sprintf(`You are an AI assistant for the customer support team of %s.
A customer named %s has just sent an email to our support inbox. A ticket has been automatically created with ID: TK-%d.

Customer's email:
"""
%s
"""

Write a very polite, concise, and professional auto-reply email acknowledging their specific issue. 
It must be clear that this is an automated confirmation, but it should sound empathetic and refer to the context of their problem briefly. 
Include their ticket ID (TK-%d) and assure them that our team will look into it within 24 hours.

Do NOT include any subject line or email headers in your output, just the raw email body.`, companyName, customerName, ticketID, emailBody, ticketID)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model": "gpt-4o-mini", // or gpt-3.5-turbo if preferred
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful customer support AI."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens": 300,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI API error: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Choices) > 0 {
		return response.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no content returned from OpenAI")
}
