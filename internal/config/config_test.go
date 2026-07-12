package config

import (
	"os"
	"strings"
	"testing"
)

const (
	envToken         = "TELEGRAM_BOT_TOKEN"
	envOwner         = "TELEGRAM_OWNER_ID"
	envChannel       = "TELEGRAM_CHANNEL_ID"
	envS3Endpoint    = "MEDIA_S3_ENDPOINT"
	envS3Region      = "MEDIA_S3_REGION"
	envS3Access      = "MEDIA_S3_ACCESS_KEY_ID"
	envS3KeyMaterial = "MEDIA_S3_SECRET_ACCESS_KEY"
	envS3Bucket      = "MEDIA_S3_BUCKET"
	envS3PublicURL   = "MEDIA_S3_PUBLIC_BASE_URL"
	envR2Account     = "R2_ACCOUNT_ID"
	envR2Access      = "R2_ACCESS_KEY_ID"
	envR2KeyMaterial = "R2_SECRET_ACCESS_KEY"
	envR2Bucket      = "R2_BUCKET"
	envR2PublicURL   = "R2_PUBLIC_BASE_URL"

	testBotToken    = "tok"
	testOwnerID     = "42"
	testChannelID   = "@ch"
	testAccessKey   = "access-key"
	testKeyMaterial = "secret-key"
	testBucketName  = "post-images"
	testAccountID   = "account"
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
func TestLoadYandexObjectStorageConfiguration(t *testing.T) {
	setupConfigTest(t)

	env := baseEnv()
	env[envS3Endpoint] = "https://storage.yandexcloud.net/"
	env[envS3Region] = "ru-central1"
	env[envS3Access] = testAccessKey
	env[envS3KeyMaterial] = testKeyMaterial
	env[envS3Bucket] = testBucketName
	env[envS3PublicURL] = "https://storage.yandexcloud.net/post-images/"

	cfg, err := loadWithEnv(t, env)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	want := Config{
		BotToken:  testBotToken,
		OwnerID:   42,
		ChannelID: testChannelID,
		Media: MediaConfig{
			Endpoint:      "https://storage.yandexcloud.net",
			Region:        "ru-central1",
			AccessKeyID:   testAccessKey,
			SecretKey:     testKeyMaterial,
			Bucket:        testBucketName,
			PublicBaseURL: "https://storage.yandexcloud.net/post-images",
		},
	}
	if cfg != want {
		t.Fatalf("unexpected config: %+v, want %+v", cfg, want)
	}
}

//nolint:paralleltest // The test changes process-wide environment variables and working directory.
func TestLoadLegacyR2Configuration(t *testing.T) {
	setupConfigTest(t)

	env := baseEnv()
	env[envR2Account] = testAccountID
	env[envR2Access] = testAccessKey
	env[envR2KeyMaterial] = testKeyMaterial
	env[envR2Bucket] = testBucketName
	env[envR2PublicURL] = "https://media.example.com/"

	cfg, err := loadWithEnv(t, env)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	want := Config{
		BotToken:  testBotToken,
		OwnerID:   42,
		ChannelID: testChannelID,
		Media: MediaConfig{
			Endpoint:      "https://account.r2.cloudflarestorage.com",
			Region:        "auto",
			AccessKeyID:   testAccessKey,
			SecretKey:     testKeyMaterial,
			Bucket:        testBucketName,
			PublicBaseURL: "https://media.example.com",
		},
	}
	if cfg != want {
		t.Fatalf("unexpected config: %+v, want %+v", cfg, want)
	}
}

//nolint:paralleltest // The test changes process-wide environment variables and working directory.
func TestLoadRejectsPartialS3Configuration(t *testing.T) {
	setupConfigTest(t)

	env := baseEnv()
	env[envS3Endpoint] = "https://storage.yandexcloud.net"
	requireLoadError(t, env, envS3Region)
}

//nolint:paralleltest // The test changes process-wide environment variables and working directory.
func TestLoadRejectsPartialR2Configuration(t *testing.T) {
	setupConfigTest(t)

	env := baseEnv()
	env[envR2Account] = testAccountID
	requireLoadError(t, env, envR2Access)
}

//nolint:paralleltest // The test changes process-wide environment variables and working directory.
func TestLoadRejectsMixedS3AndR2Configuration(t *testing.T) {
	setupConfigTest(t)

	env := baseEnv()
	addS3Config(env)
	addR2Config(env)
	requireLoadError(t, env, "MEDIA_S3_*")
}

//nolint:paralleltest // The test changes process-wide environment variables and working directory.
func TestLoadRejectsInsecureR2PublicURL(t *testing.T) {
	setupConfigTest(t)

	env := baseEnv()
	addR2Config(env)
	env[envR2PublicURL] = "http://media.example.com"
	requireLoadError(t, env, envR2PublicURL)
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

func requireLoadError(t *testing.T, env map[string]string, want string) {
	t.Helper()
	_, err := loadWithEnv(t, env)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Load() error = %v, want error containing %q", err, want)
	}
}

func baseEnv() map[string]string {
	return map[string]string{
		envToken:   testBotToken,
		envOwner:   testOwnerID,
		envChannel: testChannelID,
	}
}

func addS3Config(env map[string]string) {
	env[envS3Endpoint] = "https://storage.yandexcloud.net"
	env[envS3Region] = "ru-central1"
	env[envS3Access] = testAccessKey
	env[envS3KeyMaterial] = testKeyMaterial
	env[envS3Bucket] = testBucketName
	env[envS3PublicURL] = "https://storage.yandexcloud.net/post-images"
}

func addR2Config(env map[string]string) {
	env[envR2Account] = testAccountID
	env[envR2Access] = "legacy-access-key"
	env[envR2KeyMaterial] = "legacy-key-material"
	env[envR2Bucket] = "legacy-post-images"
	env[envR2PublicURL] = "https://media.example.com"
}

func allConfigEnvKeys() []string {
	return []string{
		envToken,
		envOwner,
		envChannel,
		envS3Endpoint,
		envS3Region,
		envS3Access,
		envS3KeyMaterial,
		envS3Bucket,
		envS3PublicURL,
		envR2Account,
		envR2Access,
		envR2KeyMaterial,
		envR2Bucket,
		envR2PublicURL,
	}
}
