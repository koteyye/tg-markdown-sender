# Telegram Rich Markdown Publisher

A single-binary Go Telegram bot that publishes rich Markdown posts and photo captions to a channel using Telegram Bot API. It runs via long polling, does not expose incoming ports, and deploys to a VPS as a static Linux binary managed by `systemd`.

## Features

- Accepts messages only from a configured owner (`TELEGRAM_OWNER_ID`).
- Renders a preview of the final post before publishing.
- Publishes rich Markdown posts with entities restored from `sendRichMessage`.
- Publishes photos with captions as a single message via `sendPhoto`.
- Targets the channel configured in `TELEGRAM_CHANNEL_ID`.
- Prevents duplicate drafts from being published twice.
- Verifies the Telegram API with `getMe` before starting polling.
- Writes structured JSON logs via `slog`.
- Gracefully shuts down on `SIGINT` and `SIGTERM`.
- Optional pprof server for runtime diagnostics via `PPROF_ADDR`.

## Quick Start

Copy the example configuration:

```bash
cp .env.example .env
```

Edit `.env`:

```env
TELEGRAM_BOT_TOKEN=
TELEGRAM_OWNER_ID=
TELEGRAM_CHANNEL_ID=
LOG_LEVEL=info
```

Run locally:

```bash
go run ./cmd/bot
```

## Build and Deploy

Build a static Linux binary:

```bash
GOARCH=amd64 bash scripts/build-linux.sh
```

The workflow in `.github/workflows/deploy.yml` runs on every push to `master`:

```text
gofmt / tests / vet / lint / vulncheck
→ build static Linux binary
→ upload binary and checksum to the VPS via SSH
→ verify checksum
→ atomically replace binary and restart systemd service
→ rollback on failure
```

## Project Structure

```text
cmd/bot                # Application entry point
internal/bot           # Message and callback handlers
internal/config        # Environment and .env configuration
internal/drafts        # In-memory draft store
internal/telegram      # Telegram Bot API client
scripts/               # Build, bootstrap, and deployment helpers
```

## Development

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
```
