# Telegram Rich Markdown Publisher

A single-binary Go Telegram bot that publishes rich Markdown posts and photos to a channel using the native **Rich Messages** of Telegram Bot API 10.2+. It runs via long polling, exposes no incoming ports, and deploys to a VPS as a static Linux binary managed by `systemd`.

## Features

- Accepts messages only from a configured owner (`TELEGRAM_OWNER_ID`).
- Accepts posts created in Telegram's built-in Rich Markdown editor (`message.rich_message`) as the preferred way to create a post, preserving headings, lists, quotes, tables, code blocks, spoilers, custom emoji, and embedded media.
- Publishes the same content (structure, formatting, media, custom emoji) to the configured channel.
- Reuses media through Telegram `file_id` — no external storage and no re-upload.
- Keeps the classic Markdown source protected inside an `md` fenced code block.
- Renders a preview before publishing and offers immediate publish, scheduled publish, or cancel.
- Schedules a preview for the nearest selected Moscow-time slot.
- Accepts a photo with a Markdown caption (with optional `{{image}}` placeholder) and builds one Rich Message using the Telegram `file_id`.
- Accepts a captionless photo and returns a short alias reference (`![](tg://photo?id=...)`) to embed in a later Markdown post.
- Supports premium/custom emoji; reports the channel restriction (Fragment username) explicitly without silently dropping the emoji.
- Targets the channel configured in `TELEGRAM_CHANNEL_ID`.
- Prevents a draft from being published twice.
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

Only `TELEGRAM_BOT_TOKEN`, `TELEGRAM_OWNER_ID`, `TELEGRAM_CHANNEL_ID`, and `LOG_LEVEL` are required. External S3-compatible storage (Yandex Object Storage / Cloudflare R2) is **no longer needed** — media is reused through the Telegram `file_id`.

## Native Rich Messages (recommended)

Create a post directly in Telegram's built-in Rich Markdown editor, then send it to the bot. The bot reads the structured `message.rich_message`, preserves paragraphs, headings, bold/italic/underline/strikethrough, links, mentions, code and code blocks, quotes, lists, tables, dividers, expandable blocks, formulas, **custom emoji** (with their `custom_emoji_id` and alternative text), nested blocks, and embedded photo/video/animation/audio/voice-note media.

The preview and the channel publication use the identical converted `InputRichMessage`, so what you preview is what gets published. Images and other media reuse the Telegram `file_id` — the bot never downloads or re-uploads them.

If the bot receives a block type it does not yet support, it returns a clear message naming the unsupported type instead of panicking or silently dropping it.

## Sending Markdown via `md` block

Telegram clients render Markdown while composing and may strip its source markers before the bot receives it. Send a Rich Markdown post inside one fenced code block with the language `md`; the bot uses only the block contents.

````md
```md
**Знакомьтесь: Рефералодав**

Обычный текст с `кодом` и [гиперссылкой](https://example.com).
```
````

### Custom emoji in Markdown

Use the Rich Markdown custom-emoji syntax directly:

```md
![🔥](tg://emoji?id=5368324170671202286) огонь!
```

The bot does **not** convert or invent custom-emoji IDs; it passes your `tg://emoji?id=...` through to Telegram as-is.

## Images via Telegram `file_id`

There are three ways to include images, all using the Telegram `file_id` and `tg://photo?id=<alias>` links:

1. **Photo without caption.** Send the bot a photo with no caption. It stores the `file_id` under a short alias (for example `photo_ab12cd34`) and replies with the line to embed:

   ```md
   ![](tg://photo?id=photo_ab12cd34)
   ```

   Place that line anywhere in a later Markdown post. `tg://video?id=...` and `tg://audio?id=...` references are also recognized.

2. **Photo with a Markdown caption.** Send a photo whose caption is a single `md` fenced code block. Put `{{image}}` on its own line to choose where the image appears; without the placeholder, the image is appended after the caption. The bot builds one Rich Message using the photo's `file_id` under the alias `cover` — no download, no external URL.

   ````md
   ```md
   # Post title

   Introductory text.

   {{image}}

   Text below the image.
   ```
   ````

3. **Native Rich Message.** Send a post from the built-in editor that already contains photo/video/animation/audio/voice-note blocks (including inside collage, slideshow, list, quote, or details). The bot preserves them all.

### Notes on media aliases

Aliases (`photo_ab12cd34`, `cover`, etc.) are short local identifiers (1–64 characters of `A-Z a-z 0-9 _ -`), **not** Telegram `file_id`s. They are stored in memory only. If an alias is lost after a restart, the bot refuses to send an invalid request and asks you to send the image again.

## Custom emoji in channels

Telegram allows custom emoji in private chats, groups, and supergroups when the bot owner has Telegram Premium. **For channel posts, the bot must have purchased an additional username on Fragment.** If publication to the channel is rejected because of a custom-emoji restriction, the bot:

- does not mark the draft as published, so you can retry;
- shows a clear message explaining the likely Fragment-username requirement;
- records the original Telegram error in the structured log.

Premium/custom emoji are never silently dropped or replaced with plain emoji.

## Scheduled Publishing

The preview has an `Отправить потом` button. Choose one of the Moscow-time slots: `09:00`, `12:00`, `15:00`, `18:00`, or `21:00`. If the selected time has already passed, the post is scheduled for that time on the next day.

Immediate and scheduled publication send the identical `InputRichMessage` content and media. A draft that was already published is never published again.

## In-memory state

Drafts, media aliases, and scheduled posts are stored **in memory** and are lost when the bot restarts. Restart the bot only after scheduled posts have been published, and re-send any media whose alias was lost.

## Rich Message limits

The bot validates the following limits before sending to Telegram:

- up to 32 768 UTF-8 characters of rich text;
- up to 500 blocks including nested blocks;
- up to 16 nesting levels;
- up to 50 media attachments;
- up to 20 table columns;
- media alias length 1–64 characters with the allowed character set.

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
internal/rich          # Native Rich Message model, conversion, aliases, validation
internal/telegram      # Telegram Bot API client
scripts/               # Build, bootstrap, and deployment helpers
```

## Development

```bash
gofmt -w cmd internal
go mod tidy
go test ./...
go test -race ./...
go vet ./...
golangci-lint run
```
