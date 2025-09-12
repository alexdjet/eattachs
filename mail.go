package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/textproto"
	"os"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message"
)

type IMAPClient interface {
	List(ref, mailbox string, ch chan *imap.MailboxInfo) error
	Select(name string, readOnly bool) (*imap.MailboxStatus, error)
	Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	Logout() error
	Search(criteria *imap.SearchCriteria) ([]uint32, error)
}

// type RIMAPClient struct {
// 	*client.Client
// }

/*
// func (r *RIMAPClient) Login(username, password string) error {
// 	return r.Client.Login(username, password)
// }
*/

// func (r *RIMAPClient) Select(name string, readOnly bool) (*imap.MailboxStatus, error) {
// 	return r.Client.Select(name, readOnly)
// }

// func (r *RIMAPClient) Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error {
// 	return r.Client.Fetch(seqset, items, ch)
// }

// func (r *RIMAPClient) Logout() error {
// 	return r.Client.Logout()
// }

// func (r *RIMAPClient) Search(criteria *imap.SearchCriteria) ([]uint32, error) {
// 	return r.Client.Search(criteria)
// }

func connectImap(host, user, password string) (IMAPClient, error) {
	log.Println("Connecting to server...")

	client, err := client.DialTLS(host, nil)

	if err != nil {
		return nil, err
	}

	log.Println("Connected")

	if err := client.Login(user, password); err != nil {
		return nil, err
	}

	log.Println("Logged in")

	return client, nil
}

func getListEmail(clt IMAPClient, cfg *Config) ([]*imap.Message, error) {
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)

	go func() {
		done <- clt.List("", "", mailboxes)
	}()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println("* " + m.Name)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	mbox, err := clt.Select("INBOX", false)
	if err != nil {
		return nil, err
	}

	log.Println("Flags for INBOX:", mbox.Flags)

	ids, err := searchBySubject(clt, true, cfg.FromEmail, cfg.FromSubject)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		log.Println()
		return nil, errors.New("нет непрочитанных писем")
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)

	go func() {
		done <- clt.Fetch(seqset, []imap.FetchItem{imap.FetchRFC822}, messages)
	}()

	var resultMessages []*imap.Message

	log.Println("Last 10 messages:")
	for msg := range messages {
		resultMessages = append(resultMessages, msg)
	}

	if err := <-done; err != nil {
		log.Println(err)
	}

	// ----
	files, err := SaveAttachmets(resultMessages, cfg)
	if err != nil {
		log.Fatal(err)
	}

	if len(files) > 0 {
		fmt.Println("Files:")
		print(files)
	}

	log.Println("Done!")

	return resultMessages, nil
}

func searchBySubject(c IMAPClient, seen bool, from, subject string) ([]uint32, error) {
	criteria := imap.NewSearchCriteria()
	criteria.Header = textproto.MIMEHeader{
		"From":    []string{from},
		"Subject": []string{subject},
	}

	if seen {
		criteria.WithoutFlags = []string{"\\Seen"}
	}

	seqNums, err := c.Search(criteria)
	if err != nil {
		return nil, err
	}
	return seqNums, nil
}

func SaveAttachmets(messages []*imap.Message, cfg *Config) ([]string, error) {

	var files []string

	for _, msg := range messages {

		r := msg.GetBody(&imap.BodySectionName{})
		if r == nil {
			log.Println("Не удалось получить тело письма")
			continue
		}

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

		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}

			if err != nil {
				log.Println("Ошибка чтения части письма:", err)
				break
			}

			var filename = "defaultName"

			disp, params, err := part.Header.ContentDisposition()
			if err == nil && (disp == "attachment" || disp == "inline") {
				if fName, ok := params["filename"]; ok {
					filename = fName
				}
			} else {
				continue
			}

			data, err := io.ReadAll(part.Body)
			if err != nil {
				log.Println("Ошибка чтения вложения:", err)
				continue
			}

			log.Printf("Получено вложение: %s, размер: %d байт\n", filename, len(data))

			dir := cfg.WorkDir
			err = os.Mkdir(dir, 0755)
			if err == nil {
				fmt.Printf("Directory '%s' created successfully.\n", dir)
			}

			path := dir + filename
			dst, err := os.Create(path)
			if err != nil {
				log.Println("Ошибка создания файла:", err)
				continue
			}
			defer dst.Close()

			os.WriteFile(path, data, 0644)
			files = append(files, path)
		}
	}

	return files, nil
}
