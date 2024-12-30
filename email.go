package main

import (
	"fmt"
	"net"
	"net/smtp"
	"os"
	"time"
)

func loadHTMLTemplate(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getHTMLTemplates() (string, string, error) {
	alertHtml, err := loadHTMLTemplate("./email/email_alert.html")
	if err != nil {
		return "", "", fmt.Errorf("failed to load alert email template: %w", err)
	}

	restoredHtml, err := loadHTMLTemplate("./email/email_restored.html")
	if err != nil {
		return "", "", fmt.Errorf("failed to load restored email template: %w", err)
	}

	return alertHtml, restoredHtml, nil
}

func sendAlertEmail(subject, content string) {
	timestamp := time.Now().UTC().Format("2006-01-02 15:04:05 MST")

	auth := smtp.PlainAuth("", appCtx.Config.SenderEmail, appCtx.Config.SenderPassword, appCtx.Config.SMTPServer)
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n", appCtx.Config.RecipientEmail, subject, fmt.Sprintf(content, appCtx.Config.TargetIP, timestamp)))

	err := smtp.SendMail(
		net.JoinHostPort(appCtx.Config.SMTPServer, appCtx.Config.SMTPPort),
		auth,
		appCtx.Config.SenderEmail,
		[]string{appCtx.Config.RecipientEmail},
		msg,
	)
	if err != nil {
		appCtx.Logger.Errorf("Failed to send email: %v", err)
		return
	}

	if err := appCtx.DB.updateLastSentEmail(timestamp); err != nil {
		appCtx.Logger.Errorf("Failed to update last sent email timestamp in DB: %v", err)
	} else {
		appCtx.Logger.Info(fmt.Sprintf("%s email sent", subject))
	}
}
