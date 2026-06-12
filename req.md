# Задача: настроить автодеплой Go Telegram-бота на VDS через GitHub Actions

## 1. Цель

Необходимо полностью настроить сборку и автоматическое развёртывание Go-приложения на существующий Linux VDS.

Приложение представляет собой Telegram-бота, работающего через Long Polling. Входящие порты, HTTP-сервер, домен, Nginx и TLS-сертификаты не требуются.

Docker не использовать.

Итоговый процесс должен выглядеть так:

```text
push в main
→ GitHub Actions запускает тесты
→ собирает статический Linux-бинарник
→ загружает бинарник на VDS по SSH
→ атомарно заменяет текущую версию
→ перезапускает systemd-сервис
→ проверяет успешный запуск
→ при ошибке возвращает предыдущую версию
```

## 2. Исходные условия

Характеристики VDS:

```text
1 vCore
1 GB RAM
40 GB SSD
```

Приложение должно запускаться как один статический Go-бинарник.

Использовать версию Go, указанную в `go.mod`. Если версия в `go.mod` отсутствует или некорректна, сначала согласовать изменение.

Название приложения и сервиса:

```text
telegram-publisher
```

Путь установки:

```text
/opt/telegram-publisher/telegram-publisher
```

Файл конфигурации:

```text
/etc/telegram-publisher/env
```

Название systemd-сервиса:

```text
telegram-publisher.service
```

Целевая ветка для автоматического деплоя:

```text
main
```

## 3. Общие требования к работе агента

Агент должен не просто написать инструкцию, а выполнить доступные действия:

1. Проанализировать текущую структуру репозитория.
2. Проверить `go.mod`, путь до `main`-пакета и текущие команды сборки.
3. Проверить архитектуру VDS командой:

```bash
uname -m
```

4. Определить соответствующий `GOARCH`:

```text
x86_64  → amd64
aarch64 → arm64
```

5. Подготовить необходимые файлы в репозитории.
6. Подключиться к VDS и выполнить первоначальную настройку.
7. Настроить отдельного пользователя приложения.
8. Настроить отдельного пользователя деплоя.
9. Создать SSH-ключ исключительно для GitHub Actions.
10. Настроить GitHub Actions Secrets.
11. Запустить первый деплой.
12. Проверить состояние приложения и логи.
13. Не считать задачу завершённой, пока workflow не выполнится успешно.

Если у агента отсутствует доступ к GitHub, VDS или `sudo`, он должен остановиться только в точке, где требуется действие пользователя, и дать одну конкретную команду или запросить одно конкретное разрешение.

Не перекладывать на пользователя действия, которые агент может выполнить самостоятельно.

## 4. Работа с секретами

Агент не должен:

* выводить токен Telegram-бота в логах;
* добавлять токен в Git;
* добавлять приватный SSH-ключ в Git;
* сохранять секреты в README;
* помещать секреты в workflow-файл;
* печатать полные значения секретов в итоговом отчёте.

Если секрет необходимо запросить у пользователя в терминале, использовать скрытый ввод:

```bash
read -rsp "Telegram bot token: " TELEGRAM_BOT_TOKEN
echo
```

В репозитории должен находиться только файл:

```text
.env.example
```

Пример:

```env
TELEGRAM_BOT_TOKEN=
TELEGRAM_OWNER_ID=
TELEGRAM_CHANNEL_ID=
```

Файлы `.env`, `env`, приватные ключи и временные бинарники должны быть добавлены в `.gitignore`.

## 5. Подготовка Go-приложения

Проверить, что приложение:

1. Корректно обрабатывает `SIGTERM` и `SIGINT`.
2. Завершает Long Polling при остановке.
3. Проверяет обязательные переменные окружения при запуске.
4. Завершается с ненулевым кодом при некорректной конфигурации.
5. Не выводит токен Telegram в лог.
6. Выполняет `getMe` при запуске для проверки токена и доступности Telegram API.
7. Не запускает обработку обновлений, если `getMe` завершился ошибкой.
8. Логирует успешный запуск без секретных данных.

Обязательные переменные:

```env
TELEGRAM_BOT_TOKEN=
TELEGRAM_OWNER_ID=
TELEGRAM_CHANNEL_ID=
```

`TELEGRAM_OWNER_ID` должен парситься как `int64`.

Если приложение уже соответствует требованиям, не переписывать его без необходимости.

## 6. Файлы, которые необходимо добавить в репозиторий

Минимальный набор:

```text
.github/workflows/deploy.yml
deploy/telegram-publisher.service
deploy/telegram-publisher.sudoers
scripts/build-linux.sh
scripts/bootstrap-vds.sh
scripts/install-service.sh
.env.example
README.md
```

Скрипты должны использовать:

```bash
set -Eeuo pipefail
```

Скрипты должны быть идемпотентными: повторный запуск не должен ломать существующую установку.

## 7. Сборка приложения

Создать:

```text
scripts/build-linux.sh
```

Скрипт должен:

1. Получать архитектуру через переменную `GOARCH`.
2. По умолчанию завершаться ошибкой, если архитектура не указана.
3. Запускать тесты.
4. Запускать `go vet`.
5. Собирать статический бинарник.
6. Создавать SHA-256 checksum.

Пример команды сборки:

```bash
CGO_ENABLED=0 \
GOOS=linux \
GOARCH="$GOARCH" \
go build \
  -trimpath \
  -ldflags="-s -w" \
  -o dist/telegram-publisher \
  ./cmd/bot
```

Путь `./cmd/bot` необходимо скорректировать, если фактический `main`-пакет находится в другом месте.

Результат:

```text
dist/telegram-publisher
dist/telegram-publisher.sha256
```

## 8. Пользователь приложения на VDS

Создать системного пользователя:

```text
telegram-publisher
```

Требования:

* без домашней директории;
* без возможности интерактивного входа;
* без пароля;
* без `sudo`;
* используется только для запуска systemd-сервиса.

Пример:

```bash
sudo useradd \
  --system \
  --no-create-home \
  --shell /usr/sbin/nologin \
  telegram-publisher
```

Команда должна выполняться только в том случае, если пользователь ещё не существует.

## 9. Пользователь для деплоя

Создать отдельного пользователя:

```text
deploy
```

Требования:

* домашняя директория `/home/deploy`;
* вход только по SSH-ключу;
* пароль заблокирован;
* пользователь не должен иметь полноценный root-доступ;
* пользователь может записывать файлы только в каталог приложения;
* через `sudo` разрешены только ограниченные команды управления конкретным сервисом.

Создать директорию:

```text
/opt/telegram-publisher
```

Владельцем директории должен быть пользователь `deploy`.

Бинарник должен быть доступен на чтение и выполнение пользователю `telegram-publisher`.

## 10. Ограниченный sudo

Создать файл:

```text
/etc/sudoers.d/telegram-publisher-deploy
```

Перед установкой проверить расположение команд:

```bash
command -v systemctl
command -v journalctl
```

Пользователю `deploy` разрешить без пароля только:

```text
systemctl restart telegram-publisher.service
systemctl is-active --quiet telegram-publisher.service
systemctl status telegram-publisher.service --no-pager
journalctl -u telegram-publisher.service -n 100 --no-pager
```

Запретить выполнение любых других команд через `sudo`.

Файл должен проверяться командой:

```bash
sudo visudo -cf /etc/sudoers.d/telegram-publisher-deploy
```

Не устанавливать некорректный sudoers-файл.

## 11. SSH-ключ для GitHub Actions

Создать отдельную пару ключей Ed25519:

```text
github-actions-telegram-publisher
```

Ключ не должен использоваться для других серверов или проектов.

Публичный ключ добавить в:

```text
/home/deploy/.ssh/authorized_keys
```

Установить права:

```text
/home/deploy/.ssh                   0700
/home/deploy/.ssh/authorized_keys   0600
```

В строке `authorized_keys` использовать ограничения:

```text
no-agent-forwarding,no-port-forwarding,no-X11-forwarding,no-pty
```

Приватный ключ сохранить только как GitHub Actions Secret.

После добавления секрета удалить локальную временную копию приватного ключа, если она больше не требуется.

## 12. Конфигурация приложения на VDS

Создать файл:

```text
/etc/telegram-publisher/env
```

Содержимое:

```env
TELEGRAM_BOT_TOKEN=...
TELEGRAM_OWNER_ID=...
TELEGRAM_CHANNEL_ID=...
```

Права:

```bash
sudo chown root:root /etc/telegram-publisher/env
sudo chmod 600 /etc/telegram-publisher/env
```

Файл не должен загружаться через GitHub Actions при каждом деплое.

Telegram-секреты должны храниться только на VDS.

## 13. Systemd-сервис

Создать файл в репозитории:

```text
deploy/telegram-publisher.service
```

Базовая конфигурация:

```ini
[Unit]
Description=Telegram Rich Markdown Publisher
After=network-online.target
Wants=network-online.target

[Service]
Type=simple

User=telegram-publisher
Group=telegram-publisher

WorkingDirectory=/opt/telegram-publisher
EnvironmentFile=/etc/telegram-publisher/env

ExecStart=/opt/telegram-publisher/telegram-publisher

Restart=on-failure
RestartSec=5

TimeoutStopSec=15
KillSignal=SIGTERM

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectKernelLogs=true
ProtectControlGroups=true
RestrictSUIDSGID=true
LockPersonality=true
RestrictAddressFamilies=AF_UNIX AF_INET AF_INET6

MemoryMax=256M

StandardOutput=journal
StandardError=journal
SyslogIdentifier=telegram-publisher

[Install]
WantedBy=multi-user.target
```

Перед окончательной установкой проверить совместимость защитных параметров с версией `systemd` на VDS.

Если конкретная директива не поддерживается установленной версией `systemd`, удалить только несовместимую директиву и зафиксировать это в отчёте.

Проверить unit-файл:

```bash
systemd-analyze verify deploy/telegram-publisher.service
```

Установить:

```text
/etc/systemd/system/telegram-publisher.service
```

Затем выполнить:

```bash
sudo systemctl daemon-reload
sudo systemctl enable telegram-publisher.service
```

## 14. GitHub Secrets

Настроить в GitHub Actions следующие secrets:

```text
VDS_HOST
VDS_PORT
VDS_USER
VDS_SSH_PRIVATE_KEY
VDS_KNOWN_HOSTS
```

Значения:

```text
VDS_USER=deploy
```

`VDS_PORT` должен соответствовать фактическому SSH-порту.

`VDS_KNOWN_HOSTS` должен содержать проверенный host key VDS.

Не использовать:

```bash
-o StrictHostKeyChecking=no
```

Не принимать host key вслепую.

Fingerprint host key необходимо сверить с ключом на самом сервере.

Если доступен авторизованный GitHub CLI, использовать:

```bash
gh secret set
```

Если GitHub CLI не авторизован, агент должен подготовить точные команды для установки secrets и попросить пользователя только выполнить авторизацию.

Создать repository variable:

```text
VDS_GOARCH
```

Значение:

```text
amd64
```

или:

```text
arm64
```

в зависимости от архитектуры сервера.

## 15. GitHub Actions workflow

Создать:

```text
.github/workflows/deploy.yml
```

Workflow должен поддерживать:

```yaml
on:
  push:
    branches:
      - main
  workflow_dispatch:
```

Установить минимальные разрешения:

```yaml
permissions:
  contents: read
```

Не допускать параллельные деплои:

```yaml
concurrency:
  group: telegram-publisher-production
  cancel-in-progress: false
```

Workflow должен выполнять следующие шаги:

1. Checkout репозитория.
2. Установку версии Go из `go.mod`.
3. Проверку форматирования:

```bash
test -z "$(gofmt -l .)"
```

4. Запуск:

```bash
go test ./...
```

5. Запуск:

```bash
go vet ./...
```

6. Сборку статического бинарника для `VDS_GOARCH`.
7. Создание SHA-256 checksum.
8. Настройку SSH из GitHub Secrets.
9. Загрузку файлов на VDS под временными именами.
10. Проверку checksum на VDS.
11. Выдачу бинарнику прав `0755`.
12. Сохранение текущего бинарника как предыдущей версии.
13. Атомарную замену бинарника через `mv` внутри одного файлового раздела.
14. Перезапуск systemd-сервиса.
15. Ожидание не менее пяти секунд.
16. Проверку:

```bash
sudo systemctl is-active --quiet telegram-publisher.service
```

17. При неуспешной проверке:

    * вывести статус сервиса;
    * вывести последние 100 строк логов;
    * вернуть предыдущий бинарник;
    * повторно перезапустить сервис;
    * завершить workflow с ошибкой.

18. При успехе:

    * удалить checksum и временные файлы;
    * оставить только одну предыдущую версию бинарника;
    * завершить workflow успешно.

## 16. Требования к atomic deploy

На сервер загружать:

```text
/opt/telegram-publisher/telegram-publisher.new
/opt/telegram-publisher/telegram-publisher.sha256
```

Текущий бинарник:

```text
/opt/telegram-publisher/telegram-publisher
```

Предыдущая версия:

```text
/opt/telegram-publisher/telegram-publisher.previous
```

Алгоритм:

```text
проверить checksum
→ chmod 0755 telegram-publisher.new
→ скопировать текущий бинарник в telegram-publisher.previous
→ mv telegram-publisher.new telegram-publisher
→ restart systemd
→ проверить запуск
→ при ошибке вернуть telegram-publisher.previous
```

Не удалять текущий рабочий бинарник до успешной загрузки и проверки нового файла.

## 17. Использование GitHub Actions

Не подключать малоизвестные сторонние Actions для SSH-деплоя.

Для передачи файлов и выполнения команд использовать стандартные системные утилиты:

```text
ssh
scp
sha256sum
```

Официальные Actions подключать с фиксацией на полный commit SHA.

Рядом с SHA оставить комментарий с названием версии, например:

```yaml
uses: actions/checkout@<FULL_COMMIT_SHA> # v4.x
```

Перед фиксацией SHA проверить актуальную стабильную версию официального Action.

## 18. Первый деплой

После настройки необходимо:

1. Запустить workflow вручную через `workflow_dispatch`.
2. Дождаться успешного завершения всех шагов.
3. Проверить на VDS:

```bash
sudo systemctl status telegram-publisher.service
```

4. Проверить:

```bash
sudo journalctl \
  -u telegram-publisher.service \
  -n 100 \
  --no-pager
```

5. Убедиться, что сервис находится в состоянии:

```text
active (running)
```

6. Убедиться, что приложение успешно выполнило `getMe`.
7. Убедиться, что приложение начало Long Polling.
8. Перезагрузить VDS либо отдельно проверить, что сервис включён в автозапуск:

```bash
sudo systemctl is-enabled telegram-publisher.service
```

Ожидаемый результат:

```text
enabled
```

## 19. Проверка безопасности

Проверить:

* Telegram-токен отсутствует в Git;
* Telegram-токен отсутствует в GitHub Actions;
* приватный SSH-ключ отсутствует в Git;
* пользователь `deploy` не имеет полноценного `sudo`;
* пользователь `telegram-publisher` не может войти по SSH;
* приложение не слушает входящие TCP-порты;
* firewall не изменялся без необходимости;
* в workflow отсутствует `StrictHostKeyChecking=no`;
* `.env` и приватные ключи находятся в `.gitignore`;
* workflow не выводит secrets;
* бот публикует сообщения только после проверки `TELEGRAM_OWNER_ID`.

## 20. README

Обновить README и описать:

1. Архитектуру развёртывания.
2. Локальный запуск.
3. Локальную сборку Linux-бинарника.
4. Расположение файлов на VDS.
5. Назначение GitHub Secrets.
6. Ручной запуск workflow.
7. Автоматический деплой через push в `main`.
8. Просмотр статуса:

```bash
sudo systemctl status telegram-publisher
```

9. Просмотр логов:

```bash
sudo journalctl -u telegram-publisher -f
```

10. Ручной откат:

```bash
sudo systemctl stop telegram-publisher

sudo cp \
  /opt/telegram-publisher/telegram-publisher.previous \
  /opt/telegram-publisher/telegram-publisher

sudo systemctl start telegram-publisher
```

11. Перевыпуск SSH deploy-ключа.
12. Перевыпуск Telegram-токена при утечке.

README не должен содержать реальные IP-адреса, токены и приватные ключи.

## 21. Что не требуется

В рамках задачи не использовать и не настраивать:

* Docker;
* Docker Compose;
* Kubernetes;
* Nginx;
* Caddy;
* домен;
* HTTPS;
* webhook;
* открытый порт для приложения;
* базу данных;
* облачное хранилище секретов;
* полноценный root-доступ для GitHub Actions.

## 22. Критерии приёмки

Задача считается выполненной, если:

1. `go test ./...` проходит успешно.
2. `go vet ./...` проходит успешно.
3. Репозиторий содержит рабочий workflow.
4. Push в `main` запускает деплой.
5. Ручной запуск через `workflow_dispatch` работает.
6. Бинарник собирается под архитектуру VDS.
7. Бинарник доставляется на VDS по SSH.
8. Checksum проверяется до установки.
9. Systemd-сервис автоматически перезапускается.
10. Сервис после деплоя находится в состоянии `active`.
11. При неуспешном запуске предусмотрен автоматический rollback.
12. После перезагрузки VDS сервис запускается автоматически.
13. Telegram-токен хранится только на VDS.
14. GitHub Actions использует отдельный ограниченный SSH-ключ.
15. В GitHub и Git отсутствуют секреты.
16. Бот отвечает владельцу и не позволяет постороннему пользователю публиковать сообщения.
17. Агент предоставил итоговый отчёт о выполненных изменениях.

## 23. Итоговый отчёт агента

После выполнения агент должен предоставить:

* список созданных и изменённых файлов;
* архитектуру VDS;
* версию Go;
* название systemd-сервиса;
* путь до установленного бинарника;
* список настроенных GitHub Secrets без их значений;
* ссылку или идентификатор успешного GitHub Actions run;
* результат `systemctl is-active`;
* результат `systemctl is-enabled`;
* краткую инструкцию следующего обновления;
* перечень действий, которые не удалось выполнить, с точной причиной.

Не выводить значения токенов, приватных ключей и других секретов.
