# Telegram Rich Markdown Publisher

A single-binary Go Telegram bot that publishes rich Markdown posts and photo captions to a channel using Telegram Bot API. It runs via long polling, does not expose incoming ports, and deploys to a VPS as a static Linux binary managed by `systemd`.

## Features

- Accepts messages only from a configured owner (`TELEGRAM_OWNER_ID`).
- Renders a preview of the final post before publishing.
- Publishes rich Markdown posts from Markdown protected in `md` code blocks.
- Schedules a preview for the nearest selected Moscow-time slot.
- Publishes photos with captions as a single message via `sendPhoto` when image storage is not configured.
- Converts a Telegram photo with a Markdown caption into a Rich Markdown media block when S3-compatible image storage is configured.
- Explains image publishing with `/infoimage` and checks authenticated bucket access with `/checkstorage`.
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

## Sending Markdown

Telegram clients can render Markdown while composing a message and remove its source markers before the bot receives it. Send every Rich Markdown post inside one fenced code block with the language `md`; the bot uses only the block contents.

````md
```md
**Знакомьтесь: Рефералодав**

![](https://example.com/image.jpg)

Обычный текст с `кодом` и [гиперссылкой](https://example.com).
```
````

## Scheduled Publishing

The preview has an `Отправить потом` button. Choose one of the Moscow-time slots: `09:00`, `12:00`, `15:00`, `18:00`, or `21:00`. If the selected time has already passed, the post is scheduled for that time on the next day.

Drafts and scheduled posts are stored in memory, so restart the bot only after scheduled posts have been published.

## Images in Rich Markdown

Use `/infoimage` in the bot for these instructions and `/checkstorage` to verify the endpoint, service account credentials, and bucket permissions without uploading an object.

Image storage is optional. Without it, a photo sent to the bot is published through `sendPhoto` with its Telegram caption.

With S3-compatible storage configured, send the bot a photo with a caption inside an `md` code block. The bot downloads the photo from Telegram, uploads it to the configured bucket, and creates one Rich Markdown post using the public image URL. Put `{{image}}` on a separate line to choose where the image appears; without the placeholder, the image is appended after the caption.

```md
# Post title

Introductory text.

{{image}}

Text below the image.
```

Telegram limits captions to 1,024 characters. For a longer post or several images, first send each image to the bot without a caption. It will reply with the Rich Markdown image block, which you can place anywhere in the final Markdown message:

```md
![](https://media.example.com/images/your-image.jpg)
```

Set up Yandex Object Storage:

1. Create a Standard bucket, for example `telegram-post-images`. Use a name without dots so that its default HTTPS URL works without a custom certificate.
2. In the bucket settings, enable public access only for **reading objects**. Do not enable listing objects or reading bucket settings.
3. Create a service account, grant it `storage.editor` for this bucket, and create a static access key. Keep the secret key only in the server configuration.
4. Add the credentials to `.env`, or to `/etc/telegram-publisher/env` on the VPS:

```env
MEDIA_S3_ENDPOINT=https://storage.yandexcloud.net
MEDIA_S3_REGION=ru-central1
MEDIA_S3_ACCESS_KEY_ID=
MEDIA_S3_SECRET_ACCESS_KEY=
MEDIA_S3_BUCKET=telegram-post-images
MEDIA_S3_PUBLIC_BASE_URL=https://storage.yandexcloud.net/telegram-post-images
```

The generated object names are random and the links are permanent. Public read access is required so Telegram can fetch images when rendering the Markdown post. Do not use pre-signed URLs because they expire.

The previous `R2_*` variables remain supported for Cloudflare R2. Do not set them together with `MEDIA_S3_*`.

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
internal/objectstore   # S3-compatible image storage
internal/telegram      # Telegram Bot API client
scripts/               # Build, bootstrap, and deployment helpers
```

## Development

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
```
