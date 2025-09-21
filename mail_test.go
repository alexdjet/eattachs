package main

import (
	"io"
	"log"
	"testing"
	"time"

	"github.com/emersion/go-imap"
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

				msg := imap.NewMessage(3, []imap.FetchItem{})

				// bodyStructure := &imap.BodyStructure{
				// 	MIMEType:    "multipart", // Основной тип MIME
				// 	MIMESubType: "mixed",     // Подтип MIME
				// 	Parts: []*imap.BodyStructure{
				// 		{
				// 			// Первая часть - текстовое сообщение
				// 			MIMEType:    "text",
				// 			MIMESubType: "plain",
				// 			Params: map[string]string{
				// 				"charset": "utf-8",
				// 				"format":  "flowed",
				// 			},
				// 			Size:     1024, // Размер в байтах
				// 			Encoding: "quoted-printable",
				// 			Lines:    20, // Количество строк
				// 			// ID:       "<text-part@example.com>",
				// 		},
				// 		{
				// 			// Вторая часть - вложение
				// 			MIMEType:    "application",
				// 			MIMESubType: "zip",
				// 			Params: map[string]string{
				// 				"name": "report.zip",
				// 			},
				// 			Size:        500000, // Размер в байтах
				// 			Encoding:    "base64",
				// 			Disposition: "attachment",
				// 			DispositionParams: map[string]string{
				// 				"filename": "отчет_2025.zip",
				// 				"size":     "500KB",
				// 			},
				// 			// ID: "<attachment-part@example.com>",
				// 		},
				// 		{
				// 			// Третья часть - HTML-сообщение
				// 			MIMEType:    "text",
				// 			MIMESubType: "html",
				// 			Params: map[string]string{
				// 				"charset": "utf-8",
				// 			},
				// 			Size:     2048,
				// 			Encoding: "quoted-printable",
				// 			Lines:    30,
				// 		},
				// 	},
				// }

				// body := map[*imap.BodySectionName]imap.Literal{
				// 	{Partial: []int{0}}: NewLiteral([]byte(`Здравствуйте! Это тестовое сообщение с вложением.
				// 		С уважением, Команда`)),
				// 	{Partial: []int{1}}: NewLiteral([]byte("BASE64-encoded zip file data...")),
				// 	{Partial: []int{2}}: NewLiteral([]byte(
				// 		`<!DOCTYPE html>
				// 		<html>
				// 		<body>
				// 			<h1>Приветствие</h1>
				// 			<p>Это HTML-версия сообщения</p>
				// 		</body>
				// 		</html>`)),
				// }

				fields := []interface{}{
					"UID", "1001",
					"RFC822.SIZE", 5000,
					"INTERNALDATE", time.Now().Format(imap.DateTimeLayout),
					"FLAGS", []interface{}{"\\Seen", "\\Flagged"},

					// imap.FetchEnvelope, &imap.Envelope{
					// 	Date:      time.Now(),
					// 	Subject:   "Приветствие",
					// 	From:      []*imap.Address{{PersonalName: "Иван Петров", MailboxName: "ivan", HostName: "example.com"}},
					// 	Sender:    []*imap.Address{{MailboxName: "support", HostName: "example.com"}},
					// 	ReplyTo:   []*imap.Address{{MailboxName: "reply", HostName: "example.com"}},
					// 	To:        []*imap.Address{{PersonalName: "Анна Сидорова", MailboxName: "anna", HostName: "example.com"}},
					// 	Cc:        []*imap.Address{{MailboxName: "cc", HostName: "example.com"}},
					// 	InReplyTo: "<reply-id@example.com>",
					// 	MessageId: "<message-id@example.com>",
					// },

					"BODYSTRUCTURE", []interface{}{
						[]interface{}{
							"text",  // MIMEType
							"plain", // MIMESubType
							[]interface{}{ // Параметры
								"charset", "utf-8",
								"format", "flowed",
							},
							"<text-id@example.com>", // ID
							"",                      // Описание
							"quoted-printable",      // Encoding
							1024,                    // Размер
							20,
						},
						[]interface{}{
							"application", // MIMEType
							"zip",         // MIMESubType
							[]interface{}{ // Параметры
								"name", "report.zip",
							},
							"<attach-id@example.com>", // ID
							"",                        // Описание
							"base64",                  // Encoding
							500000,                    // Размер
							[]interface{}{ // Disposition
								"attachment",
								[]interface{}{
									"filename", "отчет_2025.zip",
									"size", "500KB",
								},
							},
						},
						// Тип multipart
						"mixed",
						// Расширенные параметры
						[]interface{}{
							"boundary", "----boundary-string----",
						},

						// Disposition всего сообщения
						[]interface{}{
							"inline",
							[]interface{}{
								"version", "1.0",
							},
						},

						// Язык
						"ru",

						// Расположение
						[]interface{}{
							"attachment",
							"inline",
						},
					},

					// imap.FetchBodyStructure, &imap.BodyStructure{},

					"BODY", []interface{}{
						[]interface{}{
							"text",  // MIMEType
							"plain", // MIMESubType
							[]interface{}{ // Параметры
								"charset", "utf-8",
								"format", "flowed",
							},
							"<text-id@example.com>", // ID части
							"",                      // Описание
							"quoted-printable",      // Кодировка
							1024,                    // Размер
							20,                      // Количество строк
						},
						// Вторая часть - HTML
						[]interface{}{
							"text", // MIMEType
							"html", // MIMESubType
							[]interface{}{ // Параметры
								"charset", "utf-8",
							},
							"<html-id@example.com>", // ID части
							"",                      // Описание
							"quoted-printable",      // Кодировка
							2048,                    // Размер
							30,                      // Количество строк
						},
						[]interface{}{
							"application",  // MIMEType
							"octet-stream", // MIMESubType
							[]interface{}{ // Параметры
								"name", "file.zip",
							},
							"<attach-id@example.com>", // ID части
							"",                        // Описание
							"base64",                  // Кодировка
							500000,                    // Размер
							[]interface{}{ // Disposition
								"attachment",
								[]interface{}{
									"filename", "документ.zip",
									"size", "500KB",
								},
							},
						},
					},
					// imap.FetchBody, map[*imap.BodySectionName]imap.Literal{},
				}

				err := msg.Parse(fields)
				if err != nil {
					log.Println(err)
				}

				log.Println(msg.Format()...)

				r := msg.GetBody(&imap.BodySectionName{})
				log.Println(r)

				ch <- msg

				// ch <- &imap.Message{
				// 	SeqNum:       3,
				// 	Uid:          1001,
				// 	Size:         5000,
				// 	InternalDate: time.Now(),
				// 	Flags:        []string{"\\Seen"},
				// 	Envelope: &imap.Envelope{
				// 		Date:      time.Now(),
				// 		Subject:   "Приветствие",
				// 		From:      []*imap.Address{{PersonalName: "Иван Петров", MailboxName: "ivan", HostName: "example.com"}},
				// 		Sender:    []*imap.Address{{MailboxName: "support", HostName: "example.com"}},
				// 		ReplyTo:   []*imap.Address{{MailboxName: "reply", HostName: "example.com"}},
				// 		To:        []*imap.Address{{PersonalName: "Анна Сидорова", MailboxName: "anna", HostName: "example.com"}},
				// 		Cc:        []*imap.Address{{MailboxName: "cc", HostName: "example.com"}},
				// 		InReplyTo: "<reply-id@example.com>",
				// 		MessageId: "<message-id@example.com>",
				// 	},
				// 	BodyStructure: &imap.BodyStructure{
				// 		MIMEType:    "multipart",
				// 		MIMESubType: "mixed",
				// 		Parts: []*imap.BodyStructure{
				// 			{
				// 				MIMEType:    "text",
				// 				MIMESubType: "plain",
				// 				Params: map[string]string{
				// 					"charset": "utf-8",
				// 				},
				// 				Size:     1024,
				// 				Encoding: "quoted-printable",
				// 				Lines:    20,
				// 			},
				// 			{
				// 				MIMEType:    "text/csv",
				// 				MIMESubType: "csv",
				// 				Size:        500000,
				// 				Encoding:    "base64",
				// 				Disposition: "attachment",
				// 				DispositionParams: map[string]string{
				// 					"filename": "transactions-2025_09_19.csv.zip",
				// 				},
				// 			},
				// 		},
				// 	},
				// 	Body: map[*imap.BodySectionName]imap.Literal{
				// 		{Partial: []int{0, 1}}: MyLiteral{Data: []byte("Message1")},
				// 		{Partial: []int{2, 3}}: MyLiteral{Data: []byte("Message2")},
				// 	},
				// }
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
}
