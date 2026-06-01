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

func ParseMessage(body string) *ParsedAction {
	body = strings.TrimSpace(body)

	// Check for create ticket pattern: "tolong buatin tiket ..."
	if matched, _ := regexp.MatchString(`(?i)tolong\s+buatin\s+tiket`, body); matched {
		re := regexp.MustCompile(`(?i)tolong\s+buatin\s+tiket\s+(.+)`)
		matches := re.FindStringSubmatch(body)
		if len(matches) > 1 {
			return &ParsedAction{
				ActionType: "create_ticket",
				Content:    strings.TrimSpace(matches[1]),
			}
		}
	}

	// Check for ticket progress update: "TK-xxx progress: ..."
	if matched, _ := regexp.MatchString(`(?i)tk-\d+\s+progress:`, body); matched {
		re := regexp.MustCompile(`(?i)tk-(\d+)\s+progress:\s*(.+)`)
		matches := re.FindStringSubmatch(body)
		if len(matches) > 2 {
			return &ParsedAction{
				ActionType: "update",
				TicketID:   matches[1],
				Content:    strings.TrimSpace(matches[2]),
			}
		}
	}

	// Check for close ticket: "TK-xxx close: ..."
	if matched, _ := regexp.MatchString(`(?i)tk-\d+\s+close:`, body); matched {
		re := regexp.MustCompile(`(?i)tk-(\d+)\s+close:\s*(.+)`)
		matches := re.FindStringSubmatch(body)
		if len(matches) > 2 {
			return &ParsedAction{
				ActionType: "close",
				TicketID:   matches[1],
				Content:    strings.TrimSpace(matches[2]),
			}
		}
	}

	// Check for reopen ticket: "TK-xxx reopen"
	if matched, _ := regexp.MatchString(`(?i)tk-\d+\s+reopen`, body); matched {
		re := regexp.MustCompile(`(?i)tk-(\d+)\s+reopen`)
		matches := re.FindStringSubmatch(body)
		if len(matches) > 1 {
			return &ParsedAction{
				ActionType: "reopen",
				TicketID:   matches[1],
				Content:    "",
			}
		}
	}

	// Check for status check: "status TK-xxx"
	if matched, _ := regexp.MatchString(`(?i)status\s+tk-\d+`, body); matched {
		re := regexp.MustCompile(`(?i)status\s+tk-(\d+)`)
		matches := re.FindStringSubmatch(body)
		if len(matches) > 1 {
			return &ParsedAction{
				ActionType: "status_check",
				TicketID:   matches[1],
				Content:    "",
			}
		}
	}

	// No action recognized
	return nil
}
