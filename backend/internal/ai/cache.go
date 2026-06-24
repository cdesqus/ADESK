package ai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/redis/go-redis/v9"
)

// SemanticCache provides intelligent caching for AI responses.
// It uses token-sort normalization and stop-word removal to match
// semantically similar messages to the same cache entry.
type SemanticCache struct {
	client  *redis.Client
	ttl     time.Duration
	enabled bool
	// Stats
	hits   int64
	misses int64
}

// CachedResponse is the data structure stored in Redis
type CachedResponse struct {
	ActionType   string `json:"action_type"`
	TicketID     string `json:"ticket_id"`
	Content      string `json:"content"`
	NaturalReply string `json:"natural_reply"`
	CachedAt     string `json:"cached_at"`
}

// Indonesian stop-words that don't carry semantic meaning for ticket classification
var stopWords = map[string]bool{
	"yang": true, "di": true, "ke": true, "dan": true, "atau": true,
	"ini": true, "itu": true, "nya": true, "kami": true, "kita": true,
	"saya": true, "aku": true, "min": true, "mas": true, "mba": true,
	"mbak": true, "pak": true, "bu": true, "ibu": true, "bapak": true,
	"dengan": true, "untuk": true, "pada": true, "dari": true, "ada": true,
	"adalah": true, "sudah": true, "bisa": true, "juga": true, "lagi": true,
	"dong": true, "deh": true, "nih": true, "ya": true, "yah": true,
	"sih": true, "kan": true, "kah": true, "lah": true, "pun": true,
	"halo": true, "hai": true, "hi": true, "hey": true, "selamat": true,
	"pagi": true, "siang": true, "sore": true, "malam": true,
	"terima": true, "kasih": true, "makasih": true, "thanks": true,
	"mohon": true, "maaf": true, "permisi": true,
	"the": true, "a": true, "an": true, "is": true, "are": true,
	"was": true, "were": true, "be": true, "been": true, "being": true,
	"have": true, "has": true, "had": true, "do": true, "does": true,
	"did": true, "will": true, "would": true, "could": true, "should": true,
	"please": true, "help": true, "can": true, "you": true, "we": true,
}

// NewSemanticCache creates a new semantic cache instance
func NewSemanticCache(client *redis.Client, ttlStr string, enabled bool) *SemanticCache {
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		ttl = 24 * time.Hour
		log.Printf("[SemanticCache] Invalid TTL '%s', defaulting to 24h", ttlStr)
	}

	cache := &SemanticCache{
		client:  client,
		ttl:     ttl,
		enabled: enabled,
	}

	if enabled {
		log.Printf("[SemanticCache] Initialized with TTL=%v", ttl)
	} else {
		log.Printf("[SemanticCache] Disabled")
	}

	return cache
}

// Get checks the cache for a response matching the given message.
// Returns the cached response and a boolean indicating cache hit.
func (sc *SemanticCache) Get(message string) (*WhatsAppAIResponse, bool) {
	if !sc.enabled || sc.client == nil {
		return nil, false
	}

	ctx := context.Background()
	key := sc.buildCacheKey(message)

	data, err := sc.client.Get(ctx, key).Result()
	if err == redis.Nil {
		sc.misses++
		return nil, false
	}
	if err != nil {
		log.Printf("[SemanticCache] Redis GET error: %v", err)
		sc.misses++
		return nil, false
	}

	var cached CachedResponse
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		log.Printf("[SemanticCache] JSON unmarshal error: %v", err)
		sc.misses++
		return nil, false
	}

	sc.hits++
	log.Printf("[SemanticCache] HIT for message (hits=%d, misses=%d, ratio=%.1f%%)",
		sc.hits, sc.misses, float64(sc.hits)/float64(sc.hits+sc.misses)*100)

	return &WhatsAppAIResponse{
		ActionType:   cached.ActionType,
		TicketID:     cached.TicketID,
		Content:      cached.Content,
		NaturalReply: cached.NaturalReply,
	}, true
}

// Set stores an AI response in the cache
func (sc *SemanticCache) Set(message string, response *WhatsAppAIResponse) {
	if !sc.enabled || sc.client == nil || response == nil {
		return
	}

	// Only cache create_ticket and status_check responses
	// Don't cache update/close/reopen as they are stateful
	if response.ActionType != "create_ticket" && response.ActionType != "status_check" {
		return
	}

	ctx := context.Background()
	key := sc.buildCacheKey(message)

	cached := CachedResponse{
		ActionType:   response.ActionType,
		TicketID:     response.TicketID,
		Content:      response.Content,
		NaturalReply: response.NaturalReply,
		CachedAt:     time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(cached)
	if err != nil {
		log.Printf("[SemanticCache] JSON marshal error: %v", err)
		return
	}

	if err := sc.client.Set(ctx, key, data, sc.ttl).Err(); err != nil {
		log.Printf("[SemanticCache] Redis SET error: %v", err)
		return
	}

	log.Printf("[SemanticCache] STORED response for key=%s (action=%s)", key[:40], response.ActionType)
}

// buildCacheKey normalizes the message and creates a deterministic cache key.
//
// Normalization pipeline:
// 1. Lowercase
// 2. Remove mentions (@helpdesk, @628xxx)
// 3. Remove punctuation & extra whitespace
// 4. Remove stop-words
// 5. Sort remaining tokens alphabetically
// 6. SHA256 hash the result
//
// This means "server database error" and "database server error"
// produce the SAME cache key (token-sort invariant).
func (sc *SemanticCache) buildCacheKey(message string) string {
	normalized := sc.normalizeMessage(message)
	hash := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("ai:wa:cache:%x", hash)
}

// normalizeMessage applies the full normalization pipeline
func (sc *SemanticCache) normalizeMessage(message string) string {
	// 1. Lowercase
	text := strings.ToLower(message)

	// 2. Remove mentions
	reMention := regexp.MustCompile(`@[\w]+`)
	text = reMention.ReplaceAllString(text, "")

	// 3. Remove punctuation, keep only letters, digits, spaces
	var cleaned strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			cleaned.WriteRune(r)
		}
	}
	text = cleaned.String()

	// 4. Tokenize and remove stop-words
	words := strings.Fields(text)
	var meaningful []string
	for _, w := range words {
		if len(w) > 1 && !stopWords[w] {
			meaningful = append(meaningful, w)
		}
	}

	// 5. Sort tokens for order-invariant matching
	sort.Strings(meaningful)

	// 6. Join
	return strings.Join(meaningful, " ")
}

// Stats returns cache hit/miss statistics
func (sc *SemanticCache) Stats() (hits, misses int64) {
	return sc.hits, sc.misses
}
