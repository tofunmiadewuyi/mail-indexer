package parser

import (
	"os"
	"strings"
	"time"

	"github.com/jhillyerd/enmime"
)

type Email struct {
	MessageID   string
	User        string
	Subject     string
	From        string
	To          []string
	Date        time.Time
	Body        string
	Attachments []Attachment
}

type Attachment struct {
	Filename    string
	ContentType string
	Size        int
	Data        []byte
}

type Parser struct {
	user       string
	BeforeDate time.Time
}

func New(user string, beforeDate time.Time) *Parser {
	return &Parser{
		user:       user,
		BeforeDate: beforeDate,
	}
}

func (p *Parser) ParseFile(filePath string) (*Email, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	envelope, err := enmime.ReadEnvelope(file)
	if err != nil {
		return nil, err
	}

	emailDate, err := envelope.Date()
	if err != nil || emailDate.IsZero() {
		emailDate = fileInfo.ModTime()
	}

	email := &Email{
		MessageID: envelope.GetHeader("Message-ID"),
		User:      p.user,
		Subject:   envelope.GetHeader("Subject"),
		From:      envelope.GetHeader("From"),
		To:        strings.Split(envelope.GetHeader("To"), ","),
		Date:      emailDate,
		Body:      envelope.Text,
	}

	for _, att := range envelope.Attachments {
		email.Attachments = append(email.Attachments, Attachment{
			Filename:    att.FileName,
			ContentType: att.ContentType,
			Size:        len(att.Content),
			Data:        att.Content,
		})
	}

	return email, nil
}

// checks if email should be indexed based on date
func (p *Parser) ShouldIndex(email *Email) bool {
	return email.Date.Before(p.BeforeDate)
}
