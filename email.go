package main

import (
	"fmt"
	"net"
	"net/smtp"
)

func sendEmail(config *Config, subject, body string) error {
	auth := smtp.PlainAuth("", config.SenderEmail, config.SenderPassword, config.SMTPServer)
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n", config.RecipientEmail, subject, body))

	err := smtp.SendMail(
		net.JoinHostPort(config.SMTPServer, config.SMTPPort),
		auth,
		config.SenderEmail,
		[]string{config.RecipientEmail},
		msg,
	)
	return err
}
