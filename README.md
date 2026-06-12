# Telegram Rich Markdown Publisher

Go Telegram-бот для публикации Rich Markdown в канал. Бот работает через Long Polling, не открывает входящие порты и разворачивается на VDS как один статический Linux-бинарник под `systemd`.

## Возможности

- принимает сообщения только от `TELEGRAM_OWNER_ID`;
- показывает предпросмотр через Telegram Bot API `sendRichMessage`;
- публикует фото с подписью одним сообщением через `sendPhoto`;
- публикует в канал из `TELEGRAM_CHANNEL_ID`;
- защищает черновик от повторной публикации;
- проверяет Telegram API через `getMe` до запуска polling;
- пишет JSON-логи через `slog`;
- штатно завершается по `SIGTERM` и `SIGINT`.

## Локальный Запуск

Скопируйте пример конфигурации:

```bash
cp .env.example .env
```

Заполните:

```env
TELEGRAM_BOT_TOKEN=
TELEGRAM_OWNER_ID=
TELEGRAM_CHANNEL_ID=
LOG_LEVEL=info
```

Запустите:

```bash
go run ./cmd/bot
```

Для проверки без канала можно временно указать в `TELEGRAM_CHANNEL_ID` свой Telegram user ID. Тогда публикация отправится в личный чат с ботом.

Бот поддерживает два типа черновиков:

- текстовый Markdown-пост;
- фото с подписью.

Фото публикуется одним Telegram-сообщением через `sendPhoto`: изображение сверху, подпись снизу. Внешнее HTTPS-хранилище для этого режима не требуется, используется Telegram `file_id`.

## Локальная Сборка

Скрипт сборки требует явный `GOARCH`.

```bash
GOARCH=amd64 bash scripts/build-linux.sh
```

Результат:

```text
dist/telegram-publisher
dist/telegram-publisher.sha256
```

Скрипт выполняет:

```bash
go test ./...
go vet ./...
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build ...
```

## Архитектура Деплоя

Docker, Docker Compose, Nginx, домен, TLS и webhook не используются.

Процесс:

```text
push в master
→ GitHub Actions запускает gofmt, tests, vet
→ собирает статический Linux-бинарник
→ загружает бинарник и checksum на VDS по SSH
→ проверяет checksum
→ атомарно заменяет бинарник
→ перезапускает systemd-сервис
→ проверяет active
→ при ошибке возвращает предыдущий бинарник
```

## Файлы На VDS

```text
/opt/telegram-publisher/telegram-publisher
/opt/telegram-publisher/telegram-publisher.previous
/etc/telegram-publisher/env
/etc/systemd/system/telegram-publisher.service
/etc/sudoers.d/telegram-publisher-deploy
```

Сервис:

```text
telegram-publisher.service
```

Пользователи:

```text
telegram-publisher  # системный пользователь приложения без интерактивного входа
deploy              # ограниченный пользователь для GitHub Actions
```

## Первичная Настройка VDS

Скрипт `scripts/bootstrap-vds.sh` запускается на VDS от `root` или через полноценный `sudo`. Он идемпотентно:

- проверяет архитектуру через `uname -m`;
- создаёт пользователя приложения `telegram-publisher`;
- создаёт пользователя деплоя `deploy`;
- создаёт `/opt/telegram-publisher`;
- создаёт или исправляет права `/etc/telegram-publisher/env`;
- устанавливает systemd unit;
- устанавливает ограниченный sudoers-файл для `deploy`;
- включает сервис в автозапуск.

Если `/etc/telegram-publisher/env` отсутствует, скрипт интерактивно запросит Telegram-токен скрытым вводом.

## GitHub Secrets И Variable

Secrets:

```text
VDS_HOST
VDS_PORT
VDS_USER
VDS_SSH_PRIVATE_KEY
VDS_KNOWN_HOSTS
```

Repository variable:

```text
VDS_GOARCH=amd64
```

`VDS_USER` должен быть `deploy`. Telegram-токен не хранится в GitHub Actions: он находится только на VDS в `/etc/telegram-publisher/env`.

После авторизации GitHub CLI secrets можно задать так:

```bash
gh secret set VDS_HOST
gh secret set VDS_PORT
gh secret set VDS_USER
gh secret set VDS_SSH_PRIVATE_KEY < ~/.ssh/telegram-publisher-deploy
gh secret set VDS_KNOWN_HOSTS
gh variable set VDS_GOARCH --body amd64
```

Перед установкой `VDS_KNOWN_HOSTS` сверяйте fingerprint host key с ключом на сервере. Workflow не использует `StrictHostKeyChecking=no`.

## GitHub Actions

Workflow находится в:

```text
.github/workflows/deploy.yml
```

Запускается:

- автоматически при push в `master`;
- вручную через `workflow_dispatch`.

Официальные Actions закреплены по полным commit SHA. Для SSH-деплоя сторонние actions не используются: только `ssh`, `scp`, `sha256sum` и `systemctl`.

## Статус И Логи

Статус сервиса:

```bash
sudo systemctl status telegram-publisher
```

Логи:

```bash
sudo journalctl -u telegram-publisher -f
```

Проверка автозапуска:

```bash
sudo systemctl is-enabled telegram-publisher.service
```

Ожидаемый результат:

```text
enabled
```

## Ручной Откат

```bash
sudo systemctl stop telegram-publisher

sudo cp \
  /opt/telegram-publisher/telegram-publisher.previous \
  /opt/telegram-publisher/telegram-publisher

sudo systemctl start telegram-publisher
```

## Перевыпуск SSH Deploy-Ключа

1. Создайте новую пару Ed25519 только для этого проекта.
2. Добавьте публичный ключ в `/home/deploy/.ssh/authorized_keys` с ограничениями:

```text
no-agent-forwarding,no-port-forwarding,no-X11-forwarding,no-pty
```

3. Замените secret `VDS_SSH_PRIVATE_KEY` в GitHub.
4. Удалите старый публичный ключ из `authorized_keys`.

Приватный ключ нельзя добавлять в Git.

## Перевыпуск Telegram-Токена

1. Перевыпустите токен через BotFather.
2. Обновите `/etc/telegram-publisher/env` на VDS.
3. Перезапустите сервис:

```bash
sudo systemctl restart telegram-publisher.service
```

Токен нельзя добавлять в Git, README, workflow или GitHub Secrets.

## Разработка

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
```

## Частые Проблемы

### Custom emoji превращаются в обычные emoji

Бот восстанавливает входящий `custom_emoji` в Rich Markdown вида:

```markdown
![😁](tg://emoji?id=...)
```

Чтобы временно проверить, приходит ли от Telegram `custom_emoji_id`, включите debug-логи:

```bash
sudo sed -i 's/^LOG_LEVEL=.*/LOG_LEVEL=debug/' /etc/telegram-publisher/env
sudo systemctl restart telegram-publisher.service
sudo journalctl -u telegram-publisher.service -f
```

В логах будет только тип entity и `custom_emoji_id`, без полного текста поста. Если ID есть, но в канале всё равно отображается обычный emoji, это ограничение Telegram для отправки кастомных emoji ботом в канал.

### Bad Request: chat not found при публикации

Предпросмотр в личном чате может работать, а публикация падать с `chat not found`, если Telegram не видит целевой канал из `TELEGRAM_CHANNEL_ID`.

Проверьте:

- username канала указан точно, включая `@`;
- для приватного канала указан числовой ID вида `-100...`;
- бот добавлен в канал администратором;
- у бота есть право публиковать сообщения;
- на VDS обновлён `/etc/telegram-publisher/env`, а сервис перезапущен.

После изменения env:

```bash
sudo systemctl restart telegram-publisher.service
```

Если бот пишет `getMe network error` или `getUpdates network error`, сервер не может установить HTTPS-соединение с `api.telegram.org`. Проверьте сетевой доступ с VDS:

```bash
curl https://api.telegram.org
```
