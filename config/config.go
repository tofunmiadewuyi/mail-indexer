package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ESHost       string
	ESIndex      string
	MailBasePath string
	Account      string
	Domain       string
	User         string
	BeforeDate   time.Time
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func New(user, domain string, account string, beforeDate time.Time) *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, or could not find it")
	}

	return &Config{
		ESHost:       getEnv("ESHost", "http://localhost:9200"),
		ESIndex:      getEnv("ESIndex", "mail-archive"),
		MailBasePath: getEnv("MailBasePath", "/home"),
		Domain:       domain,
		Account:      account,
		User:         user,
		BeforeDate:   beforeDate,
	}
}

// GetMailPath returns full path to user's maildir
// e.g., /home/cpanel_user/mail/domain/user/
func (c *Config) GetMailPath() string {
	return c.MailBasePath + "/" + c.Account + "/mail/" + c.Domain + "/" + c.User + "/"
}
