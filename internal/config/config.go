// Package config загружает конфигурацию приложения из переменных окружения и .env файла.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config содержит обязательные параметры для работы бота.
type Config struct {
	BotToken  string
	OwnerID   int64
	ChannelID string
	Media     MediaConfig
}

// MediaConfig contains S3-compatible object storage settings for Rich Markdown images.
type MediaConfig struct {
	Endpoint      string
	Region        string
	AccessKeyID   string
	SecretKey     string
	Bucket        string
	PublicBaseURL string
}

// Enabled reports whether Rich Markdown image uploads are configured.
func (c MediaConfig) Enabled() bool {
	return c.Endpoint != ""
}

// Load читает конфигурацию из .env файла и переменных окружения.
func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return Config{}, errors.New("TELEGRAM_BOT_TOKEN is required")
	}

	ownerRaw := strings.TrimSpace(os.Getenv("TELEGRAM_OWNER_ID"))
	if ownerRaw == "" {
		return Config{}, errors.New("TELEGRAM_OWNER_ID is required")
	}

	ownerID, err := strconv.ParseInt(ownerRaw, 10, 64)
	if err != nil {
		return Config{}, errors.New("TELEGRAM_OWNER_ID must be a valid integer")
	}

	channelID := strings.TrimSpace(os.Getenv("TELEGRAM_CHANNEL_ID"))
	if channelID == "" {
		return Config{}, errors.New("TELEGRAM_CHANNEL_ID is required")
	}

	media, err := loadMediaConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{
		BotToken:  token,
		OwnerID:   ownerID,
		ChannelID: channelID,
		Media:     media,
	}, nil
}

func loadMediaConfig() (MediaConfig, error) {
	media, s3Configured, err := loadS3MediaConfig()
	if err != nil {
		return MediaConfig{}, err
	}

	legacyMedia, r2Configured, err := loadLegacyR2MediaConfig()
	if err != nil {
		return MediaConfig{}, err
	}
	if s3Configured && r2Configured {
		return MediaConfig{}, errors.New("MEDIA_S3_* and R2_* configuration cannot be used together")
	}
	if s3Configured {
		return media, nil
	}
	if r2Configured {
		return legacyMedia, nil
	}
	return MediaConfig{}, nil
}

func loadS3MediaConfig() (MediaConfig, bool, error) {
	media := MediaConfig{
		Endpoint:      strings.TrimRight(strings.TrimSpace(os.Getenv("MEDIA_S3_ENDPOINT")), "/"),
		Region:        strings.TrimSpace(os.Getenv("MEDIA_S3_REGION")),
		AccessKeyID:   strings.TrimSpace(os.Getenv("MEDIA_S3_ACCESS_KEY_ID")),
		SecretKey:     strings.TrimSpace(os.Getenv("MEDIA_S3_SECRET_ACCESS_KEY")),
		Bucket:        strings.TrimSpace(os.Getenv("MEDIA_S3_BUCKET")),
		PublicBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("MEDIA_S3_PUBLIC_BASE_URL")), "/"),
	}
	values := []struct {
		name  string
		value string
	}{
		{name: "MEDIA_S3_ENDPOINT", value: media.Endpoint},
		{name: "MEDIA_S3_REGION", value: media.Region},
		{name: "MEDIA_S3_ACCESS_KEY_ID", value: media.AccessKeyID},
		{name: "MEDIA_S3_SECRET_ACCESS_KEY", value: media.SecretKey},
		{name: "MEDIA_S3_BUCKET", value: media.Bucket},
		{name: "MEDIA_S3_PUBLIC_BASE_URL", value: media.PublicBaseURL},
	}
	configured, err := validateMediaEnv(values)
	if err != nil || !configured {
		return MediaConfig{}, configured, err
	}
	if err := validateMediaURLs(media, "MEDIA_S3_ENDPOINT", "MEDIA_S3_PUBLIC_BASE_URL"); err != nil {
		return MediaConfig{}, true, err
	}
	return media, true, nil
}

func loadLegacyR2MediaConfig() (MediaConfig, bool, error) {
	r2 := struct {
		AccountID     string
		AccessKeyID   string
		SecretKey     string
		Bucket        string
		PublicBaseURL string
	}{
		AccountID:     strings.TrimSpace(os.Getenv("R2_ACCOUNT_ID")),
		AccessKeyID:   strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID")),
		SecretKey:     strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY")),
		Bucket:        strings.TrimSpace(os.Getenv("R2_BUCKET")),
		PublicBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("R2_PUBLIC_BASE_URL")), "/"),
	}

	values := []struct {
		name  string
		value string
	}{
		{name: "R2_ACCOUNT_ID", value: r2.AccountID},
		{name: "R2_ACCESS_KEY_ID", value: r2.AccessKeyID},
		{name: "R2_SECRET_ACCESS_KEY", value: r2.SecretKey},
		{name: "R2_BUCKET", value: r2.Bucket},
		{name: "R2_PUBLIC_BASE_URL", value: r2.PublicBaseURL},
	}

	configured, err := validateMediaEnv(values)
	if err != nil || !configured {
		return MediaConfig{}, configured, err
	}
	media := MediaConfig{
		Endpoint:      "https://" + r2.AccountID + ".r2.cloudflarestorage.com",
		Region:        "auto",
		AccessKeyID:   r2.AccessKeyID,
		SecretKey:     r2.SecretKey,
		Bucket:        r2.Bucket,
		PublicBaseURL: r2.PublicBaseURL,
	}
	if err := validateMediaURLs(media, "R2_ACCOUNT_ID", "R2_PUBLIC_BASE_URL"); err != nil {
		return MediaConfig{}, true, err
	}
	return media, true, nil
}

func validateMediaEnv(values []struct {
	name  string
	value string
}) (bool, error) {
	configured := false
	for _, item := range values {
		if item.value != "" {
			configured = true
			break
		}
	}
	if !configured {
		return false, nil
	}
	for _, item := range values {
		if item.value == "" {
			return true, fmt.Errorf("%s is required when image storage is configured", item.name)
		}
	}
	return true, nil
}

func validateMediaURLs(media MediaConfig, endpointName, publicURLName string) error {
	endpoint, err := url.Parse(media.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Path != "" && endpoint.Path != "/") {
		return fmt.Errorf("%s must be an HTTPS URL without credentials, path, query, or fragment", endpointName)
	}
	publicURL, err := url.Parse(media.PublicBaseURL)
	if err != nil || publicURL.Scheme != "https" || publicURL.Host == "" || publicURL.User != nil || publicURL.RawQuery != "" || publicURL.Fragment != "" {
		return fmt.Errorf("%s must be an HTTPS URL without credentials, query, or fragment", publicURLName)
	}
	return nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path) //nolint:gosec // .env path is hardcoded; Load is not exposed to user input.
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid .env line %d: expected KEY=VALUE", lineNumber)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			return fmt.Errorf("invalid .env line %d: empty key", lineNumber)
		}
		if _, exists := os.LookupEnv(key); !exists || os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("set %s from .env: %w", key, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	return nil
}
