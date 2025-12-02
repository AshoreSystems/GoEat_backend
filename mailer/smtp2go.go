package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Config holds your SMTP2GO API configuration
type Config struct {
	APIKey string
	From   string
}

// EmailRequest represents the request payload for SMTP2GO API
type EmailRequest struct {
	Recipients []string `json:"to"`
	Subject    string   `json:"subject"`
	HTML       string   `json:"html,omitempty"`
	Text       string   `json:"text_body,omitempty"`
	From       string   `json:"sender"`
}

// SendEmail sends an email via SMTP2GO HTTP API
func SendEmail(cfg Config, to []string, subject, body string) error {
	payload := EmailRequest{
		Recipients: to,
		Subject:    subject,
		Text:       body,
		From:       cfg.From,
	}

	data, err := json.Marshal(payload)
	fmt.Println(string(data))
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.smtp2go.com/v3/email/send", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Smtp2go-Api-Key", cfg.APIKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SMTP2GO API returned status: %s", resp.Status)
	}

	fmt.Println("Email sent successfully via SMTP2GO API!")
	return nil
}
