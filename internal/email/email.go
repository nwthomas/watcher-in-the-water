package email

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

const sendTimeout = 30 * time.Second

// Config holds SMTP settings for change notifications. A zero Config disables email.
type Config struct {
	HostName      string
	Port          int
	TLS           bool
	Username      string
	Password      string
	PersonalEmail string
}

// IsSet reports whether any email setting was provided, which means email is intended to be on.
func (c Config) IsSet() bool {
	return c.HostName != "" || c.PersonalEmail != "" || c.Username != "" || c.Password != ""
}

// Validate reports missing or invalid settings required to send email.
func (c Config) Validate() error {
	var missing []string
	if c.HostName == "" {
		missing = append(missing, "EMAIL_HOST_NAME")
	}
	if c.PersonalEmail == "" {
		missing = append(missing, "EMAIL_PERSONAL_EMAIL")
	}
	if c.Username == "" {
		missing = append(missing, "EMAIL_USERNAME")
	}
	if c.Password == "" {
		missing = append(missing, "EMAIL_PASSWORD")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing %s", strings.Join(missing, ", "))
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid EMAIL_PORT %d", c.Port)
	}
	return nil
}

// NotifyIPChange emails PersonalEmail (as both sender and recipient) with the change details.
// Errors are logged; a zero Config is a no-op.
func NotifyIPChange(ctx context.Context, cfg Config, previousIP, currentIP string, updatedAt time.Time) {
	if cfg.HostName == "" || cfg.PersonalEmail == "" {
		return
	}

	msg := buildMessage(cfg.PersonalEmail, previousIP, currentIP, updatedAt, time.Now())
	if err := send(ctx, cfg, msg); err != nil {
		slog.Error("email send failed", "host", cfg.HostName, "port", cfg.Port, "err", err)
		return
	}
	slog.Info("email delivered", "host", cfg.HostName, "port", cfg.Port)
}

// send delivers msg over a single SMTP session, upgrading with STARTTLS when cfg.TLS is set.
func send(ctx context.Context, cfg Config, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	addr := net.JoinHostPort(cfg.HostName, strconv.Itoa(cfg.Port))
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		if err = conn.SetDeadline(deadline); err != nil {
			closeConn(conn, cfg.HostName)
			return fmt.Errorf("set deadline: %w", err)
		}
	}
	// Unblock any in-flight read or write if ctx is cancelled mid-session.
	stop := context.AfterFunc(ctx, func() { closeConn(conn, cfg.HostName) })
	defer stop()

	client, err := smtp.NewClient(conn, cfg.HostName)
	if err != nil {
		closeConn(conn, cfg.HostName)
		return fmt.Errorf("smtp greeting: %w", err)
	}
	defer func() {
		if cerr := client.Close(); cerr != nil {
			slog.Debug("email close connection", "host", cfg.HostName, "err", cerr)
		}
	}()

	if cfg.TLS {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("server does not support STARTTLS")
		}
		if err = client.StartTLS(&tls.Config{ServerName: cfg.HostName, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}
	if ok, _ := client.Extension("AUTH"); !ok {
		return errors.New("server does not support AUTH")
	}
	if err = client.Auth(smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.HostName)); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err = client.Mail(cfg.PersonalEmail); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err = client.Rcpt(cfg.PersonalEmail); err != nil {
		return fmt.Errorf("rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("end data: %w", err)
	}
	if err = client.Quit(); err != nil {
		return fmt.Errorf("quit: %w", err)
	}
	return nil
}

// closeConn closes conn, logging any error at debug level.
func closeConn(conn net.Conn, host string) {
	if err := conn.Close(); err != nil {
		slog.Debug("email close connection", "host", host, "err", err)
	}
}

// buildMessage returns an RFC 5322 plain-text message addressed from and to address.
func buildMessage(address, previousIP, currentIP string, updatedAt, now time.Time) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", address)
	fmt.Fprintf(&b, "To: %s\r\n", address)
	fmt.Fprintf(&b, "Subject: Public IP changed: %s -> %s\r\n", previousIP, currentIP)
	fmt.Fprintf(&b, "Date: %s\r\n", now.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: %s\r\n", messageID(address, now))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString("Your public IP address changed.\r\n\r\n")
	fmt.Fprintf(&b, "Previous IP: %s\r\n", previousIP)
	fmt.Fprintf(&b, "Current IP:  %s\r\n", currentIP)
	fmt.Fprintf(&b, "Updated at:  %s\r\n", updatedAt.UTC().Format(time.RFC3339))
	return b.Bytes()
}

// messageID builds a unique Message-ID using the domain of address.
func messageID(address string, now time.Time) string {
	domain := "watcher-in-the-water"
	if i := strings.LastIndex(address, "@"); i >= 0 && i < len(address)-1 {
		domain = address[i+1:]
	}
	random := make([]byte, 8)
	_, _ = rand.Read(random)
	return fmt.Sprintf("<%d.%s@%s>", now.UnixNano(), hex.EncodeToString(random), domain)
}
