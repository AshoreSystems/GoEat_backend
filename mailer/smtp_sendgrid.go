package mailer

import (
	"fmt"
	"net/smtp"
)

func SendOTPviaSMTP(sendgridKey, fromEmail, toEmail, subject, body string) error {

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
