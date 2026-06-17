package whatsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type WahaClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

type SessionResponse struct {
	Session Session `json:"session"`
	Status  string  `json:"status"`
	Error   string  `json:"error"`
}

type Session struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Me          MeInfo `json:"me"`
	QR          QRInfo `json:"qr"`
	Connected   bool   `json:"connected"`
	PhoneNumber string `json:"phoneNumber"`
}

type MeInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type QRInfo struct {
	Code  string `json:"code"`
	URL   string `json:"url"`
	Error string `json:"error"`
}

type SendMessageRequest struct {
	ChatID  string `json:"chatId"`
	Text    string `json:"text"`
	Session string `json:"session"`
}

type SendMessageResponse struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
	Error     string `json:"error"`
}

type WebhookPayload struct {
	Event   string          `json:"event"`
	Session string          `json:"session"`
	Data    json.RawMessage `json:"data"`
}

type MessageEvent struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Body      string    `json:"body"`
	Timestamp int64     `json:"timestamp"`
	Type      string    `json:"type"`
	HasMedia  bool      `json:"hasMedia"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewWahaClient(baseURL string, apiKey string) *WahaClient {
	return &WahaClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (w *WahaClient) CreateSession(name string) error {
	url := fmt.Sprintf("%s/api/sessions/start", w.baseURL)
	payload := map[string]string{"name": name}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Waha error creating session: %s (status %d)", string(body), resp.StatusCode)
		return fmt.Errorf("waha api error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (w *WahaClient) GetSessionQR(sessionName string) (string, error) {
	url := fmt.Sprintf("%s/api/%s/auth/qr", w.baseURL, sessionName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		log.Printf("Error getting QR code: %v", err)
		return "", fmt.Errorf("failed to get qr code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Waha error: %s", string(body))
		return "", fmt.Errorf("waha api error: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if qr, ok := result["qr"].(string); ok {
		return qr, nil
	}
	if data, ok := result["data"].(string); ok {
		return data, nil
	}

	return "", fmt.Errorf("qr code not found in response")
}

func (w *WahaClient) RequestPairingCode(sessionName string, phoneNumber string) (string, error) {
	url := fmt.Sprintf("%s/api/%s/auth/request-code", w.baseURL, sessionName)
	payload := map[string]string{"phoneNumber": phoneNumber}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request pairing code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("waha api error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if code, ok := result["code"].(string); ok {
		return code, nil
	}
	return "", fmt.Errorf("pairing code not found in response")
}

func (w *WahaClient) SendMessage(sessionName, to, message string) (string, error) {
	url := fmt.Sprintf("%s/api/sendText", w.baseURL)
	payload := SendMessageRequest{
		ChatID:  to,
		Text:    message,
		Session: sessionName,
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return "", fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Waha error: %s", string(body))
		return "", fmt.Errorf("waha api error: status %d, body: %s, apiKeyLen: %d", resp.StatusCode, string(body), len(w.apiKey))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if msgID, ok := result["messageId"].(string); ok {
		return msgID, nil
	}

	return "", fmt.Errorf("message id not found in response")
}

func (w *WahaClient) GetSessions() ([]Session, error) {
	url := fmt.Sprintf("%s/api/sessions", w.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		log.Printf("Error getting sessions: %v", err)
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Waha error: %s", string(body))
		return nil, fmt.Errorf("waha api error: status %d", resp.StatusCode)
	}

	var sessions []Session
	if err := json.NewDecoder(resp.Body).Decode(&sessions); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return sessions, nil
}

func (w *WahaClient) DeleteSession(sessionName string) error {
	url := fmt.Sprintf("%s/api/sessions/%s", w.baseURL, sessionName)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		log.Printf("Error deleting session: %v", err)
		return fmt.Errorf("failed to delete session: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Waha error: %s", string(body))
		return fmt.Errorf("waha api error: status %d", resp.StatusCode)
	}

	return nil
}

func (w *WahaClient) CheckSessionStatus(sessionName string) (Session, error) {
	url := fmt.Sprintf("%s/api/sessions/%s", w.baseURL, sessionName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Session{}, fmt.Errorf("failed to create request: %w", err)
	}
	if w.apiKey != "" {
		req.Header.Set("X-Api-Key", w.apiKey)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		log.Printf("Error checking session status: %v", err)
		return Session{}, fmt.Errorf("failed to check session status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Session{}, fmt.Errorf("waha api error: status %d", resp.StatusCode)
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return Session{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return session, nil
}
