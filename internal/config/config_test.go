package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	envToken       = "TELEGRAM_BOT_TOKEN"
	envOwner       = "TELEGRAM_OWNER_ID"
	envChannel     = "TELEGRAM_CHANNEL_ID"
	envS3Endpoint  = "MEDIA_S3_ENDPOINT"
	envS3Region    = "MEDIA_S3_REGION"
	envS3AccessKey = "MEDIA_S3_ACCESS_KEY_ID"
	envS3SecretKey = "MEDIA_S3_SECRET_ACCESS_KEY"
	envS3Bucket    = "MEDIA_S3_BUCKET"
	envS3PublicURL = "MEDIA_S3_PUBLIC_BASE_URL"
	envR2Account   = "R2_ACCOUNT_ID"
	envR2AccessKey = "R2_ACCESS_KEY_ID"
	envR2SecretKey = "R2_SECRET_ACCESS_KEY"
	envR2Bucket    = "R2_BUCKET"
	envR2PublicURL = "R2_PUBLIC_BASE_URL"
)

func TestLoad(t *testing.T) {
	t.Chdir(t.TempDir())

	tests := []struct {
		name        string
		env         map[string]string
		want        Config
		wantErr     bool
		wantErrText string
	}{
		{
			name: "loads basic configuration",
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
		{
			name: "loads Yandex Object Storage configuration",
			env: map[string]string{
				envToken:       "tok",
				envOwner:       "42",
				envChannel:     "@ch",
				envS3Endpoint:  "https://storage.yandexcloud.net/",
				envS3Region:    "ru-central1",
				envS3AccessKey: "access-key",
				envS3SecretKey: "secret-key",
				envS3Bucket:    "post-images",
				envS3PublicURL: "https://storage.yandexcloud.net/post-images/",
			},
			want: Config{
				BotToken:  "tok",
				OwnerID:   42,
				ChannelID: "@ch",
				Media: MediaConfig{
					Endpoint:      "https://storage.yandexcloud.net",
					Region:        "ru-central1",
					AccessKeyID:   "access-key",
					SecretKey:     "secret-key",
					Bucket:        "post-images",
					PublicBaseURL: "https://storage.yandexcloud.net/post-images",
				},
			},
		},
		{
			name: "loads legacy R2 configuration",
			env: map[string]string{
				envToken:       "tok",
				envOwner:       "42",
				envChannel:     "@ch",
				envR2Account:   "account",
				envR2AccessKey: "access-key",
				envR2SecretKey: "secret-key",
				envR2Bucket:    "post-images",
				envR2PublicURL: "https://media.example.com/",
			},
			want: Config{
				BotToken:  "tok",
				OwnerID:   42,
				ChannelID: "@ch",
				Media: MediaConfig{
					Endpoint:      "https://account.r2.cloudflarestorage.com",
					Region:        "auto",
					AccessKeyID:   "access-key",
					SecretKey:     "secret-key",
					Bucket:        "post-images",
					PublicBaseURL: "https://media.example.com",
				},
			},
		},
		{
			name: "partial S3 configuration fails",
			env: map[string]string{
				envToken:      "tok",
				envOwner:      "42",
				envChannel:    "@ch",
				envS3Endpoint: "https://storage.yandexcloud.net",
			},
			wantErr:     true,
			wantErrText: envS3Region,
		},
		{
			name: "partial R2 configuration fails",
			env: map[string]string{
				envToken:     "tok",
				envOwner:     "42",
				envChannel:   "@ch",
				envR2Account: "account",
			},
			wantErr:     true,
			wantErrText: envR2AccessKey,
		},
		{
			name: "S3 and R2 configuration cannot be mixed",
			env: map[string]string{
				envToken:       "tok",
				envOwner:       "42",
				envChannel:     "@ch",
				envS3Endpoint:  "https://storage.yandexcloud.net",
				envS3Region:    "ru-central1",
				envS3AccessKey: "access-key",
				envS3SecretKey: "secret-key",
				envS3Bucket:    "post-images",
				envS3PublicURL: "https://storage.yandexcloud.net/post-images",
				envR2Account:   "account",
				envR2AccessKey: "legacy-access-key",
				envR2SecretKey: "legacy-secret-key",
				envR2Bucket:    "legacy-post-images",
				envR2PublicURL: "https://media.example.com",
			},
			wantErr:     true,
			wantErrText: "MEDIA_S3_*",
		},
		{
			name: "R2 public URL must be HTTPS",
			env: map[string]string{
				envToken:       "tok",
				envOwner:       "42",
				envChannel:     "@ch",
				envR2Account:   "account",
				envR2AccessKey: "access-key",
				envR2SecretKey: "secret-key",
				envR2Bucket:    "post-images",
				envR2PublicURL: "http://media.example.com",
			},
			wantErr:     true,
			wantErrText: envR2PublicURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range allConfigEnvKeys() {
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

func allConfigEnvKeys() []string {
	return []string{envToken, envOwner, envChannel, envS3Endpoint, envS3Region, envS3AccessKey, envS3SecretKey, envS3Bucket, envS3PublicURL, envR2Account, envR2AccessKey, envR2SecretKey, envR2Bucket, envR2PublicURL}
}

func TestLoadReadsDotEnv(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	for _, key := range allConfigEnvKeys() {
		t.Setenv(key, "")
	}

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

	for _, key := range allConfigEnvKeys() {
		t.Setenv(key, "")
	}

	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("invalid line\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid .env line")
	}
}
