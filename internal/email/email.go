package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const DEFAULT_TIMEOUT = 15 * time.Second

// Config contains the SMTP settings required to send IP change emails.
type Config struct {
	HostName      string
	Password      string
	PersonalEmail string
	Port          int
	TLS           bool
	Username      string
}

// RawConfig mirrors the EMAIL_* environment variables before validation.
type RawConfig struct {
	HostName      string
	Password      string
	PersonalEmail string
	Port          string
	TLS           string
	Username      string
}

// SMTPNotifier sends IP change notifications through an authenticated SMTP server.
type SMTPNotifier struct {
	Config Config
}

// ParseConfig validates the required EMAIL_* settings.
func ParseConfig(raw RawConfig) (Config, error) {
	missing := missingFields(raw)
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required email environment variables: %s", strings.Join(missing, ", "))
	}

	port, err := strconv.Atoi(strings.TrimSpace(raw.Port))
	if err != nil {
		return Config{}, fmt.Errorf("parse EMAIL_PORT: %w", err)
	}
	if port <= 0 || port > 65535 {
		return Config{}, fmt.Errorf("EMAIL_PORT must be between 1 and 65535")
	}

	useTLS, err := strconv.ParseBool(strings.TrimSpace(raw.TLS))
	if err != nil {
		return Config{}, fmt.Errorf("parse EMAIL_TLS: %w", err)
	}

	usernameAddress, err := mail.ParseAddress(strings.TrimSpace(raw.Username))
	if err != nil {
		return Config{}, fmt.Errorf("parse EMAIL_USERNAME: %w", err)
	}

	personalEmailAddress, err := mail.ParseAddress(strings.TrimSpace(raw.PersonalEmail))
	if err != nil {
		return Config{}, fmt.Errorf("parse EMAIL_PERSONAL_EMAIL: %w", err)
	}

	return Config{
		HostName:      strings.TrimSpace(raw.HostName),
		Password:      raw.Password,
		PersonalEmail: personalEmailAddress.Address,
		Port:          port,
		TLS:           useTLS,
		Username:      usernameAddress.Address,
	}, nil
}

// NotifyIPChange sends a plain text email describing the observed public IP change.
func (n SMTPNotifier) NotifyIPChange(ctx context.Context, previousIP, currentIP string, updatedAt time.Time) error {
	return SendIPChange(ctx, n.Config, previousIP, currentIP, updatedAt)
}

// SendIPChange sends a plain text email describing the observed public IP change.
func SendIPChange(ctx context.Context, cfg Config, previousIP, currentIP string, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	addr := net.JoinHostPort(cfg.HostName, strconv.Itoa(cfg.Port))
	dialer := &net.Dialer{Timeout: DEFAULT_TIMEOUT}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("connect SMTP server: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Debug("smtp connection close failed", "err", closeErr)
		}
	}()

	client, err := smtp.NewClient(conn, cfg.HostName)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer func() {
		if closeErr := client.Close(); closeErr != nil {
			slog.Debug("smtp client close failed", "err", closeErr)
		}
	}()

	if cfg.TLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not advertise STARTTLS")
		}
		if opErr := client.StartTLS(&tls.Config{ServerName: cfg.HostName, MinVersion: tls.VersionTLS12}); opErr != nil {
			return fmt.Errorf("start TLS: %w", opErr)
		}
	}

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.HostName)
	if opErr := client.Auth(auth); opErr != nil {
		return fmt.Errorf("authenticate SMTP: %w", opErr)
	}
	if opErr := client.Mail(cfg.Username); opErr != nil {
		return fmt.Errorf("set SMTP sender: %w", opErr)
	}
	if opErr := client.Rcpt(cfg.PersonalEmail); opErr != nil {
		return fmt.Errorf("set SMTP recipient: %w", opErr)
	}

	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("open SMTP data writer: %w", err)
	}

	_, writeErr := wc.Write(buildMessage(cfg, previousIP, currentIP, updatedAt))
	closeErr := wc.Close()
	if writeErr != nil {
		return fmt.Errorf("write SMTP message: %w", writeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close SMTP data writer: %w", closeErr)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("quit SMTP session: %w", err)
	}

	return nil
}

func missingFields(raw RawConfig) []string {
	checks := []struct {
		name  string
		value string
	}{
		{name: "EMAIL_HOST_NAME", value: raw.HostName},
		{name: "EMAIL_PASSWORD", value: raw.Password},
		{name: "EMAIL_PERSONAL_EMAIL", value: raw.PersonalEmail},
		{name: "EMAIL_PORT", value: raw.Port},
		{name: "EMAIL_TLS", value: raw.TLS},
		{name: "EMAIL_USERNAME", value: raw.Username},
	}

	var missing []string
	for _, check := range checks {
		if strings.TrimSpace(check.value) == "" {
			missing = append(missing, check.name)
		}
	}
	return missing
}

func buildMessage(cfg Config, previousIP, currentIP string, updatedAt time.Time) []byte {
	updatedAt = updatedAt.UTC()
	subject := fmt.Sprintf("Watcher in the Water: public IP changed to %s", currentIP)
	body := fmt.Sprintf("The public IP address changed.\r\n\r\nPrevious IP: %s\r\nCurrent IP: %s\r\nUpdated at: %s\r\n",
		previousIP,
		currentIP,
		updatedAt.Format(time.RFC3339),
	)

	headers := []string{
		"From: " + cfg.Username,
		"To: " + cfg.PersonalEmail,
		"Subject: " + subject,
		"Date: " + updatedAt.Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
	}

	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body)
}
