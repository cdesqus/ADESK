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

// ParseMessage parses the WhatsApp message body to determine the action.
// isDirectedToBot is true for private messages, or group messages that mention the bot.
func ParseMessage(body string, isDirectedToBot bool) *ParsedAction {
	body = strings.TrimSpace(body)

	// Strip mentions (e.g. @628123456789)
	reMention := regexp.MustCompile(`@\d+\s*`)
	cleanBody := reMention.ReplaceAllString(body, "")
	cleanBody = strings.TrimSpace(cleanBody)

	// Check for ticket progress update: "2026-06-001 progress: ..." or "2026-06-001 update: ..."
	if matched, _ := regexp.MatchString(`(?i)\d{4}-\d{2}-\d{3,}\s+(progress|update):`, cleanBody); matched {
		re := regexp.MustCompile(`(?i)(\d{4}-\d{2}-\d{3,})\s+(progress|update):\s*(.+)`)
		matches := re.FindStringSubmatch(cleanBody)
		if len(matches) > 3 {
			return &ParsedAction{
				ActionType: "update",
				TicketID:   matches[1],
				Content:    strings.TrimSpace(matches[3]),
			}
		}
	}

	// Check for close ticket: "2026-06-001 close: ..."
	if matched, _ := regexp.MatchString(`(?i)\d{4}-\d{2}-\d{3,}\s+close:`, cleanBody); matched {
		re := regexp.MustCompile(`(?i)(\d{4}-\d{2}-\d{3,})\s+close:\s*(.+)`)
		matches := re.FindStringSubmatch(cleanBody)
		if len(matches) > 2 {
			return &ParsedAction{
				ActionType: "close",
				TicketID:   matches[1],
				Content:    strings.TrimSpace(matches[2]),
			}
		}
	}

	// Check for reopen ticket: "2026-06-001 reopen"
	if matched, _ := regexp.MatchString(`(?i)\d{4}-\d{2}-\d{3,}\s+reopen`, cleanBody); matched {
		re := regexp.MustCompile(`(?i)(\d{4}-\d{2}-\d{3,})\s+reopen`)
		matches := re.FindStringSubmatch(cleanBody)
		if len(matches) > 1 {
			return &ParsedAction{
				ActionType: "reopen",
				TicketID:   matches[1],
				Content:    "",
			}
		}
	}

	// Check for status check: "status 2026-06-001" or "2026-06-001 status"
	if matched, _ := regexp.MatchString(`(?i)(status\s+\d{4}-\d{2}-\d{3,}|\d{4}-\d{2}-\d{3,}\s+status)`, cleanBody); matched {
		re := regexp.MustCompile(`(?i)(\d{4}-\d{2}-\d{3,})`)
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
	if matched, _ := regexp.MatchString(`(?i)tolong\s+buatin\s+tiket`, cleanBody); matched {
		re := regexp.MustCompile(`(?i)tolong\s+buatin\s+tiket\s+(.+)`)
		matches := re.FindStringSubmatch(cleanBody)
		if len(matches) > 1 {
			return &ParsedAction{
				ActionType: "create_ticket",
				Content:    strings.TrimSpace(matches[1]),
			}
		}
	}

	// If directed to bot and no command matched, treat as create ticket with full body
	if isDirectedToBot && cleanBody != "" {
		return &ParsedAction{
			ActionType: "create_ticket",
			Content:    cleanBody,
		}
	}

	// No action recognized
	return nil
}
