package mail

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fcs-autoreport/internal/domain"
)

const MohRecipient = "vet_control@moh.gov.il"

var ReplyCCRecipients = []string{
	"office@dolina.co.il",
	"alexei.dolina@gmail.com",
	"rode.19917@gmail.com",
}

func SendReport(settings domain.Settings, reportPath string) (string, error) {
	missing := make([]string, 0, 4)
	if strings.TrimSpace(settings.SMTPHost) == "" {
		missing = append(missing, "SMTPHost")
	}
	if strings.TrimSpace(settings.SMTPUser) == "" {
		missing = append(missing, "SMTPUser")
	}
	if strings.TrimSpace(settings.SMTPPassword) == "" {
		missing = append(missing, "SMTPPassword")
	}
	if settings.SMTPPort <= 0 {
		missing = append(missing, "SMTPPort")
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("smtp не настроен: отсутствуют поля %s", strings.Join(missing, ", "))
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		return "", fmt.Errorf("чтение вложения: %w", err)
	}
	fileName := filepath.Base(reportPath)
	subject := fmt.Sprintf("FCS report %s", fileName)
	boundary := fmt.Sprintf("fcs-boundary-%d", time.Now().UnixNano())
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", settings.SMTPUser))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", MohRecipient))
	msg.WriteString(fmt.Sprintf("Reply-To: %s\r\n", strings.Join(ReplyCCRecipients, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n", boundary))
	msg.WriteString("\r\n")
	msg.WriteString("--" + boundary + "\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
	msg.WriteString("Автоматическая отправка FCS отчета.\r\n\r\n")
	msg.WriteString("--" + boundary + "\r\n")
	msg.WriteString("Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet\r\n")
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", fileName))
	b := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(b, data)
	for i := 0; i < len(b); i += 76 {
		end := i + 76
		if end > len(b) {
			end = len(b)
		}
		msg.Write(b[i:end])
		msg.WriteString("\r\n")
	}
	msg.WriteString("--" + boundary + "--\r\n")

	addr := fmt.Sprintf("%s:%d", settings.SMTPHost, settings.SMTPPort)
	auth := smtp.PlainAuth("", settings.SMTPUser, settings.SMTPPassword, settings.SMTPHost)
	recipients := []string{MohRecipient}
	if settings.SMTPPort == 465 {
		return subject, sendTLS(addr, settings.SMTPHost, settings.SMTPUser, settings.SMTPPassword, recipients, []byte(msg.String()))
	}
	if err := smtp.SendMail(addr, auth, settings.SMTPUser, recipients, []byte(msg.String())); err != nil {
		return "", fmt.Errorf("smtp send: %w", err)
	}
	return subject, nil
}

func sendTLS(addr, host, user, pass string, recipients []string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	defer conn.Close()
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	auth := smtp.PlainAuth("", user, pass, host)
	if err := c.Auth(auth); err != nil {
		return err
	}
	if err := c.Mail(user); err != nil {
		return err
	}
	for _, rcpt := range recipients {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

func ParseReplyText(body string) []domain.ApprovalResult {
	lines := strings.Split(body, "\n")
	out := make([]domain.ApprovalResult, 0)
	for _, ln := range lines {
		parts := strings.Split(strings.TrimSpace(ln), " - ")
		if len(parts) < 3 {
			continue
		}
		invoice := strings.TrimSpace(parts[1])
		reason := strings.TrimSpace(strings.Join(parts[2:], " - "))
		if invoice == "" {
			continue
		}
		out = append(out, domain.ApprovalResult{
			InvoiceNum:   invoice,
			Status:       "rejected",
			RejectReason: reason,
		})
	}
	return out
}
