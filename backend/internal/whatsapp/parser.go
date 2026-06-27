package whatsapp

import (
	"regexp"
	"strings"
)

type ParsedAction struct {
	ActionType string
	TicketID   string
	Content    string
}

// Common helpdesk keywords that strongly signal the user is reporting an issue.
// If any of these appear in a directed message, we treat it as a ticket creation.
var issueKeywords = []string{
	// Indonesian
	"error", "down", "masalah", "gagal", "rusak", "mati", "lambat", "hang",
	"crash", "bug", "gangguan", "kendala", "trouble", "bermasalah", "tidak bisa",
	"gak bisa", "ga bisa", "nggak bisa", "ngga bisa", "gabisa", "tdk bisa",
	"tolong", "bantuin", "bantu", "minta tolong", "urgent", "darurat", "penting",
	"nge-lag", "lag", "stuck", "freeze", "pending", "timeout", "timed out",
	"tidak jalan", "gak jalan", "ga jalan", "tidak berfungsi", "tidak bekerja",
	"500", "404", "502", "503", "connection refused", "connection timeout",
	// English
	"broken", "failed", "failure", "issue", "problem", "not working",
	"can't access", "unable to", "outage", "offline", "unavailable",
}

// ParseMessage parses the WhatsApp message body to determine the action.
// isDirectedToBot is true for private messages, or group messages that mention the bot.
func ParseMessage(body string, isDirectedToBot bool) *ParsedAction {
	body = strings.TrimSpace(body)

	// Strip mentions (e.g. @628123456789 or @Helpdesk IDE)
	reMention := regexp.MustCompile(`(?i)@(?:helpdesk(?:\s+ide)?|\d+)\s*`)
	cleanBody := reMention.ReplaceAllString(body, "")
	cleanBody = strings.TrimSpace(cleanBody)

	// Check for ticket progress update: "2026-06-001 progress: ..." or "IDE-2606-14 update: ..."
	if matched, _ := regexp.MatchString(`(?i)(?:\d{4}-\d{2}-\d{3,}|IDE-\d{4}-\d{2,3})\s+(progress|update):`, cleanBody); matched {
		re := regexp.MustCompile(`(?i)((?:\d{4}-\d{2}-\d{3,}|IDE-\d{4}-\d{2,3}))\s+(progress|update):\s*(.+)`)
		matches := re.FindStringSubmatch(cleanBody)
		if len(matches) > 2 {
			return &ParsedAction{
				ActionType: "update",
				TicketID:   matches[1],
				Content:    strings.TrimSpace(matches[3]),
			}
		}
	}

	// Check for close ticket: "2026-06-001 close: ..." or "IDE-2606-14 close: ..."
	if matched, _ := regexp.MatchString(`(?i)(?:\d{4}-\d{2}-\d{3,}|IDE-\d{4}-\d{2,3})\s+close:`, cleanBody); matched {
		re := regexp.MustCompile(`(?i)((?:\d{4}-\d{2}-\d{3,}|IDE-\d{4}-\d{2,3}))\s+close:\s*(.+)`)
		matches := re.FindStringSubmatch(cleanBody)
		if len(matches) > 2 {
			return &ParsedAction{
				ActionType: "close",
				TicketID:   matches[1],
				Content:    strings.TrimSpace(matches[2]),
			}
		}
	}

	// Check for reopen ticket: "2026-06-001 reopen" or "IDE-2606-14 reopen"
	if matched, _ := regexp.MatchString(`(?i)(?:\d{4}-\d{2}-\d{3,}|IDE-\d{4}-\d{2,3})\s+reopen`, cleanBody); matched {
		re := regexp.MustCompile(`(?i)((?:\d{4}-\d{2}-\d{3,}|IDE-\d{4}-\d{2,3}))\s+reopen`)
		matches := re.FindStringSubmatch(cleanBody)
		if len(matches) > 1 {
			return &ParsedAction{
				ActionType: "reopen",
				TicketID:   matches[1],
				Content:    "",
			}
		}
	}

	// Check for status check: "status 2026-06-001" or "IDE-2606-14 status"
	if matched, _ := regexp.MatchString(`(?i)(status\s+(?:\d{4}-\d{2}-\d{3,}|IDE-\d{4}-\d{2,3})|(?:\d{4}-\d{2}-\d{3,}|IDE-\d{4}-\d{2,3})\s+status)`, cleanBody); matched {
		re := regexp.MustCompile(`(?i)((?:\d{4}-\d{2}-\d{3,}|IDE-\d{4}-\d{2,3}))`)
		matches := re.FindStringSubmatch(cleanBody)
		if len(matches) > 1 {
			return &ParsedAction{
				ActionType: "status_check",
				TicketID:   matches[1],
				Content:    "",
			}
		}
	}

	// Check for explicit create ticket pattern: "tolong buatin tiket ..."
	if matched, _ := regexp.MatchString(`(?i)tolong\s+buat(?:in|kan)\s+tiket`, cleanBody); matched {
		re := regexp.MustCompile(`(?i)tolong\s+buat(?:in|kan)\s+tiket\s*(.+)`)
		matches := re.FindStringSubmatch(cleanBody)
		if len(matches) > 1 {
			return &ParsedAction{
				ActionType: "create_ticket",
				Content:    strings.TrimSpace(matches[1]),
			}
		}
	}

	// Keyword-based issue detection for directed messages
	// If the message is directed to the bot and contains common support keywords,
	// treat it as a ticket creation even without the explicit "tolong buatin tiket" phrase.
	if isDirectedToBot && containsIssueKeyword(cleanBody) {
		return &ParsedAction{
			ActionType: "create_ticket",
			Content:    cleanBody,
		}
	}

	// If directed to bot and no command matched, treat as create ticket with full body.
	// This is the ultimate fallback: any private message or @mention with text → ticket.
	if isDirectedToBot && cleanBody != "" {
		return &ParsedAction{
			ActionType: "create_ticket",
			Content:    cleanBody,
		}
	}

	// No action recognized
	return nil
}

// containsIssueKeyword checks if the message body contains any known issue/support keyword
func containsIssueKeyword(body string) bool {
	lower := strings.ToLower(body)
	for _, kw := range issueKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

