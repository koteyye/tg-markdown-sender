#!/usr/bin/env bash
set -Eeuo pipefail

APP_NAME="telegram-publisher"
MAIN_PACKAGE="./cmd/bot"
DIST_DIR="dist"

if [[ -z "${GOARCH:-}" ]]; then
  echo "GOARCH is required. Supported values for this project: amd64 or arm64." >&2
  exit 1
fi

case "$GOARCH" in
  amd64|arm64)
    ;;
  *)
    echo "Unsupported GOARCH: $GOARCH" >&2
    exit 1
    ;;
esac

go test ./...
go vet ./...

mkdir -p "$DIST_DIR"

CGO_ENABLED=0 \
GOOS=linux \
GOARCH="$GOARCH" \
go build \
  -trimpath \
  -ldflags="-s -w" \
  -o "$DIST_DIR/$APP_NAME" \
  "$MAIN_PACKAGE"

(
  cd "$DIST_DIR"
  sha256sum "$APP_NAME" > "$APP_NAME.sha256"
)
