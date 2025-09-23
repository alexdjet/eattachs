package main

import (
	"io"
	"log"
	"os"
	"testing"
	"time"

	"bytes"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message"
)

type MockIMAPClient struct {
	searchFunc func(criteria *imap.SearchCriteria) ([]uint32, error)
	listFunc   func(ref, mailbox string, ch chan *imap.MailboxInfo) error
	selectFunc func(name string, readOnly bool) (*imap.MailboxStatus, error)
	fetchFunc  func(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	logoutFunc func() error
}

func (m *MockIMAPClient) Search(criteria *imap.SearchCriteria) ([]uint32, error) {
	return m.searchFunc(criteria)
}

func (m *MockIMAPClient) List(ref, mailbox string, ch chan *imap.MailboxInfo) error {
	return m.listFunc(ref, mailbox, ch)
}

func (m *MockIMAPClient) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
	return m.selectFunc(name, readOnly)
}

func (m *MockIMAPClient) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
	return m.fetchFunc(seqset, items, ch)
}

func (m *MockIMAPClient) Logout() error {
	return m.logoutFunc()
}

// // Stub functions
// func mockSearchBySubject(clt IMAPClient, unread bool, fromEmail, fromSubject string) ([]uint32, error) {
// 	return []uint32{1, 2}, nil
// }

// func mockGetAttachments(msgs []*imap.Message, cfg *Config) ([]string, error) {
// 	return []string{"file1.txt"}, nil
// }

type literalStruct struct {
	data []byte
	pos  int
}

func (l *literalStruct) Read(p []byte) (n int, err error) {
	if l.pos >= len(l.data) {
		return 0, io.EOF
	}

	n = copy(p, l.data[l.pos:])
	l.pos += n
	return n, nil
}

func (l *literalStruct) Len() int {
	return len(l.data)
}

func NewLiteral(data []byte) imap.Literal {
	return &literalStruct{
		data: data,
	}
}

func TestGetListEmail_InboxSuccess(t *testing.T) {
	mockClient := &MockIMAPClient{
		searchFunc: func(criteria *imap.SearchCriteria) ([]uint32, error) {
			return []uint32{1, 3}, nil
		},
		listFunc: func(ref, mailbox string, ch chan *imap.MailboxInfo) error {
			ch <- &imap.MailboxInfo{Name: "INBOX", Attributes: []string{"\\Noselect", "\\HasChildren", "\\Marked"}}
			close(ch)
			return nil
		},
		selectFunc: func(name string, readOnly bool) (*imap.MailboxStatus, error) {
			return &imap.MailboxStatus{
				ReadOnly:       false,
				Items:          make(map[imap.StatusItem]interface{}),
				Flags:          []string{"\\Seen", "\\Flagged"},
				PermanentFlags: []string{"\\Answered", "\\Deleted"},
				Messages:       10,
				Recent:         5,
				Unseen:         2,
				UidNext:        1001,
				UidValidity:    123456,
				AppendLimit:    25000000,
			}, nil

			// return &imap.MailboxStatus{Flags: []string{"\\Seen"}}, nil
		},
		fetchFunc: func(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
			if seqset.Contains(1) {
				ch <- &imap.Message{SeqNum: 1}
			}

			if seqset.Contains(2) {
				ch <- &imap.Message{SeqNum: 2}
			}

			if seqset.Contains(3) {
				ch <- &imap.Message{SeqNum: 3}
			}

			if seqset.Contains(4) {
				ch <- &imap.Message{SeqNum: 4}
			}

			close(ch)
			return nil
		},
		logoutFunc: func() error {
			return nil
		},
	}

	cfg := &Config{FromEmail: "search@example1.com", FromSubject: "Hello"}

	msgs, err := getListEmail(mockClient, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(msgs) != 2 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestGetListEmail_NoUnread(t *testing.T) {
	mockClient := &MockIMAPClient{
		searchFunc: func(criteria *imap.SearchCriteria) ([]uint32, error) {
			return []uint32{}, nil
		},
		listFunc: func(ref, mailbox string, ch chan *imap.MailboxInfo) error {
			close(ch)
			return nil
		},
		selectFunc: func(name string, readOnly bool) (*imap.MailboxStatus, error) {
			return &imap.MailboxStatus{Flags: []string{}}, nil
		},
		fetchFunc: func(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
			close(ch)
			return nil
		},
		logoutFunc: func() error {
			return nil
		},
	}

	cfg := &Config{FromEmail: "test@example.com", FromSubject: "Hello"}

	_, err := getListEmail(mockClient, cfg)

	if err == nil || err.Error() != "нет непрочитанных писем" {
		t.Fatalf("expected 'нет непрочитанных писем' error, got %v", err)
	}
}

func createTestMessageWithAttachment() *imap.Message {
	msg := &imap.Message{
		SeqNum: 1,
		Uid:    1001,
		Flags:  []string{imap.SeenFlag},
		Envelope: &imap.Envelope{
			Date:      time.Now(),
			Subject:   "Test email with attachment",
			From:      []*imap.Address{{PersonalName: "Ivan Petrov", AtDomainList: "example.com", MailboxName: "ivan", HostName: "example.com"}},
			Sender:    []*imap.Address{{PersonalName: "Ivan Petrov", AtDomainList: "example.com", MailboxName: "ivan", HostName: "example.com"}},
			ReplyTo:   []*imap.Address{{PersonalName: "Reply", AtDomainList: "example.com", MailboxName: "reply", HostName: "example.com"}},
			To:        []*imap.Address{{PersonalName: "To", AtDomainList: "example.com", MailboxName: "support", HostName: "example.com"}},
			Cc:        []*imap.Address{{PersonalName: "cc", AtDomainList: "example.com", MailboxName: "cc", HostName: "example.com"}},
			Bcc:       []*imap.Address{{PersonalName: "bcc", AtDomainList: "example.com", MailboxName: "bcc", HostName: "example.com"}},
			InReplyTo: "<reply-id@example.com>",
			MessageId: "<message-id@example.com>",
		},

		Body: make(map[*imap.BodySectionName]imap.Literal),
		BodyStructure: &imap.BodyStructure{
			// multipart/mixed
			Parts: []*imap.BodyStructure{
				{
					MIMEType:    "text",
					MIMESubType: "plain",
					Params: map[string]string{
						"charset": "utf-8",
						"format":  "flowed",
					},
					Encoding: "quoted-printable",
					Size:     1024,
					Lines:    20,
				},
				{
					MIMEType:    "text",
					MIMESubType: "html",
					Params: map[string]string{
						"charset": "utf-8",
					},
					Encoding: "quoted-printable",
					Size:     2048,
					Lines:    30,
				},
				{
					MIMEType:    "application",
					MIMESubType: "octet-stream",
					// Params: map[string]string{
					// 	"name": "transactions-2025_07_08.csv.zip",
					// },
					Id:          "<attach-id@example.com>",
					Description: "DescriptionBody3",
					Encoding:    "base64",
					Size:        500000,
					// Disposition: &imap.BodyDisposition{
					// 	Type: "attachment",
					// 	Params: map[string]string{
					// 		"filename":    "transactions-2025_07_08.csv.zip",
					// 		"size":        "500KB",
					// 		"modify-date": "2025-09-22T12:43:37Z",
					// 	},
					// },
					Language: []string{"ru"},
					Location: []string{""},
				},
			},
			// Тип multipart
			MIMEType:    "multipart",
			MIMESubType: "mixed",
			Params: map[string]string{
				"boundary": "----boundary-string----",
				"format":   "flowed",
			},
			Size: 500000 + 1024 + 2048,
			// Дополнительные поля (ContentMD5, etc.) можно добавить при необходимости
		},
	}

	// Создаем тело письма: 1-я часть — plain text
	rawText := []byte("This is the plain text part of the email")

	// 2-я часть — HTML
	rawHTML := []byte("<html><body>This is the <b>HTML</b> part</body></html>")

	file, err := os.ReadFile("data/transactions-2025_07_09.csv.zip")
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// 3-я часть — вложение (например, zip)
	// rawAttachment := []byte{0x50, 0x4b, 0x03, 0x04, 0x50, 0x4b, 0x03, 0x04, 0x50, 0x4b, 0x03, 0x04} // пример начальных байтов zip

	// Определяем BodySectionName для каждой части
	sectionText := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			Path: []int{1},
		},
	}
	sectionHTML := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			Path: []int{2},
		},
	}
	sectionAttachment := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			Path: []int{3},
		},
	}

	// Заполняем тело письма соответствующими частями
	msg.Body[sectionText] = bytes.NewReader(rawText)
	msg.Body[sectionHTML] = bytes.NewReader(rawHTML)
	msg.Body[sectionAttachment] = bytes.NewReader(file)

	return msg
}

func TestGetListEmail_Files(t *testing.T) {
	mockClient := &MockIMAPClient{
		searchFunc: func(criteria *imap.SearchCriteria) ([]uint32, error) {
			return []uint32{3}, nil
		},
		listFunc: func(ref, mailbox string, ch chan *imap.MailboxInfo) error {
			ch <- &imap.MailboxInfo{Name: "INBOX", Attributes: []string{"\\Noselect", "\\HasChildren", "\\Marked"}}
			close(ch)
			return nil
		},
		selectFunc: func(name string, readOnly bool) (*imap.MailboxStatus, error) {
			return &imap.MailboxStatus{
				ReadOnly:       false,
				Items:          make(map[imap.StatusItem]interface{}),
				Flags:          []string{"\\Seen", "\\Flagged"},
				PermanentFlags: []string{"\\Answered", "\\Deleted"},
				Messages:       10,
				Recent:         5,
				Unseen:         2,
				UidNext:        1001,
				UidValidity:    123456,
				AppendLimit:    25000000,
			}, nil

			// return &imap.MailboxStatus{Flags: []string{"\\Seen"}}, nil
		},
		fetchFunc: func(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
			if seqset.Contains(1) {
				ch <- &imap.Message{SeqNum: 1}
			}

			if seqset.Contains(2) {
				ch <- &imap.Message{SeqNum: 2}
			}

			if seqset.Contains(3) {

				msg := createTestMessageWithAttachment()
				ch <- msg
			}

			if seqset.Contains(4) {
				ch <- &imap.Message{SeqNum: 4}
			}

			close(ch)
			return nil
		},
		logoutFunc: func() error {
			return nil
		},
	}

	cfg := &Config{FromEmail: "files@example1.com", FromSubject: "Report files", WorkDir: "./data1"}

	msgs, err := getListEmail(mockClient, cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(msgs) != 1 {
		t.Errorf("expected 1 messages, got %d", len(msgs))
	}

	for _, msg := range msgs {

		// log.Println(msg.Envelope.Format()...)

		for i := range 5 {
			bs := imap.BodySectionName{
				BodyPartName: imap.BodyPartName{
					Path: []int{i + 1},
				},
			}

			r := msg.GetBody(&bs)
			if r == nil {
				log.Println("Server didn't return message body [", i+1, "]")
				continue
			}

			// log.Println(r)

			entity, err := message.Read(r)
			if err != nil {
				log.Println("Ошибка парсинга MIME:", err)
				continue
			}

			mr := entity.MultipartReader()
			if mr == nil {
				log.Println("Письмо не multipart, вложений нет")
				continue
			}
		}
	}

}
