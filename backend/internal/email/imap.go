package email

import (
	"fmt"
	"log"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type IMAPClient struct {
	host     string
	port     int
	user     string
	password string
	client   *client.Client
}

// NewIMAPClient creates a new IMAP client
func NewIMAPClient(host string, port int, user, password string) *IMAPClient {
	return &IMAPClient{
		host:     host,
		port:     port,
		user:     user,
		password: password,
	}
}

// Connect establishes IMAP connection
func (ic *IMAPClient) Connect() error {
	addr := fmt.Sprintf("%s:%d", ic.host, ic.port)
	c, err := client.DialTLS(addr, nil)
	if err != nil {
		log.Printf("IMAP connection failed to %s: %v", addr, err)
		return fmt.Errorf("IMAP connection failed: %w", err)
	}

	if err := c.Login(ic.user, ic.password); err != nil {
		c.Logout()
		log.Printf("IMAP login failed for %s: %v", ic.user, err)
		return fmt.Errorf("IMAP login failed: %w", err)
	}

	ic.client = c
	log.Printf("IMAP connected successfully for %s", ic.user)
	return nil
}

// FetchUnreadEmails fetches all unread emails from inbox
func (ic *IMAPClient) FetchUnreadEmails() ([]*imap.Message, error) {
	if ic.client == nil {
		return nil, fmt.Errorf("IMAP client not connected")
	}

	// Select inbox
	mbox, err := ic.client.Select("INBOX", false)
	if err != nil {
		log.Printf("Failed to select INBOX: %v", err)
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	log.Printf("INBOX has %d messages, %d unseen", mbox.Messages, mbox.Unseen)

	if mbox.Unseen == 0 {
		return nil, fmt.Errorf("Connected successfully, but server reports INBOX has %d total messages and 0 unseen messages. Are you sure you logged into the correct email account?", mbox.Messages)
	}

	// Search for unread emails
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	uids, err := ic.client.Search(criteria)
	if err != nil {
		log.Printf("Failed to search unread emails: %v", err)
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(uids) == 0 {
		return nil, fmt.Errorf("Server reported %d unseen messages, but search for unseen returned 0. This is unusual.", mbox.Unseen)
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	// Fetch messages
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- ic.client.Fetch(seqset, []imap.FetchItem{
			imap.FetchEnvelope,
			imap.FetchBodyStructure,
			"BODY.PEEK[]",
			imap.FetchUid,
		}, messages)
	}()

	var result []*imap.Message
	for msg := range messages {
		result = append(result, msg)
	}

	if err := <-done; err != nil {
		log.Printf("Failed to fetch messages: %v", err)
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return result, nil
}

// MarkAsRead marks an email as read by UID
func (ic *IMAPClient) MarkAsRead(uid uint32) error {
	if ic.client == nil {
		return fmt.Errorf("IMAP client not connected")
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)

	flags := []interface{}{imap.SeenFlag}

	if err := ic.client.Store(seqset, imap.StoreItem("+FLAGS"), flags, nil); err != nil {
		log.Printf("Failed to mark UID %d as read: %v", uid, err)
		return fmt.Errorf("failed to mark as read: %w", err)
	}

	return nil
}

// Close closes the IMAP connection
func (ic *IMAPClient) Close() error {
	if ic.client == nil {
		return nil
	}

	if err := ic.client.Logout(); err != nil {
		log.Printf("IMAP logout error: %v", err)
		return err
	}

	return nil
}

// Reconnect reconnects to IMAP if connection is lost
func (ic *IMAPClient) Reconnect() error {
	_ = ic.Close()
	return ic.Connect()
}
