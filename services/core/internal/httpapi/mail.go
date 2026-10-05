package httpapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"tutor-platform/services/core/internal/config"
)

func sendResetEmail(ctx context.Context, cfg config.Config, recipient, token string) error {
	if cfg.SMTPAddr == "" {
		return fmt.Errorf("SMTP_ADDR is not configured")
	}
	host, _, err := net.SplitHostPort(cfg.SMTPAddr)
	if err != nil {
		return err
	}
	conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", cfg.SMTPAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if cfg.AppEnv == "production" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP STARTTLS is required in production")
		}
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if cfg.SMTPUser != "" {
		if err := client.Auth(smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, host)); err != nil {
			return err
		}
	}
	if err := client.Mail(cfg.SMTPFrom); err != nil {
		return err
	}
	if err := client.Rcpt(recipient); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	message := strings.Join([]string{
		"From: " + cfg.SMTPFrom,
		"To: " + recipient,
		"Subject: Repet password reset",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		"Open the password reset page and enter this one-time code:",
		"",
		token,
		"",
		"The code expires in 30 minutes.",
		"",
	}, "\r\n")
	if _, err := writer.Write([]byte(message)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
