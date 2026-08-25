#!/usr/bin/env bash
set -e

# Resolve repository root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "==> Running mail_cli_fake example..."
cd "$ROOT_DIR"
go run ./example/mail_cli_fake "$@"
