package notifications

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"task-manager/internal/events"
)

type EmailMessage struct {
	To        []string
	Subject   string
	Body      string
	EventID   string
	EventType events.EventType
}

type EmailSender interface {
	Send(ctx context.Context, message EmailMessage) error
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	Timeout  time.Duration
}

type SMTPEmailSender struct {
	config SMTPConfig
}

func NewSMTPEmailSender(config SMTPConfig) *SMTPEmailSender {
	if config.Timeout <= 0 {
		config.Timeout = 15 * time.Second
	}
	return &SMTPEmailSender{config: config}
}

func (s *SMTPEmailSender) Send(ctx context.Context, message EmailMessage) (err error) {
	ctx, span := tracer.Start(ctx, "EmailSender.Send", trace.WithAttributes(attribute.String("event.type", string(message.EventType))))
	defer func() { finishSpan(span, err) }()

	if len(message.To) == 0 {
		return nil
	}
	address := net.JoinHostPort(s.config.Host, fmt.Sprintf("%d", s.config.Port))
	dialer := net.Dialer{Timeout: s.config.Timeout}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect to SMTP server: %w", err)
	}
	defer connection.Close()

	deadline := time.Now().Add(s.config.Timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set SMTP deadline: %w", err)
	}
	stopClose := context.AfterFunc(ctx, func() { _ = connection.Close() })
	defer stopClose()

	client, err := smtp.NewClient(connection, s.config.Host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}
	if s.config.Username != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate SMTP client: %w", err)
		}
	}
	if err := client.Mail(s.config.From); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	for _, recipient := range message.To {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("set SMTP recipient: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP message body: %w", err)
	}
	if err := writeMessage(w, s.config.From, message); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close SMTP message body: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("finish SMTP session: %w", err)
	}
	return nil
}

func writeMessage(w io.Writer, from string, message EmailMessage) error {
	headerRecipients := make([]string, 0, len(message.To))
	for _, recipient := range message.To {
		headerRecipients = append(headerRecipients, sanitizeHeader(recipient))
	}
	headers := []string{
		"From: " + sanitizeHeader(from),
		"To: " + strings.Join(headerRecipients, ", "),
		"Subject: " + mime.QEncoding.Encode("UTF-8", message.Subject),
		"Message-ID: <" + sanitizeHeader(message.EventID) + "@task-manager.local>",
		"X-Event-ID: " + sanitizeHeader(message.EventID),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}
	writer := bufio.NewWriter(w)
	if _, err := writer.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n" + message.Body); err != nil {
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush SMTP message: %w", err)
	}
	return nil
}

func sanitizeHeader(value string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(value)
}
