#!/usr/bin/env bash
# Download the GitHub Release deploy bundle (compose + stack-up) and run stack-up.sh.
# No full git clone required.
set -euo pipefail

REPO="${OKTOPUS_GITHUB_REPO:-Camyil-89/oktopus}"
INSTALL_DIR="${OKTOPUS_INSTALL_DIR:-./oktopus}"
VERSION=""
FORCE=false
FETCH_ONLY=false
FETCH_RELEASE=false
STACK_UP_ARGS=()

usage() {
  cat <<'EOF'
Usage: install.sh [options] [-- stack-up options]

  Downloads oktopus-<version>-deploy.tar.gz from GitHub Releases, extracts it,
  then runs scripts/stack-up.sh (pull images + docker compose up).

Options:
  --version VER     Release version without leading v (default: latest release)
  --dir PATH        Install directory (default: ./oktopus or OKTOPUS_INSTALL_DIR)
  --force           Re-download and replace bundle in the install directory
  --fetch-only      Download and extract only; do not run stack-up
  --fetch           Fetch release even when run from a git checkout
  -h, --help

stack-up options (after --): --write-env-only, --build

Environment:
  OKTOPUS_VERSION, OKTOPUS_INSTALL_DIR, OKTOPUS_GITHUB_REPO

One-liner (Linux/macOS):
  curl -fsSL https://raw.githubusercontent.com/Camyil-89/oktopus/main/scripts/install.sh | bash

Non-interactive example:
  OKTOPUS_API_HOST=203.0.113.10 OKTOPUS_API_PORT=8000 curl -fsSL ... | bash
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:?--version requires a value}"
      shift 2
      ;;
    --dir)
      INSTALL_DIR="${2:?--dir requires a path}"
      shift 2
      ;;
    --force) FORCE=true; shift ;;
    --fetch-only) FETCH_ONLY=true; shift ;;
    --fetch) FETCH_RELEASE=true; shift ;;
    -h|--help) usage; exit 0 ;;
    --)
      shift
      STACK_UP_ARGS+=("$@")
      break
      ;;
    --write-env-only|--build)
      STACK_UP_ARGS+=("$1")
      shift
      ;;
    *)
      echo "Unknown option: $1 (use --help)" >&2
      exit 1
      ;;
  esac
done

if [[ -n "${OKTOPUS_VERSION:-}" && -z "$VERSION" ]]; then
  VERSION="${OKTOPUS_VERSION#v}"
fi

script_dir=""
if [[ -n "${BASH_SOURCE[0]:-}" && "${BASH_SOURCE[0]}" != "-" && -f "${BASH_SOURCE[0]}" ]]; then
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

repo_root=""
if [[ -n "$script_dir" && -f "$script_dir/../docker-compose.yml" ]]; then
  repo_root="$(cd "$script_dir/.." && pwd)"
fi

if [[ -n "$repo_root" && "$FETCH_RELEASE" == false && -z "$VERSION" && "$FORCE" == false ]]; then
  echo "Using local checkout at $repo_root (pass --fetch to download a release bundle instead)."
  cd "$repo_root"
  exec "$repo_root/scripts/stack-up.sh" "${STACK_UP_ARGS[@]+"${STACK_UP_ARGS[@]}"}"
fi

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "$1 is required but not found in PATH." >&2
    exit 1
  fi
}

need_cmd curl
need_cmd tar
need_cmd docker

resolve_latest_version() {
  local api="https://api.github.com/repos/${REPO}/releases/latest"
  if command -v jq >/dev/null 2>&1; then
    local tag
    tag="$(curl -fsSL "$api" | jq -r '.tag_name')"
    if [[ -z "$tag" || "$tag" == "null" ]]; then
      echo "Could not read latest release tag from GitHub." >&2
      exit 1
    fi
    echo "${tag#v}"
    return
  fi
  local tag_line
  tag_line="$(curl -fsSL "$api" | grep -m1 '"tag_name":' || true)"
  if [[ -z "$tag_line" ]]; then
    echo "Could not read latest release from GitHub (install jq for reliable parsing)." >&2
    exit 1
  fi
  local tag
  tag="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\(.*\)".*/\1/p' <<<"$tag_line")"
  echo "${tag#v}"
}

if [[ -z "$VERSION" ]]; then
  echo "Resolving latest release..."
  VERSION="$(resolve_latest_version)"
fi

INSTALL_DIR="$(mkdir -p "$INSTALL_DIR" && cd "$INSTALL_DIR" && pwd)"
MARKER="$INSTALL_DIR/.oktopus-bundle-version"
BUNDLE_NAME="oktopus-${VERSION}-deploy.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${BUNDLE_NAME}"

should_download=true
if [[ -f "$INSTALL_DIR/docker-compose.yml" && "$FORCE" == false ]]; then
  if [[ -f "$MARKER" ]] && [[ "$(cat "$MARKER")" == "$VERSION" ]]; then
    should_download=false
  elif [[ ! -f "$MARKER" ]]; then
    echo "Install directory already contains compose files; use --force to replace or set --version."
    should_download=false
  fi
fi

if [[ "$should_download" == true ]]; then
  tmp="$(mktemp "${TMPDIR:-/tmp}/oktopus-bundle.XXXXXX.tar.gz")"
  trap 'rm -f "$tmp"' EXIT
  echo "Downloading ${URL}"
  if ! curl -fsSL -o "$tmp" "$URL"; then
    echo "Download failed. Check version (${VERSION}) and https://github.com/${REPO}/releases" >&2
    exit 1
  fi
  echo "Extracting to ${INSTALL_DIR}"
  tar -xzf "$tmp" -C "$INSTALL_DIR"
  printf '%s\n' "$VERSION" >"$MARKER"
  rm -f "$tmp"
  trap - EXIT
fi

chmod +x "$INSTALL_DIR/scripts/stack-up.sh" 2>/dev/null || true

if [[ "$FETCH_ONLY" == true ]]; then
  echo "Bundle ready in ${INSTALL_DIR}"
  exit 0
fi

cd "$INSTALL_DIR"
exec "$INSTALL_DIR/scripts/stack-up.sh" "${STACK_UP_ARGS[@]+"${STACK_UP_ARGS[@]}"}"