package email

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     RawConfig
		want    Config
		wantErr string
	}{
		{
			name: "valid config",
			raw: RawConfig{
				HostName:      " smtp.example.com ",
				Password:      "app-password",
				PersonalEmail: "me@example.com",
				Port:          "587",
				TLS:           "true",
				Username:      "sender@example.com",
			},
			want: Config{
				HostName:      "smtp.example.com",
				Password:      "app-password",
				PersonalEmail: "me@example.com",
				Port:          587,
				TLS:           true,
				Username:      "sender@example.com",
			},
		},
		{
			name:    "missing required fields",
			raw:     RawConfig{},
			wantErr: "EMAIL_HOST_NAME",
		},
		{
			name: "invalid port",
			raw: RawConfig{
				HostName:      "smtp.example.com",
				Password:      "app-password",
				PersonalEmail: "me@example.com",
				Port:          "many",
				TLS:           "true",
				Username:      "sender@example.com",
			},
			wantErr: "parse EMAIL_PORT",
		},
		{
			name: "invalid tls",
			raw: RawConfig{
				HostName:      "smtp.example.com",
				Password:      "app-password",
				PersonalEmail: "me@example.com",
				Port:          "587",
				TLS:           "maybe",
				Username:      "sender@example.com",
			},
			wantErr: "parse EMAIL_TLS",
		},
		{
			name: "invalid recipient",
			raw: RawConfig{
				HostName:      "smtp.example.com",
				Password:      "app-password",
				PersonalEmail: "not an address",
				Port:          "587",
				TLS:           "true",
				Username:      "sender@example.com",
			},
			wantErr: "parse EMAIL_PERSONAL_EMAIL",
		},
		{
			name: "invalid username",
			raw: RawConfig{
				HostName:      "smtp.example.com",
				Password:      "app-password",
				PersonalEmail: "me@example.com",
				Port:          "587",
				TLS:           "true",
				Username:      "not an address",
			},
			wantErr: "parse EMAIL_USERNAME",
		},
		{
			name: "display names normalize to envelope addresses",
			raw: RawConfig{
				HostName:      "smtp.example.com",
				Password:      "app-password",
				PersonalEmail: "Me <me@example.com>",
				Port:          "587",
				TLS:           "true",
				Username:      "Sender <sender@example.com>",
			},
			want: Config{
				HostName:      "smtp.example.com",
				Password:      "app-password",
				PersonalEmail: "me@example.com",
				Port:          587,
				TLS:           true,
				Username:      "sender@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseConfig(tt.raw)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseConfig() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBuildMessage(t *testing.T) {
	t.Parallel()

	msg := buildMessage(Config{
		Username:      "sender@example.com",
		PersonalEmail: "me@example.com",
	}, "192.0.2.1", "192.0.2.2", time.Date(2026, 8, 11, 17, 30, 0, 0, time.UTC))

	checks := [][]byte{
		[]byte("From: sender@example.com\r\n"),
		[]byte("To: me@example.com\r\n"),
		[]byte("Subject: Watcher in the Water: public IP changed to 192.0.2.2\r\n"),
		[]byte("Previous IP: 192.0.2.1\r\n"),
		[]byte("Current IP: 192.0.2.2\r\n"),
		[]byte("Updated at: 2026-08-11T17:30:00Z\r\n"),
	}
	for _, check := range checks {
		if !bytes.Contains(msg, check) {
			t.Fatalf("message missing %q:\n%s", check, msg)
		}
	}
}
