package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
				"TELEGRAM_BOT_TOKEN":  "test-token",
				"TELEGRAM_OWNER_ID":   "42",
				"TELEGRAM_CHANNEL_ID": "@test_channel",
			},
			want: Config{
				BotToken:  "test-token",
				OwnerID:   42,
				ChannelID: "@test_channel",
			},
		},
		{
			name:        "missing token",
			env:         map[string]string{"TELEGRAM_OWNER_ID": "42", "TELEGRAM_CHANNEL_ID": "@ch"},
			wantErr:     true,
			wantErrText: "TELEGRAM_BOT_TOKEN",
		},
		{
			name:        "missing owner",
			env:         map[string]string{"TELEGRAM_BOT_TOKEN": "tok", "TELEGRAM_CHANNEL_ID": "@ch"},
			wantErr:     true,
			wantErrText: "TELEGRAM_OWNER_ID",
		},
		{
			name:        "invalid owner",
			env:         map[string]string{"TELEGRAM_BOT_TOKEN": "tok", "TELEGRAM_OWNER_ID": "abc", "TELEGRAM_CHANNEL_ID": "@ch"},
			wantErr:     true,
			wantErrText: "TELEGRAM_OWNER_ID",
		},
		{
			name:        "missing channel",
			env:         map[string]string{"TELEGRAM_BOT_TOKEN": "tok", "TELEGRAM_OWNER_ID": "42"},
			wantErr:     true,
			wantErrText: "TELEGRAM_CHANNEL_ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := []string{"TELEGRAM_BOT_TOKEN", "TELEGRAM_OWNER_ID", "TELEGRAM_CHANNEL_ID"}
			for _, key := range keys {
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
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}

	_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
	_ = os.Unsetenv("TELEGRAM_OWNER_ID")
	_ = os.Unsetenv("TELEGRAM_CHANNEL_ID")

	env := "TELEGRAM_BOT_TOKEN=test-token\nTELEGRAM_OWNER_ID=42\nTELEGRAM_CHANNEL_ID=@test_channel\n"
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
	t.Parallel()

	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}

	_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
	_ = os.Unsetenv("TELEGRAM_OWNER_ID")
	_ = os.Unsetenv("TELEGRAM_CHANNEL_ID")

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("invalid line\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for invalid .env line")
	}
}
