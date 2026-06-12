package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsDotEnv(t *testing.T) {
	oldToken, hadToken := os.LookupEnv("TELEGRAM_BOT_TOKEN")
	oldOwner, hadOwner := os.LookupEnv("TELEGRAM_OWNER_ID")
	oldChannel, hadChannel := os.LookupEnv("TELEGRAM_CHANNEL_ID")
	defer restoreEnv("TELEGRAM_BOT_TOKEN", oldToken, hadToken)
	defer restoreEnv("TELEGRAM_OWNER_ID", oldOwner, hadOwner)
	defer restoreEnv("TELEGRAM_CHANNEL_ID", oldChannel, hadChannel)
	_ = os.Unsetenv("TELEGRAM_BOT_TOKEN")
	_ = os.Unsetenv("TELEGRAM_OWNER_ID")
	_ = os.Unsetenv("TELEGRAM_CHANNEL_ID")

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

func restoreEnv(key, value string, existed bool) {
	if existed {
		_ = os.Setenv(key, value)
		return
	}
	_ = os.Unsetenv(key)
}
