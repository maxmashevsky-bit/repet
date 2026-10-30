package httpapi

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"tutor-platform/services/core/internal/config"
)

func sendResetEmail(ctx context.Context, cfg config.Config, recipient, token string) error {
	return sendEmail(ctx, cfg, recipient, "Репет: восстановление пароля", "Откройте "+strings.TrimRight(cfg.AppPublicURL, "/")+"/reset-password и введите одноразовый код:\n\n"+token+"\n\nКод действует 30 минут. Если вы не запрашивали смену пароля, проигнорируйте письмо.")
}

func sendVerificationEmail(ctx context.Context, cfg config.Config, recipient, token string) error {
	return sendEmail(ctx, cfg, recipient, "Репет: подтвердите почту", "Подтвердите адрес электронной почты для входа в «Репет»:\n\n"+strings.TrimRight(cfg.AppPublicURL, "/")+"/verify-email#token="+token+"\n\nСсылка действует один час. Откройте её в браузере, где вы вошли в свой аккаунт. Если вы не регистрировались, проигнорируйте письмо.")
}

func sendEmail(ctx context.Context, cfg config.Config, recipient, subject, body string) error {
	for _, address := range []string{cfg.SMTPFrom, recipient} {
		parsed, err := mail.ParseAddress(address)
		if err != nil || parsed.Address != address || strings.ContainsAny(address, "\r\n") {
			return fmt.Errorf("invalid email address")
		}
	}
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
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	} else if cfg.AppEnv == "production" {
		return fmt.Errorf("SMTP STARTTLS is required in production")
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
		"Subject: " + mime.QEncoding.Encode("UTF-8", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		strings.ReplaceAll(body, "\n", "\r\n"),
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
