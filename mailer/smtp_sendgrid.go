package mailer

import (
	"fmt"
	"net/smtp"
	"os"
)

func SendOTPviaSMTP(toEmail, subject, body string) error {

	sendgridKey := os.Getenv("SENDGRID_API_KEY")
	fromEmail := os.Getenv("SENDGRID_FROM_EMAIL")
	if sendgridKey == "" {
		return fmt.Errorf("SENDGRID_API_KEY not set")
	}
	// SendGrid SMTP auth
	auth := smtp.PlainAuth(
		"",          // identity, keep empty
		"apikey",    // ALWAYS this username for SendGrid
		sendgridKey, // your SendGrid API key
		"smtp.sendgrid.net",
	)

	msg := []byte(
		"From: " + fromEmail + "\r\n" +
			"To: " + toEmail + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/plain; charset=\"utf-8\"\r\n" +
			"\r\n" +
			body + "\r\n",
	)

	// Send email
	err := smtp.SendMail(
		"smtp.sendgrid.net:587",
		auth,
		fromEmail,
		[]string{toEmail},
		msg,
	)

	if err != nil {
		return fmt.Errorf("SMTP send error: %v", err)
	}

	return nil
}

func SendHTMLEmail(toEmail, subject, htmlBody string) error {

	sendgridKey := os.Getenv("SENDGRID_API_KEY")
	fromEmail := os.Getenv("SENDGRID_FROM_EMAIL")

	if sendgridKey == "" {
		return fmt.Errorf("SENDGRID_API_KEY not set")
	}

	// SendGrid SMTP auth
	auth := smtp.PlainAuth(
		"",          // identity
		"apikey",    // SendGrid requires this as username
		sendgridKey, // API Key as password
		"smtp.sendgrid.net",
	)

	msg := []byte(
		"From: " + fromEmail + "\r\n" +
			"To: " + toEmail + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
			"\r\n" +
			htmlBody + "\r\n",
	)

	err := smtp.SendMail(
		"smtp.sendgrid.net:587",
		auth,
		fromEmail,
		[]string{toEmail},
		msg,
	)

	if err != nil {
		return fmt.Errorf("SMTP HTML send error: %v", err)
	}

	return nil
}
