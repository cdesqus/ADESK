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
	cache  *SemanticCache
}

// AIClassification holds the AI's analysis of an email
type AIClassification struct {
	Category string `json:"category"` // PROBLEM, REQUEST, INQUIRY, FEEDBACK
	Priority string `json:"priority"` // LOW, MEDIUM, HIGH, URGENT
	Reply    string `json:"reply"`    // Contextual auto-reply body
}

// WhatsAppAIResponse holds the AI's analysis of a WhatsApp message
type WhatsAppAIResponse struct {
	ActionType   string `json:"action_type"`   // create_ticket, update, close, status_check
	TicketID     string `json:"ticket_id"`     // empty if create_ticket
	Content      string `json:"content"`       // the extracted problem description or update message
	NaturalReply string `json:"natural_reply"` // friendly reply to be sent back
}

func NewOpenAIClient(apiKey string, cache *SemanticCache) *OpenAIClient {
	return &OpenAIClient{
		apiKey: apiKey,
		cache:  cache,
	}
}

// ParseWhatsAppMessage extracts action and generates a conversational response using OpenAI.
// It checks the semantic cache first to avoid redundant API calls for similar messages.
func (c *OpenAIClient) ParseWhatsAppMessage(message string) (*WhatsAppAIResponse, error) {
	// 1. Check semantic cache first
	if c.cache != nil {
		if cached, hit := c.cache.Get(message); hit {
			log.Printf("[AI WA] Cache HIT — action=%s", cached.ActionType)
			return cached, nil
		}
	}

	// 2. Check API key
	if c.apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is missing")
	}

	prompt := fmt.Sprintf(`Kamu adalah agen Helpdesk WhatsApp. Seseorang mengirim pesan ke bot kamu.
Pesan: "%s"

TUGAS:
1. Tentukan action_type dari pesan tersebut. Pilih salah satu:
   - "create_ticket": Jika pengguna melaporkan masalah atau meminta dibuatkan tiket baru.
   - "update": Jika pengguna memberikan informasi tambahan/update pada tiket yang sudah ada (harus ada nomor tiket di pesannya atau tersirat).
   - "status_check": Jika pengguna menanyakan status dari tiketnya.
   - "close": Jika pengguna meminta tiket ditutup atau masalah sudah selesai.
2. Jika pesan merujuk ke tiket tertentu (misalnya ada teks seperti "2026-06-001"), ekstrak nomor tiket tersebut ke "ticket_id". Jika tidak ada, kosongkan ("").
3. Ekstrak inti laporan atau pesan ke "content". Bersihkan dari sapaan (seperti "tolong", "min", "@Helpdesk").
4. Buat balasan "natural_reply" yang SANGAT ramah, empatik, menggunakan bahasa Indonesia santai tapi profesional, seperti layaknya CS WhatsApp manusia. Jangan gunakan markdown tebal/miring kecuali untuk nomor tiket.
   Contoh reply create_ticket: "Halo! Siap, tiketnya sudah saya buatkan ya. Tim kami akan segera mengecek masalah ini. Mohon ditunggu! 🛠️"

RESPOND HANYA dalam format JSON berikut (tanpa markdown code block):
{"action_type":"create_ticket","ticket_id":"","content":"pesan intinya saja","natural_reply":"Halo!..."}`, message)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "Kamu adalah AI WhatsApp helpdesk parser. Selalu respond dengan valid JSON tanpa markdown."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  300,
	})
	if err != nil {
		return nil, err
	}

	// 3. Call OpenAI
	content, err := c.callOpenAI(requestBody)
	if err != nil {
		return nil, err
	}

	// Clean up JSON response
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var waResp WhatsAppAIResponse
	if err := json.Unmarshal([]byte(content), &waResp); err != nil {
		return nil, fmt.Errorf("failed to parse WhatsApp AI JSON: %w", err)
	}

	// Fallback/normalization
	if waResp.ActionType == "" {
		waResp.ActionType = "create_ticket"
	}
	if waResp.NaturalReply == "" {
		waResp.NaturalReply = "Siap, pesan sudah kami terima. Tim kami akan segera menindaklanjutinya!"
	}

	log.Printf("[AI WA] Action: %s, TicketID: %s", waResp.ActionType, waResp.TicketID)

	// 4. Store in semantic cache
	if c.cache != nil {
		c.cache.Set(message, &waResp)
	}

	return &waResp, nil
}

// GenerateFallbackResponse creates a sensible default response when both AI and regex parsing fail.
// This ensures the customer ALWAYS gets a reply, even during OpenAI outages.
func GenerateFallbackResponse(message string) *WhatsAppAIResponse {
	return &WhatsAppAIResponse{
		ActionType:   "create_ticket",
		TicketID:     "",
		Content:      message,
		NaturalReply: "Halo! Pesan kamu sudah kami terima dan tiket sudah dibuatkan. Tim support kami akan segera menghubungi kamu. Terima kasih sudah menghubungi kami! 🙏",
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
   - Tambahkan sapaan pembuka yang profesional dan sopan (seperti "Dear", "Halo Bapak/Ibu").
   - JANGAN tambahkan salam penutup (seperti "Salam", "Terima kasih", "Best Regards").
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
