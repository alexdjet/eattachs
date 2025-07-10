package main

import (
	"fmt"
	"log"
)

func main() {

	clientImap, err := connectImap("imap.server.net:993", "user@server.net", "password")
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	defer clientImap.Logout()

	messages, err := getListEmail(clientImap)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}

	for i, _ := range messages {
		fmt.Printf("Message #%d\n", i+1)

		// fmt.Println("Body:", string(m.Envelope.Subject))
		// fmt.Println("------")
	}

	fmt.Println("------")
}
