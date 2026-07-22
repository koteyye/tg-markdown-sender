package config

import (
	"os"
	"strings"
	"testing"
)

const (
	envToken      = "TELEGRAM_BOT_TOKEN"
	envOwner      = "TELEGRAM_OWNER_ID"
	envChannel    = "TELEGRAM_CHANNEL_ID"
	envS3Endpoint = "MEDIA_S3_ENDPOINT"
	envR2Account  = "R2_ACCOUNT_ID"

	testBotToken  = "tok"
	testOwnerID   = "42"
	testChannelID = "@ch"
)

//nolint:paralleltest // The test changes process-wide environment variables and working directory.
func TestLoadRequiredConfiguration(t *testing.T) {
	setupConfigTest(t)

	tests := []struct {
		name        string
		env         map[string]string
		wantErrText string
	}{
		{
			name:        "missing token",
			env:         map[string]string{envOwner: testOwnerID, envChannel: testChannelID},
			wantErrText: envToken,
		},
		{
			name:        "missing owner",
			env:         map[string]string{envToken: testBotToken, envChannel: testChannelID},
			wantErrText: envOwner,
		},
		{
			name:        "invalid owner",
			env:         map[string]string{envToken: testBotToken, envOwner: "abc", envChannel: testChannelID},
			wantErrText: envOwner,
		},
		{
			name:        "missing channel",
			env:         map[string]string{envToken: testBotToken, envOwner: testOwnerID},
			wantErrText: envChannel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadWithEnv(t, tt.env)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrText) {
				t.Fatalf("Load() error = %v, want error containing %q", err, tt.wantErrText)
			}
		})
	}
}

//nolint:paralleltest // The test changes process-wide environment variables and working directory.
func TestLoadBasicConfiguration(t *testing.T) {
	setupConfigTest(t)

	cfg, err := loadWithEnv(t, baseEnv())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	want := Config{BotToken: testBotToken, OwnerID: 42, ChannelID: testChannelID}
	if cfg != want {
		t.Fatalf("unexpected config: %+v, want %+v", cfg, want)
	}
}

//nolint:paralleltest // The test changes process-wide environment variables and working directory.
func TestLoadIgnoresLegacyS3AndR2Variables(t *testing.T) {
	setupConfigTest(t)

	env := baseEnv()
	env[envS3Endpoint] = "https://storage.yandexcloud.net"
	env[envR2Account] = "account"

	cfg, err := loadWithEnv(t, env)
	if err != nil {
		t.Fatalf("legacy variables must not cause a load error: %v", err)
	}
	want := Config{BotToken: testBotToken, OwnerID: 42, ChannelID: testChannelID}
	if cfg != want {
		t.Fatalf("legacy variables must be ignored: got %+v, want %+v", cfg, want)
	}
}

//nolint:paralleltest // The test changes process-wide environment variables and working directory.
func TestLoadReadsDotEnv(t *testing.T) {
	setupConfigTest(t)

	env := envToken + "=test-token\n" + envOwner + "=" + testOwnerID + "\n" + envChannel + "=@test_channel\n"
	if err := os.WriteFile(".env", []byte(env), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.BotToken != "test-token" || cfg.OwnerID != 42 || cfg.ChannelID != "@test_channel" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

//nolint:paralleltest // The test changes process-wide environment variables and working directory.
func TestLoadDotEnvInvalidLine(t *testing.T) {
	setupConfigTest(t)

	if err := os.WriteFile(".env", []byte("invalid line\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid .env line")
	}
}

func setupConfigTest(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, key := range allConfigEnvKeys() {
		t.Setenv(key, "")
	}
}

func loadWithEnv(t *testing.T, env map[string]string) (Config, error) {
	t.Helper()
	for key, value := range env {
		t.Setenv(key, value)
	}
	return Load()
}

func baseEnv() map[string]string {
	return map[string]string{
		envToken:   testBotToken,
		envOwner:   testOwnerID,
		envChannel: testChannelID,
	}
}

func allConfigEnvKeys() []string {
	return []string{
		envToken,
		envOwner,
		envChannel,
		envS3Endpoint,
		envR2Account,
	}
}
