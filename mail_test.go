package main

import (
	"io"
	"os"
	"testing"

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
				bsn := imap.BodySectionName{}
				msg := &imap.Message{
					SeqNum: 1,
					Uid:    1001,
					Flags:  []string{imap.SeenFlag},
					Body: map[*imap.BodySectionName]imap.Literal{
						&bsn: NewLiteral([]byte(
							`Date: Mon, 22 Sep 2025 12:53:40 +0300 (MSK)
From: PayAnyWay <noreply@support.payanyway.ru>
To: pay@vspu.ru
Message-ID: <1513202856.35376.1758534820444@support.moneta.ru>
Subject: =?UTF-8?B?0JXQttC10LTQvdC10LLQvdGL0Lkg0L7RgtGH0LXRgiDQv9C+INC+?=
    =?UTF-8?B?0L/QtdGA0LDRhtC40Y/QvCDQt9CwIDIxLjA5LjIwMjU=?=
MIME-Version: 1.0
Content-Type: multipart/mixed; 
    boundary="----=_Part_35374_276279577.1758534820324"

------=_Part_35374_276279577.1758534820324
Content-Type: multipart/related; 
    boundary="----=_Part_35375_2120977340.1758534820324"

------=_Part_35375_2120977340.1758534820324
Content-Type: text/plain; charset=UTF-8
Content-Transfer-Encoding: quoted-printable


    MONETA.RU

---------------------------------------------------------------------------=
----

    =D0=97=D0=B4=D1=80=D0=B0=D0=B2=D1=81=D1=82=D0=B2=D1=83=D0=B9=D1=82=D0=B5,

    =D0=95=D0=B6=D0=B5=D0=B4=D0=BD=D0=B5=D0=B2=D0=BD=D1=8B=D0=B9 =D0=BE=D1=82=
=D1=87=D0=B5=D1=82 =D0=BF=D0=BE =D0=BE=D0=BF=D0=B5=D1=80=D0=B0=D1=86=D0=B8=
=D1=8F=D0=BC =D0=B7=D0=B0 21.09.2025.

---------------------------------------------------------------------------=
----

    =D0=A1 =D1=83=D0=B2=D0=B0=D0=B6=D0=B5=D0=BD=D0=B8=D0=B5=D0=BC,
    =D0=A1=D0=BB=D1=83=D0=B6=D0=B1=D0=B0 =D0=BF=D0=BE =D1=80=D0=B0=D0=B1=D0=
=BE=D1=82=D0=B5 =D1=81 =D0=BA=D0=BB=D0=B8=D0=B5=D0=BD=D1=82=D0=B0=D0=BC=D0=
=B8 www.moneta.ru

------=_Part_35375_2120977340.1758534820324--

------=_Part_35374_276279577.1758534820324
Content-Type: application/zip; name=transactions1.csv.zip
Content-Transfer-Encoding: base64
Content-Disposition: attachment; filename=transactions1.csv.zip

UEsDBBQACAAIAIZ4PVsAAAAAAAAAAE8AAAAJACAAdGVzdDEuY3N2VVQNAAe9ddpoRmnaaMl12mh1
eAsAAQToAwAABOgDAAArSS0usS4BEoZg0ghMGoNJEzBpas0FljaAqIIqg1DGEMoEQpkaWnMBAFBL
Bwg96xv1KQAAAE8AAABQSwECFAMUAAgACACGeD1bPesb9SkAAABPAAAACQAgAAAAAAAAAAAAtIEA
AAAAdGVzdDEuY3N2VVQNAAe9ddpoRmnaaMl12mh1eAsAAQToAwAABOgDAABQSwUGAAAAAAEAAQBX
AAAAgAAAAAAA


------=_Part_35374_276279577.1758534820324
Content-Type: application/zip; name=transactions2.csv.zip
Content-Transfer-Encoding: base64
Content-Disposition: attachment; filename=transactions2.csv.zip

UEsDBBQACAAIAAZ4PVsAAAAAAAAAAEcAAAAJACAAdGVzdDIuY3N2VVQNAAfNdNpozXTaaM102mh1
eAsAAQToAwAABOgDAABLzs8xtE7OzzECEcYgwgREmIIIMxBhbs2VTFgNAFBLBwheYiLJHQAAAEcA
AABQSwECFAMUAAgACAAGeD1bXmIiyR0AAABHAAAACQAgAAAAAAAAAAAAtIEAAAAAdGVzdDIuY3N2
VVQNAAfNdNpozXTaaM102mh1eAsAAQToAwAABOgDAABQSwUGAAAAAAEAAQBXAAAAdAAAAAAA
------=_Part_35374_276279577.1758534820324--

`)),
					},
				}

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

	cfg := &Config{FromEmail: "files@example1.com", FromSubject: "Report files", WorkDir: "./tempData"}

	msgs, err := getListEmail(mockClient, cfg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(msgs) != 1 {
		t.Errorf("expected 1 messages, got %d", len(msgs))
	}

	filesWorkDir, err := os.ReadDir(cfg.WorkDir)
	if err != nil {
		t.Errorf("Error reading directory: %v", err)
	}

	if len(filesWorkDir) != 2 {
		t.Errorf("expected 2 attached files, got %d", len(filesWorkDir))
	}

	for _, fl := range filesWorkDir {
		name := fl.Name()

		arr := make(map[string]struct{})
		arr["transactions1.csv.zip"] = struct{}{}
		arr["transactions2.csv.zip"] = struct{}{}

		_, found := arr[name]
		if !found {
			t.Errorf("No files working directory")
		}
	}

	errRemoveAl := os.RemoveAll(cfg.WorkDir)
	if err != nil {
		t.Fatalf("failed to remove directory %s: %v", cfg.WorkDir, errRemoveAl)
	}
}
