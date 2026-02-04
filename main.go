package main

import (
	"fmt"
	"log"

	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
)

type Config struct {
	ImapServer  string `env:"IMAP_SERVER"`
	ImapUser    string `env:"IMAP_USER"`
	ImapPass    string `env:"IMAP_PASS"`
	FromEmail   string `env:"FROM_EMAIL"`
	FromSubject string `env:"FROM_SUBJECT"`
	WorkDir     string `env:"WORK_DIR"`
}

var (
	Version   string
	BuildTime string
)

func main() {

	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := Config{}
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("Error load enviroment variables in .env file: %v", err)
	}

	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Build Time: %s\n", BuildTime)

	clientImap, err := connectImap(cfg.ImapServer, cfg.ImapUser, cfg.ImapPass)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}
	defer clientImap.Logout()

	messages, err := getListEmail(clientImap, &cfg)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}

	for i, _ := range messages {
		fmt.Printf("Message #%d\n", i+1)
	}
}
