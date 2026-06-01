package email

import (
	"log"
	"strings"

	"ai-desk/internal/models"
	"gorm.io/gorm"
)

type DomainMatcher struct {
	db *gorm.DB
}

// NewDomainMatcher creates a new domain matcher
func NewDomainMatcher(db *gorm.DB) *DomainMatcher {
	return &DomainMatcher{db: db}
}

// MatchDomain matches an email domain to a customer
func (dm *DomainMatcher) MatchDomain(emailAddress string) *DomainMatchResult {
	senderDomain := ExtractDomain(emailAddress)
	if senderDomain == "" {
		log.Printf("Failed to extract domain from email: %s", emailAddress)
		return &DomainMatchResult{IsMatch: false}
	}

	log.Printf("Matching domain for email %s, extracted domain: %s", emailAddress, senderDomain)

	// First, try exact domain match
	var customer models.Customer
	if err := dm.db.Where("LOWER(domain) = LOWER(?)", senderDomain).First(&customer).Error; err == nil {
		log.Printf("Found exact domain match: %s -> Customer ID %d (%s)", senderDomain, customer.ID, customer.Name)
		return &DomainMatchResult{
			CustomerID:   customer.ID,
			CustomerName: customer.Name,
			Domain:       customer.Domain,
			IsMatch:      true,
		}
	}

	// Try fuzzy matching - check if sender domain is subdomain of customer domain
	var customers []models.Customer
	if err := dm.db.Where("domain IS NOT NULL AND domain != ''").Find(&customers).Error; err != nil {
		log.Printf("Error fetching customers for fuzzy match: %v", err)
		return &DomainMatchResult{IsMatch: false}
	}

	for _, c := range customers {
		if c.Domain == "" {
			continue
		}

		// Check if sender domain matches or is subdomain of customer domain
		if fuzzyMatchDomain(senderDomain, c.Domain) {
			log.Printf("Found fuzzy domain match: %s matches %s -> Customer ID %d (%s)", senderDomain, c.Domain, c.ID, c.Name)
			return &DomainMatchResult{
				CustomerID:   c.ID,
				CustomerName: c.Name,
				Domain:       c.Domain,
				IsMatch:      true,
			}
		}
	}

	log.Printf("No domain match found for: %s", senderDomain)
	return &DomainMatchResult{IsMatch: false}
}

// fuzzyMatchDomain checks if two domains match (including subdomains)
func fuzzyMatchDomain(senderDomain, customerDomain string) bool {
	senderDomain = strings.ToLower(strings.TrimSpace(senderDomain))
	customerDomain = strings.ToLower(strings.TrimSpace(customerDomain))

	// Exact match
	if senderDomain == customerDomain {
		return true
	}

	// Remove common TLD extensions and check base domain
	senderBase := getBaseDomain(senderDomain)
	customerBase := getBaseDomain(customerDomain)

	if senderBase != "" && customerBase != "" && senderBase == customerBase {
		return true
	}

	// Check if sender domain is a subdomain of customer domain
	if strings.HasSuffix(senderDomain, "."+customerDomain) {
		return true
	}

	// Check if customer domain is a subdomain of sender domain (less likely but possible)
	if strings.HasSuffix(customerDomain, "."+senderDomain) {
		return true
	}

	// Check partial match - if main domain part matches
	senderParts := strings.Split(senderDomain, ".")
	customerParts := strings.Split(customerDomain, ".")

	if len(senderParts) > 0 && len(customerParts) > 0 {
		// Compare main domain names (second to last part)
		senderMainIdx := len(senderParts) - 2
		customerMainIdx := len(customerParts) - 2

		if senderMainIdx >= 0 && customerMainIdx >= 0 {
			if senderParts[senderMainIdx] == customerParts[customerMainIdx] {
				return true
			}
		}
	}

	return false
}

// getBaseDomain extracts base domain (without TLD)
func getBaseDomain(domain string) string {
	parts := strings.Split(strings.ToLower(domain), ".")
	if len(parts) < 2 {
		return ""
	}

	// Return second to last part (main domain name)
	return parts[len(parts)-2]
}
