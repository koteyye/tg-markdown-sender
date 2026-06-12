package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/koteyye/tg-markdown-sender/internal/bot"
	"github.com/koteyye/tg-markdown-sender/internal/config"
	"github.com/koteyye/tg-markdown-sender/internal/drafts"
	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	runCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	httpClient := &http.Client{Timeout: 65 * time.Second}
	tg := telegram.NewClient(cfg.BotToken, httpClient, logger)
	app := bot.New(cfg, tg, drafts.NewMemoryStore(), logger)

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 15*time.Second)
	botInfo, err := tg.GetMe(startupCtx)
	cancelStartup()
	if err != nil {
		logger.Error("telegram api startup check failed", "method", "getMe", "error", err)
		os.Exit(1)
	}
	logger.Info("telegram api startup check passed", "bot_id", botInfo.ID, "bot_username", botInfo.Username)

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run(runCtx)
	}()

	logger.Info("bot started")

	select {
	case err := <-errCh:
		if errors.Is(err, context.Canceled) && runCtx.Err() != nil {
			logger.Info("shutdown signal received")
		} else if err != nil {
			logger.Error("bot stopped with error", "error", err)
			os.Exit(1)
		}
	case <-runCtx.Done():
		logger.Info("shutdown signal received")
		stopSignals()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("bot stopped with error during shutdown", "error", err)
				os.Exit(1)
			}
		case <-shutdownCtx.Done():
			logger.Error("graceful shutdown timed out", "timeout", "10s")
			os.Exit(1)
		}
	}

	logger.Info("bot stopped")
}
