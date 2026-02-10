#!/usr/bin/env bash
set -euo pipefail

if ! command -v prek >/dev/null 2>&1; then
  cat <<'EOF'
prek is not installed.

Install it (choose one):
  brew install prek
  curl --proto '=https' --tlsv1.2 -LsSf https://github.com/j178/prek/releases/latest/download/prek-installer.sh | sh

Then re-run:
  ./scripts/install-hooks.sh
EOF
  exit 1
fi

if ! command -v golangci-lint >/dev/null 2>&1; then
  cat <<'EOF'
golangci-lint is not installed.

Install it (choose one):
  brew install golangci-lint
  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

Then re-run:
  ./scripts/install-hooks.sh
EOF
  exit 1
fi

# Install git hooks (force overwrite if already installed)
# Install both pre-commit and pre-push hook scripts.
prek install -f --hook-type pre-commit --hook-type pre-push

echo "Installed git hooks via prek (pre-commit + pre-push)."
