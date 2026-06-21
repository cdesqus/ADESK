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

	prompt := fmt.Sprintf(`Kamu adalah AI helpdesk assistant yang cerdas dan ramah untuk %s.
Seorang pelanggan bernama %s baru saja mengirim email ke support inbox kami. Tiket otomatis sudah dibuat dengan ID: TK-%d.

Email dari pelanggan:
"""
%s
"""

INSTRUKSI:
1. Baca dan pahami isi email pelanggan dengan seksama.
2. Berikan JAWABAN atau SOLUSI AWAL yang relevan terhadap pertanyaan/masalah mereka. Jika mereka menanyakan sesuatu, coba jawab sebisa mungkin berdasarkan konteks. Jika mereka melaporkan masalah teknis, berikan langkah troubleshooting awal.
3. Sebutkan nomor tiket TK-%d agar mereka bisa follow up.
4. Sampaikan bahwa tim teknis kami juga akan meninjau dan menindaklanjuti dalam 24 jam.
5. Gunakan bahasa yang SAMA dengan bahasa email pelanggan (jika email dalam Bahasa Indonesia, balas dalam Bahasa Indonesia; jika dalam Bahasa Inggris, balas dalam Bahasa Inggris).
6. Tulis dengan gaya natural, ramah, dan helpful — BUKAN seperti template otomatis yang kaku.
7. Jangan terlalu panjang, cukup to the point tapi informatif.
8. JANGAN sertakan subject line atau header email, hanya body email saja.
9. JANGAN gunakan frasa seperti "ini adalah pesan otomatis" atau "jangan balas email ini".`, companyName, customerName, ticketID, emailBody, ticketID)

	requestBody, err := json.Marshal(map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "Kamu adalah AI helpdesk assistant yang cerdas, ramah, dan solutif. Kamu bisa menjawab pertanyaan teknis, memberikan troubleshooting awal, dan membantu pelanggan dengan masalah mereka. Selalu responsif dan helpful."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens": 600,
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
