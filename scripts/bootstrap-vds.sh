#!/usr/bin/env bash
set -Eeuo pipefail

APP_NAME="telegram-publisher"
APP_USER="telegram-publisher"
DEPLOY_USER="deploy"
APP_DIR="/opt/$APP_NAME"
CONFIG_DIR="/etc/$APP_NAME"
CONFIG_FILE="$CONFIG_DIR/env"
SERVICE_NAME="$APP_NAME.service"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  echo "This script must be run as root or through sudo." >&2
  exit 1
fi

arch="$(uname -m)"
case "$arch" in
  x86_64)
    goarch="amd64"
    ;;
  aarch64)
    goarch="arm64"
    ;;
  *)
    echo "Unsupported VDS architecture: $arch" >&2
    exit 1
    ;;
esac

echo "Detected architecture: $arch (GOARCH=$goarch)"

if ! id "$APP_USER" >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin "$APP_USER"
fi

if ! id "$DEPLOY_USER" >/dev/null 2>&1; then
  useradd --create-home --home-dir "/home/$DEPLOY_USER" --shell /bin/bash "$DEPLOY_USER"
fi

passwd -l "$DEPLOY_USER" >/dev/null 2>&1 || true
gpasswd -d "$DEPLOY_USER" sudo >/dev/null 2>&1 || true
gpasswd -d "$DEPLOY_USER" admin >/dev/null 2>&1 || true

install -d -m 0755 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$APP_DIR"
install -d -m 0755 -o root -g root "$CONFIG_DIR"

if [[ ! -f "$CONFIG_FILE" ]]; then
  echo "Creating $CONFIG_FILE. Values are read without echo where appropriate."
  read -rsp "Telegram bot token: " TELEGRAM_BOT_TOKEN
  echo
  read -rp "Telegram owner id: " TELEGRAM_OWNER_ID
  read -rp "Telegram channel id: " TELEGRAM_CHANNEL_ID

  tmp_env="$(mktemp)"
  cat > "$tmp_env" <<EOF
TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN
TELEGRAM_OWNER_ID=$TELEGRAM_OWNER_ID
TELEGRAM_CHANNEL_ID=$TELEGRAM_CHANNEL_ID
EOF
  install -m 0600 -o root -g root "$tmp_env" "$CONFIG_FILE"
  rm -f "$tmp_env"
else
  chown root:root "$CONFIG_FILE"
  chmod 0600 "$CONFIG_FILE"
fi

ssh_dir="/home/$DEPLOY_USER/.ssh"
authorized_keys="$ssh_dir/authorized_keys"
install -d -m 0700 -o "$DEPLOY_USER" -g "$DEPLOY_USER" "$ssh_dir"
touch "$authorized_keys"
chown "$DEPLOY_USER:$DEPLOY_USER" "$authorized_keys"
chmod 0600 "$authorized_keys"

if [[ -n "${DEPLOY_PUBLIC_KEY:-}" ]] && ! grep -Fq "$DEPLOY_PUBLIC_KEY" "$authorized_keys"; then
  printf 'no-agent-forwarding,no-port-forwarding,no-X11-forwarding,no-pty %s\n' "$DEPLOY_PUBLIC_KEY" >> "$authorized_keys"
fi

bash "$REPO_ROOT/scripts/install-service.sh"

echo "Bootstrap completed."
echo "Install path: $APP_DIR/$APP_NAME"
echo "Config path: $CONFIG_FILE"
echo "Service: $SERVICE_NAME"
echo "GitHub Actions repository variable VDS_GOARCH should be: $goarch"
