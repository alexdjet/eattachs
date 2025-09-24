package main

import (
	"io"
	"log"
	"testing"

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
		// SeqNum: 1,
		// Uid:    1001,
		// Flags:  []string{imap.SeenFlag},
		// Envelope: &imap.Envelope{
		// 	Date:      time.Now(),
		// 	Subject:   "Test email with attachment",
		// 	From:      []*imap.Address{{PersonalName: "Ivan Petrov", AtDomainList: "example.com", MailboxName: "ivan", HostName: "example.com"}},
		// 	Sender:    []*imap.Address{{PersonalName: "Ivan Petrov", AtDomainList: "example.com", MailboxName: "ivan", HostName: "example.com"}},
		// 	ReplyTo:   []*imap.Address{{PersonalName: "Reply", AtDomainList: "example.com", MailboxName: "reply", HostName: "example.com"}},
		// 	To:        []*imap.Address{{PersonalName: "To", AtDomainList: "example.com", MailboxName: "support", HostName: "example.com"}},
		// 	Cc:        []*imap.Address{{PersonalName: "cc", AtDomainList: "example.com", MailboxName: "cc", HostName: "example.com"}},
		// 	Bcc:       []*imap.Address{{PersonalName: "bcc", AtDomainList: "example.com", MailboxName: "bcc", HostName: "example.com"}},
		// 	InReplyTo: "<reply-id@example.com>",
		// 	MessageId: "<message-id@example.com>",
		// },

		Body: make(map[*imap.BodySectionName]imap.Literal),
	}

	// Создаем тело письма: 1-я часть — plain text
	rawText := []byte(`Message-ID: <7cf4fa0c71e78fff42bf60af763beec4@pay.example.com>
	Date: Tue, 23 Sep 2025 09:58:25 +0300
	Subject: =?utf-8?Q?=D0=A0=D0=B5=D0=B5=D1=81=D1=82=D1=80_=D0=BF?=
	 =?utf-8?Q?=D0=BB=D0=B0=D1=82=D0=B5=D0=B6=D0=B5=D0=B9_=D0=BF=D0=BE_=D0=9D?=
	 =?utf-8?Q?=D0=9A=D0=9E_=C2=AB=D0=9C=D0=9E=D0=9D=D0=95=D0=A2=D0=90=C2=BB_?=
	 =?utf-8?Q?=D1=81?= 15.09.2025 =?utf-8?Q?=D0=BF=D0=BE?= 21.09.2025
	From: pay@example1.com
	To: ubuifk@example1.com
	Cc: support@example1.com
	MIME-Version: 1.0
	Content-Type: multipart/mixed;
	boundary="--boundary--"
	

	--boundary
	`)

	// 2-я часть — HTML
	rawHTML := []byte(`Content-Type: multipart/html; charset=utf-8
	Content-Transfer-Encoding: quoted-printable
	
	<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www=
	.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">
		  <html
		   =20
			dir=3D"ltr"
			xmlns=3D"http://www.w3.org/1999/xhtml"
			xmlns:v=3D"urn:schemas-microsoft-com:vml"
			=
	xmlns:o=3D"urn:schemas-microsoft-com:office:office">
			<head>
			  <meta http-equiv=3D"Content-Type" content=3D"text/html; =
	charset=3Dutf-8" />
			  <meta http-equiv=3D"X-UA-Compatible" =
	content=3D"IE=3Dedge" />
			  <meta name=3D"viewport" =
	content=3D"width=3Ddevice-width"/>
	
			  <title>=D0=94=D0=BE=D0=B1=
	=D1=80=D0=BE =D0=BF=D0=BE=D0=B6=D0=B0=D0=BB=D0=BE=D0=B2=D0=B0=D1=82=D1=8C =
	=D0=B2 Yonote</title>
	
	--boundary
	`)

	// file, err := os.ReadFile("data/transactions-2025_07_09.csv.zip")
	// if err != nil {
	// 	log.Fatalf("Error reading file: %v", err)
	// }

	// 3-я часть — вложение (например, zip)
	rawAttachment := []byte(`Content-Type: multipart/vnd.openxmlformats-officedocument.spreadsheetml.sheet;
   boundary="--boundary"
	name=2025_09_21-2025_09_15.xlsx
   Content-Transfer-Encoding: base64
   Content-Disposition: attachment; filename=2025_09_21-2025_09_15.xlsx
   
   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
   
   --boundary
   

   `)

	// Определяем BodySectionName для каждой части
	sectionText := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			// Path: []int{1},
		},
	}
	sectionHTML := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			// Path: []int{2},
		},
	}
	sectionAttachment := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			// Path: []int{3},
		},
	}

	// Заполняем тело письма соответствующими частями
	msg.Body[sectionText] = bytes.NewReader(rawText)
	msg.Body[sectionHTML] = bytes.NewReader(rawHTML)
	msg.Body[sectionAttachment] = bytes.NewReader(rawAttachment)

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

		// log.Println(msg.Body)

		bs := imap.BodySectionName{
			BodyPartName: imap.BodyPartName{
				Path: []int{},
			},
		}

		r := msg.GetBody(&bs)
		if r == nil {
			log.Println("Server didn't return message body")
			continue
		}

		entity, err := message.Read(r)
		if err != nil {
			log.Println(err)
			continue
		}

		log.Println(entity.Header.Get(""))

		mr := entity.MultipartReader()

		if mr == nil {
			log.Println("Письмо не multipart, вложений нет")
			continue
		}

		log.Println(mr.NextPart())

		// for {
		// 	part, err := mr.NextPart()
		// 	if err == io.EOF {
		// 		break
		// 	}

		// 	if err != nil {
		// 		log.Println("Ошибка чтения части письма:", err)
		// 		break
		// 	}

		// 	log.Println(part)
		// }

	}

}
