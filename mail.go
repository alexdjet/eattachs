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

func connectImap(host, user, password string) (*client.Client, error) {
	log.Println("Connecting to server...")

	client, err := client.DialTLS(host, nil)

	if err != nil {
		return nil, err
	}

	log.Println("Connected")

	// Login
	if err := client.Login(user, password); err != nil {
		return nil, err
	}

	log.Println("Logged in")

	return client, nil
}

func getListEmail(clt *client.Client) ([]*imap.Message, error) {
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

	// Select INBOX
	mbox, err := clt.Select("INBOX", false)
	if err != nil {
		return nil, err
	}

	log.Println("Flags for INBOX:", mbox.Flags)

	ids, err := searchBySubject(clt, true, "subuser@server.net", "Subject")
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		log.Println()
		return nil, errors.New("нет непрочитанных писем")
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	// var criteria = imap.NewSearchCriteria()
	// criteria.WithoutFlags = []string{"\\Seen"}
	// criteria.Header = textproto.MIMEHeader{
	// 	"From":    []string{"subuser@server.net"},
	// 	"Subject": []string{"Subject"},
	// }

	// seqNums, err := clt.Search(criteria)
	// if err != nil {
	// 	return nil, err
	// }

	//// Get the last 4 messages
	// start := uint32(1)
	// stop := mbox.Messages

	// if mbox.Messages > 3 {
	// 	start = mbox.Messages - 3
	// }

	// seqset := new(imap.SeqSet)
	// seqset.AddRange(start, stop)

	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)

	go func() {
		done <- clt.Fetch(seqset, []imap.FetchItem{imap.FetchRFC822}, messages)
	}()

	var resultMessages []*imap.Message

	log.Println("Last 10 messages:")
	for msg := range messages {
		// log.Println("* " + msg.Envelope.Subject)
		resultMessages = append(resultMessages, msg)
	}

	if err := <-done; err != nil {
		log.Println(err)
	}

	// ----
	files, err := GetAttachmets(resultMessages)
	if err != nil {
		log.Fatal(err)
	}

	if len(files) > 0 {
		fmt.Println("Files:")
		print(files)
	}

	// ----

	log.Println("Done!")

	return resultMessages, nil
}

func searchBySubject(c *client.Client, seen bool, from, subject string) ([]uint32, error) {
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

func GetAttachmets(messages []*imap.Message) ([]string, error) {

	var files []string

	for _, msg := range messages {

		// Получаем тело письма в формате RFC822
		r := msg.GetBody(&imap.BodySectionName{})
		if r == nil {
			log.Println("Не удалось получить тело письма")
			continue
		}

		// Парсим MIME-сообщение
		entity, err := message.Read(r)
		if err != nil {
			log.Println("Ошибка парсинга MIME:", err)
			continue
		}

		// Если письмо multipart (с вложениями)
		mr := entity.MultipartReader()
		if mr == nil {
			log.Println("Письмо не multipart, вложений нет")
			continue
		}

		// Перебираем части письма
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

			// Получаем заголовок Content-Disposition, чтобы проверить, является ли часть вложением
			disp, params, err := part.Header.ContentDisposition()
			if err == nil && (disp == "attachment" || disp == "inline") {
				if fName, ok := params["filename"]; ok {
					filename = fName
				}
			} else {
				continue
			}

			// // Если не нашли, пробуем из Content-Type
			// _, params, err = mime.ParseMediaType(header.Get("Content-Type"))
			// if err == nil {
			// 	if name, ok := params["name"]; ok {
			// 		return name
			// 	}
			// }

			// // Получаем имя файла вложения
			// filename, _ := part.FileName()
			// if filename == "" {
			// 	filename = "unknown"
			// }

			// Читаем содержимое вложения
			data, err := io.ReadAll(part.Body)
			if err != nil {
				log.Println("Ошибка чтения вложения:", err)
				continue
			}

			// Здесь можно сохранить data в файл или базу данных
			log.Printf("Получено вложение: %s, размер: %d байт\n", filename, len(data))

			dir := "./data/"
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

			// Например, сохранить на диск:
			os.WriteFile(path, data, 0644)

			files = append(files, path)
		}
	}

	return files, nil
}
