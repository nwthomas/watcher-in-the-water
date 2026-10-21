package email

import (
	"bufio"
	"context"
	"encoding/base64"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSMTP is a minimal plaintext SMTP server for a single session.
type fakeSMTP struct {
	listener      net.Listener
	advertiseTLS  bool
	rejectAuth    bool
	rejectRcpt    bool
	silent        bool
	mu            sync.Mutex
	authUser      string
	authPass      string
	mailFrom      string
	rcptTo        string
	data          string
	connections   int
	sessionClosed chan struct{}
}

func newFakeSMTP(t *testing.T, configure func(*fakeSMTP)) *fakeSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeSMTP{listener: ln, sessionClosed: make(chan struct{})}
	if configure != nil {
		configure(f)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go f.serve()
	return f
}

func (f *fakeSMTP) config() Config {
	host, port, _ := net.SplitHostPort(f.listener.Addr().String())
	p, _ := strconv.Atoi(port)
	return Config{
		HostName:      host,
		Port:          p,
		Username:      "me@icloud.com",
		Password:      "app-password",
		PersonalEmail: "me@icloud.com",
	}
}

func (f *fakeSMTP) serve() {
	conn, err := f.listener.Accept()
	if err != nil {
		return
	}
	defer close(f.sessionClosed)
	defer func() { _ = conn.Close() }()
	f.mu.Lock()
	f.connections++
	f.mu.Unlock()
	if f.silent {
		// Hold the connection open without a greeting until the client gives up.
		_, _ = bufio.NewReader(conn).ReadByte()
		return
	}

	tp := textproto.NewConn(conn)
	reply := func(s string) { _ = tp.PrintfLine("%s", s) }
	reply("220 fake ESMTP")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		verb, arg, _ := strings.Cut(line, " ")
		switch strings.ToUpper(verb) {
		case "EHLO":
			if f.advertiseTLS {
				reply("250-fake")
				reply("250-STARTTLS")
			} else {
				reply("250-fake")
			}
			reply("250 AUTH PLAIN")
		case "AUTH":
			_, encoded, _ := strings.Cut(arg, " ")
			decoded, _ := base64.StdEncoding.DecodeString(encoded)
			parts := strings.Split(string(decoded), "\x00")
			if len(parts) == 3 {
				f.mu.Lock()
				f.authUser, f.authPass = parts[1], parts[2]
				f.mu.Unlock()
			}
			if f.rejectAuth {
				reply("535 authentication failed")
				continue
			}
			reply("235 ok")
		case "MAIL":
			f.mu.Lock()
			f.mailFrom = arg
			f.mu.Unlock()
			reply("250 ok")
		case "RCPT":
			if f.rejectRcpt {
				reply("550 no such user")
				continue
			}
			f.mu.Lock()
			f.rcptTo = arg
			f.mu.Unlock()
			reply("250 ok")
		case "DATA":
			reply("354 go ahead")
			lines, err := tp.ReadDotLines()
			if err != nil {
				return
			}
			f.mu.Lock()
			f.data = strings.Join(lines, "\n")
			f.mu.Unlock()
			reply("250 queued")
		case "QUIT":
			reply("221 bye")
			return
		default:
			reply("502 not implemented")
		}
	}
}

func (f *fakeSMTP) waitClosed(t *testing.T) {
	t.Helper()
	select {
	case <-f.sessionClosed:
	case <-time.After(5 * time.Second):
		t.Fatal("smtp session did not finish")
	}
}

func TestSend(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		configure func(*fakeSMTP)
		tls       bool
		ctx       func() (context.Context, context.CancelFunc)
		wantErr   string
		wantSent  bool
	}{
		{
			name:     "delivers message",
			wantSent: true,
		},
		{
			name:      "auth rejected",
			configure: func(f *fakeSMTP) { f.rejectAuth = true },
			wantErr:   "auth",
		},
		{
			name:      "recipient rejected",
			configure: func(f *fakeSMTP) { f.rejectRcpt = true },
			wantErr:   "rcpt to",
		},
		{
			name:    "tls required but starttls not offered",
			tls:     true,
			wantErr: "STARTTLS",
		},
		{
			name:      "context deadline while server is silent",
			configure: func(f *fakeSMTP) { f.silent = true },
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 200*time.Millisecond)
			},
			wantErr: "smtp greeting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			srv := newFakeSMTP(t, tt.configure)
			cfg := srv.config()
			cfg.TLS = tt.tls

			ctx, cancel := context.Background(), context.CancelFunc(func() {})
			if tt.ctx != nil {
				ctx, cancel = tt.ctx()
			}
			defer cancel()

			msg := buildMessage(cfg.PersonalEmail, "192.0.2.1", "192.0.2.2", time.Unix(0, 0), time.Unix(0, 0))
			err := send(ctx, cfg, msg)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("send() err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("send() err = %v", err)
			}
			srv.waitClosed(t)

			srv.mu.Lock()
			defer srv.mu.Unlock()
			if srv.authUser != "me@icloud.com" || srv.authPass != "app-password" {
				t.Fatalf("auth = %q/%q", srv.authUser, srv.authPass)
			}
			if !strings.Contains(srv.mailFrom, "<me@icloud.com>") || !strings.Contains(srv.rcptTo, "<me@icloud.com>") {
				t.Fatalf("envelope from=%q to=%q", srv.mailFrom, srv.rcptTo)
			}
			if !strings.Contains(srv.data, "Subject: Public IP changed: 192.0.2.1 -> 192.0.2.2") {
				t.Fatalf("data missing subject:\n%s", srv.data)
			}
		})
	}
}

func TestNotifyIPChange_disabledDoesNotConnect(t *testing.T) {
	t.Parallel()
	srv := newFakeSMTP(t, nil)
	cfg := srv.config()
	cfg.PersonalEmail = ""

	NotifyIPChange(context.Background(), cfg, "192.0.2.1", "192.0.2.2", time.Now())

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.connections != 0 {
		t.Fatalf("connections = %d, want 0", srv.connections)
	}
}

func TestNotifyIPChange_delivers(t *testing.T) {
	t.Parallel()
	srv := newFakeSMTP(t, nil)

	NotifyIPChange(context.Background(), srv.config(), "192.0.2.1", "192.0.2.2", time.Now())
	srv.waitClosed(t)

	srv.mu.Lock()
	defer srv.mu.Unlock()
	if !strings.Contains(srv.data, "Current IP:  192.0.2.2") {
		t.Fatalf("data = %q", srv.data)
	}
}

func TestConfigIsSetAndValidate(t *testing.T) {
	t.Parallel()
	valid := Config{
		HostName:      "smtp.mail.me.com",
		Port:          587,
		TLS:           true,
		Username:      "me@icloud.com",
		Password:      "app-password",
		PersonalEmail: "me@icloud.com",
	}
	tests := []struct {
		name      string
		cfg       Config
		wantSet   bool
		wantError string
	}{
		{name: "zero config", cfg: Config{Port: 587, TLS: true}, wantSet: false, wantError: "missing"},
		{name: "valid", cfg: valid, wantSet: true},
		{
			name:      "missing password",
			cfg:       func() Config { c := valid; c.Password = ""; return c }(),
			wantSet:   true,
			wantError: "EMAIL_PASSWORD",
		},
		{
			name:      "only host set",
			cfg:       Config{HostName: "smtp.mail.me.com", Port: 587},
			wantSet:   true,
			wantError: "EMAIL_PERSONAL_EMAIL, EMAIL_USERNAME, EMAIL_PASSWORD",
		},
		{
			name:      "invalid port",
			cfg:       func() Config { c := valid; c.Port = 0; return c }(),
			wantSet:   true,
			wantError: "EMAIL_PORT",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.cfg.IsSet(); got != tt.wantSet {
				t.Fatalf("IsSet() = %v, want %v", got, tt.wantSet)
			}
			err := tt.cfg.Validate()
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("Validate() = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Validate() = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestBuildMessage(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	msg := string(buildMessage("me@icloud.com", "192.0.2.1", "192.0.2.2", now, now))

	for _, want := range []string{
		"From: me@icloud.com\r\n",
		"To: me@icloud.com\r\n",
		"Subject: Public IP changed: 192.0.2.1 -> 192.0.2.2\r\n",
		"Date: Wed, 21 Oct 2026 12:00:00 +0000\r\n",
		"@icloud.com>\r\n",
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n",
		"Updated at:  2026-10-21T12:00:00Z\r\n",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q:\n%s", want, msg)
		}
	}
}
