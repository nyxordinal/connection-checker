package main

import (
	"fmt"
	"os"
)

func loadHTMLTemplate(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getHTMLTemplates() (string, string, error) {
	alertHtml, err := loadHTMLTemplate("email_alert.html")
	if err != nil {
		return "", "", fmt.Errorf("failed to load alert email template: %w", err)
	}

	restoredHtml, err := loadHTMLTemplate("email_restored.html")
	if err != nil {
		return "", "", fmt.Errorf("failed to load restored email template: %w", err)
	}

	return alertHtml, restoredHtml, nil
}
