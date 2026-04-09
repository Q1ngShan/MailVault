package imap

import (
	"fmt"
	"io"
	"mime"
	"strings"
	"time"

	goimap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"mailvault/internal/models"
)

const imapServer = "outlook.live.com:993"

// xoauth2Auth implements sasl.Client for XOAUTH2 mechanism
type xoauth2Auth struct {
	user        string
	accessToken string
}

func (a *xoauth2Auth) Start() (string, []byte, error) {
	token := fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", a.user, a.accessToken)
	return "XOAUTH2", []byte(token), nil
}

func (a *xoauth2Auth) Next(_ []byte) ([]byte, error) {
	return nil, nil
}

func connect(email, accessToken string) (*client.Client, error) {
	c, err := client.DialTLS(imapServer, nil)
	if err != nil {
		return nil, fmt.Errorf("dial failed: %w", err)
	}

	auth := &xoauth2Auth{user: email, accessToken: accessToken}
	if err := c.Authenticate(auth); err != nil {
		c.Logout()
		return nil, fmt.Errorf("auth failed: %w", err)
	}

	return c, nil
}

func folderName(folder string) string {
	switch strings.ToLower(folder) {
	case "spam", "junk":
		return "Junk"
	default:
		return "INBOX"
	}
}

// FetchMails retrieves paginated mail list from the given folder.
func FetchMails(email, accessToken, folder string, page, pageSize int) (*models.MailListResponse, error) {
	c, err := connect(email, accessToken)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	mbox, err := c.Select(folderName(folder), true)
	if err != nil {
		return nil, fmt.Errorf("select folder failed: %w", err)
	}

	total := int(mbox.Messages)
	if total == 0 {
		return &models.MailListResponse{
			Email:    email,
			Folder:   folder,
			Page:     page,
			PageSize: pageSize,
			Total:    0,
			Items:    []models.MailItem{},
		}, nil
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	// Newest first: calculate sequence range
	start := total - (page-1)*pageSize
	end := start - pageSize + 1
	if end < 1 {
		end = 1
	}
	if start < 1 {
		return &models.MailListResponse{
			Email:    email,
			Folder:   folder,
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			Items:    []models.MailItem{},
		}, nil
	}

	seqSet := new(goimap.SeqSet)
	seqSet.AddRange(uint32(end), uint32(start))

	fetchItems := []goimap.FetchItem{goimap.FetchEnvelope, goimap.FetchUid}
	messages := make(chan *goimap.Message, pageSize)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqSet, fetchItems, messages)
	}()

	var mailItems []models.MailItem
	for msg := range messages {
		if msg.Envelope == nil {
			continue
		}
		subject := decodeHeader(msg.Envelope.Subject)
		from := formatAddress(msg.Envelope.From)
		date := ""
		if !msg.Envelope.Date.IsZero() {
			date = msg.Envelope.Date.Format(time.DateTime)
		}
		mailItems = append(mailItems, models.MailItem{
			UID:     fmt.Sprintf("%d", msg.Uid),
			Subject: subject,
			From:    from,
			Date:    date,
			Folder:  folder,
		})
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch failed: %w", err)
	}

	// Reverse to show newest first
	for i, j := 0, len(mailItems)-1; i < j; i, j = i+1, j-1 {
		mailItems[i], mailItems[j] = mailItems[j], mailItems[i]
	}

	return &models.MailListResponse{
		Email:    email,
		Folder:   folder,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Items:    mailItems,
	}, nil
}

// FetchMailDetail retrieves the full content of a specific message by UID.
func FetchMailDetail(email, accessToken, folder, uid string) (*models.MailDetailResponse, error) {
	c, err := connect(email, accessToken)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	if _, err := c.Select(folderName(folder), true); err != nil {
		return nil, fmt.Errorf("select folder failed: %w", err)
	}

	seqSet := new(goimap.SeqSet)
	var uidNum uint32
	fmt.Sscanf(uid, "%d", &uidNum)
	seqSet.AddNum(uidNum)

	section := &goimap.BodySectionName{}
	fetchItems := []goimap.FetchItem{section.FetchItem(), goimap.FetchEnvelope, goimap.FetchUid}

	messages := make(chan *goimap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqSet, fetchItems, messages)
	}()

	var detail models.MailDetail
	for msg := range messages {
		if msg.Envelope != nil {
			detail.Subject = decodeHeader(msg.Envelope.Subject)
			detail.From = formatAddress(msg.Envelope.From)
			detail.To = formatAddress(msg.Envelope.To)
			if !msg.Envelope.Date.IsZero() {
				detail.Date = msg.Envelope.Date.Format(time.DateTime)
			}
		}

		r := msg.GetBody(section)
		if r != nil {
			mr, err := mail.CreateReader(r)
			if err == nil {
				for {
					p, err := mr.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						break
					}
					switch h := p.Header.(type) {
					case *mail.InlineHeader:
						ct, _, _ := h.ContentType()
						body, _ := io.ReadAll(p.Body)
						switch ct {
						case "text/html":
							detail.BodyHTML = string(body)
						case "text/plain":
							if detail.BodyText == "" {
								detail.BodyText = string(body)
							}
						}
					}
				}
			}
		}
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("uid fetch failed: %w", err)
	}

	return &models.MailDetailResponse{
		Email:     email,
		Folder:    folder,
		MessageID: uid,
		Detail:    detail,
	}, nil
}

func formatAddress(addrs []*goimap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	addr := addrs[0]
	name := decodeHeader(addr.PersonalName)
	email := fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
	if name != "" {
		return fmt.Sprintf("%s <%s>", name, email)
	}
	return email
}

func decodeHeader(s string) string {
	dec := mime.WordDecoder{}
	decoded, err := dec.DecodeHeader(s)
	if err != nil {
		return s
	}
	return decoded
}

// ─── OTP helpers for Codex OAuth email verification ──────────────────────────

// otpFolders are the mailbox names checked when polling for OTP emails.
// OpenAI verification emails are often filtered to Junk by Outlook.
var otpFolders = []string{"INBOX", "Junk"}

// SnapshotInboxUIDs returns the set of UID numbers currently in INBOX and Junk.
// Used to detect new messages arriving after this call.
func SnapshotInboxUIDs(email, accessToken string) (map[uint32]bool, error) {
	c, err := connect(email, accessToken)
	if err != nil {
		return nil, err
	}
	defer c.Logout()

	ids := make(map[uint32]bool)
	for _, folder := range otpFolders {
		mbox, err := c.Select(folder, true)
		if err != nil {
			continue // folder may not exist
		}
		if mbox.Messages == 0 {
			continue
		}
		seqSet := new(goimap.SeqSet)
		seqSet.AddRange(1, mbox.Messages)
		msgs := make(chan *goimap.Message, 64)
		done := make(chan error, 1)
		go func() { done <- c.Fetch(seqSet, []goimap.FetchItem{goimap.FetchUid}, msgs) }()
		for msg := range msgs {
			ids[msg.Uid] = true
		}
		<-done
	}
	return ids, nil
}

// WaitForOTPCode polls INBOX and Junk for a new message containing a 6-digit OTP.
// oldUIDs is the snapshot taken before triggering the email send.
// Polls every 3 seconds until timeout.
func WaitForOTPCode(email, accessToken string, oldUIDs map[uint32]bool, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		code, err := checkFoldersForOTP(email, accessToken, oldUIDs)
		if err == nil && code != "" {
			return code, nil
		}
		time.Sleep(3 * time.Second)
	}
	return "", fmt.Errorf("等待邮箱验证码超时（%v）", timeout)
}

func checkFoldersForOTP(email, accessToken string, oldUIDs map[uint32]bool) (string, error) {
	c, err := connect(email, accessToken)
	if err != nil {
		return "", err
	}
	defer c.Logout()

	for _, folder := range otpFolders {
		code, err := checkFolderForOTP(c, folder, oldUIDs)
		if err == nil && code != "" {
			return code, nil
		}
	}
	return "", nil
}

func checkFolderForOTP(c interface {
	Select(name string, readOnly bool) (*goimap.MailboxStatus, error)
	Fetch(seqSet *goimap.SeqSet, items []goimap.FetchItem, ch chan *goimap.Message) error
}, folder string, oldUIDs map[uint32]bool) (string, error) {
	mbox, err := c.Select(folder, true)
	if err != nil {
		return "", err
	}
	if mbox.Messages == 0 {
		return "", nil
	}

	start := mbox.Messages
	end := uint32(1)
	if start > 20 {
		end = start - 19
	}

	seqSet := new(goimap.SeqSet)
	seqSet.AddRange(end, start)

	section := &goimap.BodySectionName{}
	items := []goimap.FetchItem{goimap.FetchUid, goimap.FetchEnvelope, section.FetchItem()}

	msgs := make(chan *goimap.Message, 20)
	done := make(chan error, 1)
	go func() { done <- c.Fetch(seqSet, items, msgs) }()

	var code string
	for msg := range msgs {
		if oldUIDs[msg.Uid] {
			continue
		}
		// Prefer subject extraction: "Your OpenAI code is XXXXXX"
		if msg.Envelope != nil {
			if found := extractOTPFromSubject(msg.Envelope.Subject); found != "" {
				code = found
				continue
			}
		}
		r := msg.GetBody(section)
		if r == nil {
			continue
		}
		body, _ := io.ReadAll(r)
		if found := extractOTPFromBody(string(body)); found != "" {
			code = found
		}
	}
	<-done
	return code, nil
}

// extractOTPFromSubject extracts a 6-digit code from subjects like
// "Your OpenAI code is 123456" or "123456 is your verification code".
func extractOTPFromSubject(subject string) string {
	return findSixDigits(subject)
}

// ExtractOTPFromBody finds a 6-digit OTP code in an email body.
func ExtractOTPFromBody(body string) string { return extractOTPFromBody(body) }

// extractOTPFromBody finds a 6-digit OTP code in an email body.
func extractOTPFromBody(body string) string {
	// Strategy 1: OpenAI styled HTML block
	if idx := strings.Index(body, "background-color:#F3F3F3"); idx != -1 {
		block := body[idx:]
		if end := strings.Index(block, "</p>"); end != -1 {
			block = block[:end]
			if code := findSixDigits(block); code != "" {
				return code
			}
		}
	}
	// Strategy 2: between HTML tags
	for i := 0; i < len(body)-7; i++ {
		if body[i] == '>' {
			j := i + 1
			for j < len(body) && body[j] == ' ' {
				j++
			}
			if j+6 <= len(body) && isDigits(body[j:j+6]) {
				k := j + 6
				for k < len(body) && body[k] == ' ' {
					k++
				}
				if k < len(body) && body[k] == '<' && body[j:j+6] != "177010" {
					return body[j : j+6]
				}
			}
		}
	}
	// Strategy 3: standalone 6-digit number
	return findSixDigits(body)
}

func findSixDigits(s string) string {
	for i := 0; i+6 <= len(s); i++ {
		if isDigits(s[i:i+6]) {
			before := i == 0 || !isDigit(s[i-1])
			after := i+6 >= len(s) || !isDigit(s[i+6])
			if before && after && s[i:i+6] != "177010" {
				return s[i : i+6]
			}
		}
	}
	return ""
}

func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
