package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type OpenAIClient struct {
	apiKey string
}

// AIClassification holds the AI's analysis of an email
type AIClassification struct {
	Category string `json:"category"` // PROBLEM, REQUEST, INQUIRY, FEEDBACK
	Priority string `json:"priority"` // LOW, MEDIUM, HIGH, URGENT
	Reply    string `json:"reply"`    // Contextual auto-reply body
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		apiKey: apiKey,
	}
}

// ClassifyAndReply classifies the email and generates a contextual reply in one API call
func (c *OpenAIClient) ClassifyAndReply(emailBody string, customerName string, companyName string, ticketNum string) (*AIClassification, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is missing")
	}

	prompt := fmt.Sprintf(`Kamu adalah AI helpdesk assistant yang cerdas untuk %s.
Seorang pelanggan bernama %s mengirim email. Tiket ID: %s.

Email pelanggan:
"""
%s
"""

TUGAS (harus kamu lakukan semua):

1. KLASIFIKASI email ini ke salah satu kategori:
   - PROBLEM: pelanggan melaporkan masalah/gangguan/error/kerusakan
   - REQUEST: pelanggan meminta layanan baru, perubahan, penambahan, atau tindakan tertentu
   - INQUIRY: pelanggan bertanya informasi, harga, prosedur, atau hal umum
   - FEEDBACK: pelanggan memberi feedback, ucapan terima kasih, atau saran

2. TENTUKAN prioritas berdasarkan urgency:
   - URGENT: layanan down, tidak bisa kerja, dampak besar
   - HIGH: gangguan signifikan tapi masih bisa kerja sebagian
   - MEDIUM: permintaan/masalah standar
   - LOW: informasi umum, feedback positif, tidak mendesak

3. TULIS reply email yang sesuai kategori:
   - PROBLEM → empati + troubleshooting awal + janji follow-up cepat
   - REQUEST → konfirmasi request diterima + estimasi proses
   - INQUIRY → jawab pertanyaan sebisa mungkin + arahkan jika perlu
   - FEEDBACK → apresiasi + tanya apakah ada hal lain yang bisa dibantu
   
   Aturan reply:
   - JANGAN tambahkan sapaan pembuka (seperti "Dear", "Halo").
   - JANGAN tambahkan salam penutup (seperti "Salam", "Terima kasih", "Best Regards").
   - Hanya tulis inti pesan balasan saja.
   - Gunakan bahasa SAMA dengan email pelanggan.
   - Gaya natural, ramah, helpful — BUKAN template kaku.
   - JANGAN sertakan subject line atau header email.

RESPOND HANYA dalam format JSON berikut (tanpa markdown code block, tanpa penjelasan tambahan):
{"category":"PROBLEM","priority":"HIGH","reply":"isi reply disini"}`, companyName, customerName, emailBody)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "Kamu adalah AI helpdesk classifier dan assistant. Selalu respond dalam format JSON yang valid. Tidak pernah menambahkan teks di luar JSON."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.4,
		"max_tokens":  700,
	})
	if err != nil {
		return nil, err
	}

	content, err := c.callOpenAI(requestBody)
	if err != nil {
		return nil, err
	}

	// Parse the JSON response
	classification, err := parseAIClassification(content)
	if err != nil {
		log.Printf("[AI] Failed to parse classification JSON, using reply as-is: %v", err)
		// Fallback: use the raw content as reply with defaults
		return &AIClassification{
			Category: "PROBLEM",
			Priority: "MEDIUM",
			Reply:    content,
		}, nil
	}

	// Validate category
	validCategories := map[string]bool{"PROBLEM": true, "REQUEST": true, "INQUIRY": true, "FEEDBACK": true}
	if !validCategories[classification.Category] {
		classification.Category = "PROBLEM"
	}

	// Validate priority
	validPriorities := map[string]bool{"LOW": true, "MEDIUM": true, "HIGH": true, "URGENT": true}
	if !validPriorities[classification.Priority] {
		classification.Priority = "MEDIUM"
	}

	log.Printf("[AI] Email classified: category=%s, priority=%s", classification.Category, classification.Priority)
	return classification, nil
}

// GenerateAutoReply is a backward-compatible wrapper that returns just the reply text
func (c *OpenAIClient) GenerateAutoReply(emailBody string, customerName string, companyName string, ticketNum string) (string, error) {
	classification, err := c.ClassifyAndReply(emailBody, customerName, companyName, ticketNum)
	if err != nil {
		return "", err
	}
	return classification.Reply, nil
}

// callOpenAI makes the API call and returns the content string
func (c *OpenAIClient) callOpenAI(requestBody []byte) (string, error) {
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

// parseAIClassification parses the AI response JSON into an AIClassification struct
func parseAIClassification(content string) (*AIClassification, error) {
	// Clean up the response - remove markdown code blocks if present
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var classification AIClassification
	if err := json.Unmarshal([]byte(content), &classification); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON: %w, raw content: %s", err, content)
	}

	if classification.Reply == "" {
		return nil, fmt.Errorf("empty reply in AI response")
	}

	return &classification, nil
}
