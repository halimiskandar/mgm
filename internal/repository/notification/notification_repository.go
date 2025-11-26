package notification

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pobyzaarif/goshortcute"
)

type MailjetConfig struct {
	MailjetBaseURL           string
	MailjetBasicAuthUsername string
	MailjetBasicAuthPassword string
	MailjetSenderEmail       string
	MailjetSenderName        string
}

type MailjetRepository struct {
	mailjetConfig MailjetConfig
}

func NewMailjetRepository(cfg MailjetConfig) *MailjetRepository {
	return &MailjetRepository{
		cfg,
	}
}

type payloadSendEmail struct {
	Messages []Messages `json:"Messages"`
}

type From struct {
	Email string `json:"Email"`
	Name  string `json:"Name"`
}

type To struct {
	Email string `json:"Email"`
	Name  string `json:"Name"`
}

type Messages struct {
	From     From   `json:"From"`
	To       []To   `json:"To"`
	Subject  string `json:"Subject"`
	TextPart string `json:"TextPart"`
	HTMLPart string `json:"HTMLPart"`
}

func (r MailjetRepository) SendEmail(toName, toEmail, subject, message string) (err error) {
	url := r.mailjetConfig.MailjetBaseURL + "/v3.1/send"
	method := http.MethodPost

	toBody := []To{}
	toBody = append(toBody, To{
		Email: toEmail,
		Name:  toName,
	})

	messageBody := Messages{
		To: toBody,
		From: From{
			Email: r.mailjetConfig.MailjetSenderEmail,
			Name:  r.mailjetConfig.MailjetSenderName,
		},
		Subject:  subject,
		TextPart: message,
		HTMLPart: message,
	}
	constructMessages := []Messages{}
	constructMessages = append(constructMessages, messageBody)

	payload := payloadSendEmail{
		Messages: constructMessages,
	}

	payloadByte, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal json payload: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(method, url, strings.NewReader(string(payloadByte)))
	if err != nil {
		return err
	}

	buildBasicAuth := goshortcute.StringtoBase64Encode(r.mailjetConfig.MailjetBasicAuthUsername + ":" + r.mailjetConfig.MailjetBasicAuthPassword)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+buildBasicAuth)

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return nil
	}
	bodyBytes, _ := io.ReadAll(res.Body)
	fmt.Println("Mailjet Response:", string(bodyBytes))

	return fmt.Errorf("mailer service return negative response %v", res.StatusCode)
}
