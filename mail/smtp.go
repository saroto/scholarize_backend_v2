package mail

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"

	"github.com/spf13/viper"
)

var (
	emailTemplate *template.Template
)

type EmailTemplateData struct {
	PreviewHeader string
	EmailPurpose  string
	ActionURL     string
	Action        string
	EmailEnding   string
}

func init() {
	var err error
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get root directory: %v", err)
	}
	emailTemplate, err = loadTemplate(rootDir + "/mail/template.html")
	if err != nil {
		log.Fatalf("Failed to load email template: %v", err)
	}
}

func loadTemplate(filepath string) (*template.Template, error) {
	htmlData, err := os.ReadFile(filepath)
	if err != nil {
		log.Printf("Error reading template file: %v", err)
		return nil, err
	}
	tmpl, err := template.New("email").Parse(string(htmlData))
	if err != nil {
		log.Printf("Error parsing template: %v", err)
		return nil, err
	}
	return tmpl, nil
}

func CustomizeHTML(data EmailTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := emailTemplate.Execute(&buf, data); err != nil {
		log.Printf("Error executing template: %v", err)
		return "", err
	}
	return buf.String(), nil
}

func SendEmail(to, subject, body string) error {
	smtpHost := viper.GetString("mailsmtp.host")
	smtpPort := viper.GetString("mailsmtp.port")
	smtpUser := viper.GetString("mailsmtp.email")
	smtpPass := viper.GetString("mailsmtp.password")

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	addr := smtpHost + ":" + smtpPort

	header := "From: " + smtpUser + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0;\r\n" +
		"Content-Type: text/html; charset=UTF-8;\r\n\r\n"

	msg := []byte(header + body)

	fmt.Printf("Sending email to %s\n", to)

	if err := smtp.SendMail(addr, auth, smtpUser, []string{to}, msg); err != nil {
		fmt.Printf("Error sending email: %v", err)
		return err
	}
	return nil
}

func SendSMTPtoEmail(emailBody EmailTemplateData, email, subject string) error {
	// Customize the email template
	emailBodyData, err := CustomizeHTML(emailBody)
	if err != nil {
		return fmt.Errorf("error customizing email: %v", err)
	}

	// Send email to user
	errSending := SendEmail(email, subject, emailBodyData)
	if errSending != nil {
		return fmt.Errorf("error sending email to user %s: %v", email, errSending)
	}
	fmt.Printf("Email subject: '%s' was sent to user %s\n", subject, email)
	return nil
}
