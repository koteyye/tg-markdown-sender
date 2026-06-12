package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken  string
	OwnerID   int64
	ChannelID string
}

func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	ownerRaw := strings.TrimSpace(os.Getenv("TELEGRAM_OWNER_ID"))
	if ownerRaw == "" {
		return Config{}, fmt.Errorf("TELEGRAM_OWNER_ID is required")
	}

	ownerID, err := strconv.ParseInt(ownerRaw, 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("TELEGRAM_OWNER_ID must be a valid integer")
	}

	channelID := strings.TrimSpace(os.Getenv("TELEGRAM_CHANNEL_ID"))
	if channelID == "" {
		return Config{}, fmt.Errorf("TELEGRAM_CHANNEL_ID is required")
	}

	return Config{
		BotToken:  token,
		OwnerID:   ownerID,
		ChannelID: channelID,
	}, nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

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
		if _, exists := os.LookupEnv(key); !exists {
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
