package mail

import (
	"fmt"
	"net/smtp"
)

// Mailer defines the interface for sending emails.
type Mailer interface {
	Send(to, subject, body string) error
}

// SMTPMailer implements Mailer using net/smtp.
type SMTPMailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func NewSMTPMailer(host, port, username, password, from string) *SMTPMailer {
	return &SMTPMailer{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
	}
}

func (m *SMTPMailer) Send(to, subject, body string) error {
	// Authentication for SMTP
	auth := smtp.PlainAuth("", m.Username, m.Password, m.Host)

	// Combine headers and body
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", to, subject, body))

	addr := fmt.Sprintf("%s:%s", m.Host, m.Port)
	return smtp.SendMail(addr, auth, m.From, []string{to}, msg)
}

// LogMailer is a mock mailer for local development that just prints to logs.
type LogMailer struct{}

func (m *LogMailer) Send(to, subject, body string) error {
	fmt.Printf("\n--- [EMAIL LOG] ---\nTo: %s\nSubject: %s\nBody: %s\n-------------------\n", to, subject, body)
	return nil
}
