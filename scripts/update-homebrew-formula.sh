#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Update Homebrew formula for skulls in a tap repository.

Usage:
  ./scripts/update-homebrew-formula.sh <version> [tap-dir]

Arguments:
  version   Release version tag (for example: v0.1.0)
  tap-dir   Tap repository path (default: ../homebrew-tap)

The script downloads the GitHub source tarball for the given tag,
computes SHA256, and writes Formula/skulls.rb in the tap repo.
EOF
}

if [[ ${1:-} == "-h" || ${1:-} == "--help" ]]; then
  usage
  exit 0
fi

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
  exit 2
fi

VERSION="$1"
TAP_DIR="${2:-../homebrew-tap}"

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Error: version must look like v0.1.0 (got: $VERSION)" >&2
  exit 2
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ABS_TAP_DIR="$(cd "$REPO_ROOT" && cd "$TAP_DIR" && pwd)"

if [[ ! -d "$ABS_TAP_DIR/.git" ]]; then
  echo "Error: tap dir is not a git repo: $ABS_TAP_DIR" >&2
  exit 2
fi

OWNER="kaofelix"
REPO="skulls"
TARBALL_URL="https://github.com/${OWNER}/${REPO}/archive/refs/tags/${VERSION}.tar.gz"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

tarball="$tmp/${REPO}-${VERSION}.tar.gz"

echo "Downloading $TARBALL_URL"
curl -fsSL -o "$tarball" "$TARBALL_URL"

SHA256="$(shasum -a 256 "$tarball" | awk '{print $1}')"

FORMULA_DIR="$ABS_TAP_DIR/Formula"
FORMULA_PATH="$FORMULA_DIR/skulls.rb"
mkdir -p "$FORMULA_DIR"

cat > "$FORMULA_PATH" <<EOF
class Skulls < Formula
  desc "Dead simple skills installer"
  homepage "https://github.com/${OWNER}/${REPO}"
  url "${TARBALL_URL}"
  sha256 "${SHA256}"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w", output: bin/"skulls"), "./cmd/skulls"
  end

  test do
    output = shell_output("#{bin}/skulls --help")
    assert_match "skulls", output
  end
end
EOF

echo "Updated formula: $FORMULA_PATH"
echo "Version: $VERSION"
echo "SHA256: $SHA256"
