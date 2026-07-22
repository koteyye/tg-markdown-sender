// Package main запускает Telegram-бота для публикации Rich Markdown постов.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/koteyye/tg-markdown-sender/internal/bot"
	"github.com/koteyye/tg-markdown-sender/internal/config"
	"github.com/koteyye/tg-markdown-sender/internal/drafts"
	"github.com/koteyye/tg-markdown-sender/internal/telegram"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel()}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		return
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
		return
	}
	logger.Info("telegram api startup check passed", "bot_id", botInfo.ID, "bot_username", botInfo.Username)

	startPprof(logger)

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run(runCtx)
	}()

	logger.Info("bot started")

	exitCode := 0
	select {
	case err := <-errCh:
		if errors.Is(err, context.Canceled) && runCtx.Err() != nil {
			logger.Info("shutdown signal received")
		} else if err != nil {
			logger.Error("bot stopped with error", "error", err)
			exitCode = 1
		}
	case <-runCtx.Done():
		logger.Info("shutdown signal received")
		stopSignals()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		select {
		case err := <-errCh:
			cancel()
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("bot stopped with error during shutdown", "error", err)
				exitCode = 1
			}
		case <-shutdownCtx.Done():
			cancel()
			logger.Error("graceful shutdown timed out", "timeout", "10s")
			exitCode = 1
		}
	}

	logger.Info("bot stopped")
	//nolint:gocritic // stopSignals is a sync.OnceFunc; single os.Exit is intentional at program end.
	os.Exit(exitCode)
}

func logLevel() slog.Level {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	return level
}

func startPprof(logger *slog.Logger) {
	addr := strings.TrimSpace(os.Getenv("PPROF_ADDR"))
	if addr == "" {
		return
	}

	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      http.DefaultServeMux,
	}

	go func() {
		logger.Info("pprof server started", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("pprof server failed", "error", err)
		}
	}()
}
