// Package config загружает конфигурацию приложения из переменных окружения и .env файла.
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config содержит обязательные параметры для работы бота.
type Config struct {
	BotToken  string
	OwnerID   int64
	ChannelID string
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

	return Config{
		BotToken:  token,
		OwnerID:   ownerID,
		ChannelID: channelID,
	}, nil
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
