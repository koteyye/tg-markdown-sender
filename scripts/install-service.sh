#!/usr/bin/env bash
set -Eeuo pipefail

APP_NAME="telegram-publisher"
SERVICE_NAME="$APP_NAME.service"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVICE_SRC="$REPO_ROOT/deploy/$SERVICE_NAME"
SUDOERS_TEMPLATE="$REPO_ROOT/deploy/$APP_NAME.sudoers"
SERVICE_DST="/etc/systemd/system/$SERVICE_NAME"
SUDOERS_DST="/etc/sudoers.d/$APP_NAME-deploy"

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  echo "This script must be run as root." >&2
  exit 1
fi

SYSTEMCTL="$(command -v systemctl)"
JOURNALCTL="$(command -v journalctl)"
VISUDO="$(command -v visudo)"

tmp_dir="$(mktemp -d)"
tmp_service="$tmp_dir/$SERVICE_NAME"
tmp_sudoers="$(mktemp)"
cleanup() {
  rm -rf "$tmp_dir"
  rm -f "$tmp_sudoers"
}
trap cleanup EXIT

install -m 0644 "$SERVICE_SRC" "$tmp_service"

if command -v systemd-analyze >/dev/null 2>&1; then
  systemd-analyze verify "$tmp_service"
fi

sed \
  -e "s#@SYSTEMCTL@#$SYSTEMCTL#g" \
  -e "s#@JOURNALCTL@#$JOURNALCTL#g" \
  "$SUDOERS_TEMPLATE" > "$tmp_sudoers"

chmod 0440 "$tmp_sudoers"
"$VISUDO" -cf "$tmp_sudoers"

install -m 0644 "$tmp_service" "$SERVICE_DST"
install -m 0440 "$tmp_sudoers" "$SUDOERS_DST"

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
