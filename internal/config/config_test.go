package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	envToken   = "TELEGRAM_BOT_TOKEN"
	envOwner   = "TELEGRAM_OWNER_ID"
	envChannel = "TELEGRAM_CHANNEL_ID"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		want        Config
		wantErr     bool
		wantErrText string
	}{
		{
			name: "loads from .env",
			env: map[string]string{
				envToken:   "test-token",
				envOwner:   "42",
				envChannel: "@test_channel",
			},
			want: Config{
				BotToken:  "test-token",
				OwnerID:   42,
				ChannelID: "@test_channel",
			},
		},
		{
			name:        "missing token",
			env:         map[string]string{envOwner: "42", envChannel: "@ch"},
			wantErr:     true,
			wantErrText: envToken,
		},
		{
			name:        "missing owner",
			env:         map[string]string{envToken: "tok", envChannel: "@ch"},
			wantErr:     true,
			wantErrText: envOwner,
		},
		{
			name:        "invalid owner",
			env:         map[string]string{envToken: "tok", envOwner: "abc", envChannel: "@ch"},
			wantErr:     true,
			wantErrText: envOwner,
		},
		{
			name:        "missing channel",
			env:         map[string]string{envToken: "tok", envOwner: "42"},
			wantErr:     true,
			wantErrText: envChannel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range []string{envToken, envOwner, envChannel} {
				t.Setenv(key, "")
			}
			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.wantErrText) {
					t.Fatalf("error must mention %q: %v", tt.wantErrText, err)
				}
				return
			}
			if cfg != tt.want {
				t.Fatalf("unexpected config: %+v, want %+v", cfg, tt.want)
			}
		})
	}
}

func TestLoadReadsDotEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	t.Setenv(envToken, "")
	t.Setenv(envOwner, "")
	t.Setenv(envChannel, "")

	env := envToken + "=test-token\n" + envOwner + "=42\n" + envChannel + "=@test_channel\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.BotToken != "test-token" {
		t.Fatalf("unexpected token: %q", cfg.BotToken)
	}
	if cfg.OwnerID != 42 {
		t.Fatalf("unexpected owner id: %d", cfg.OwnerID)
	}
	if cfg.ChannelID != "@test_channel" {
		t.Fatalf("unexpected channel id: %q", cfg.ChannelID)
	}
}

func TestLoadDotEnvInvalidLine(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	t.Setenv(envToken, "")
	t.Setenv(envOwner, "")
	t.Setenv(envChannel, "")

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("invalid line\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid .env line")
	}
}
