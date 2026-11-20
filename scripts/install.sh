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
Set TAG=dev in .env to use dev prereleases and images.

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
local_runner="${repo_root:+$repo_root/scripts/stack-up.sh}"

export OKTOPUS_STACK_CONFIG=/etc/oktopus/stack.env
export OKTOPUS_STACK_PROJECT=oktopus-onprem
export OKTOPUS_STACK_MANAGED=1

manage_stack() {
  local ids container installed_dir choice confirmation images image used resources
  local -a containers volumes networks
  local project_filter="label=com.docker.compose.project=$OKTOPUS_STACK_PROJECT"
  ids=$(docker ps -aq --filter "$project_filter")
  if [[ -z "$ids" ]]; then
    resources=$(docker volume ls -q --filter "$project_filter")
    [[ -n "$resources" && -s /etc/oktopus/install-dir ]] || return 0
    IFS= read -r installed_dir </etc/oktopus/install-dir
  else
    container=${ids%%$'\n'*}
    installed_dir=$(docker inspect --format '{{index .Config.Labels "com.docker.compose.project.working_dir"}}' "$container")
    [[ "$installed_dir" == /* ]] || { echo 'Cannot determine the Oktopus installation directory.' >&2; exit 1; }
  fi

  echo "Oktopus installation: $installed_dir"
  if ! read_stack_reply 'Update [u], delete [d], or cancel [Enter]: ' choice; then
    echo 'An interactive terminal is required to manage the installed stack.' >&2
    exit 1
  fi
  case "$choice" in
    u|U)
      INSTALL_DIR=$installed_dir
      repo_root=$installed_dir
      if [[ -f "$installed_dir/.oktopus-bundle-version" && "${OKTOPUS_LOCAL_STACK:-}" != 1 ]]; then
        FETCH_RELEASE=true
      fi
      ;;
    d|D)
      read_stack_reply 'Type delete to permanently remove Oktopus Docker volumes and their data: ' confirmation
      [[ "$confirmation" == delete ]] || exit 0
      if [[ -n "$ids" ]]; then
        mapfile -t containers <<<"$ids"
        images=$(docker inspect --format '{{.Image}}' "${containers[@]}" | sort -u)
        docker rm -f "${containers[@]}"
        while IFS= read -r image; do
          used=$(docker ps -aq --filter "ancestor=$image")
          [[ -z "$used" ]] || continue
          docker image rm "$image" || echo "Image retained: $image" >&2
        done <<<"$images"
      fi
      resources=$(docker volume ls -q --filter "$project_filter")
      if [[ -n "$resources" ]]; then
        mapfile -t volumes <<<"$resources"
        docker volume rm "${volumes[@]}"
      fi
      resources=$(docker network ls -q --filter "$project_filter")
      if [[ -n "$resources" ]]; then
        mapfile -t networks <<<"$resources"
        docker network rm "${networks[@]}"
      fi
      rm -f -- "$OKTOPUS_STACK_CONFIG" /etc/oktopus/install-dir
      exit 0
      ;;
    *) exit 0 ;;
  esac
}

read_stack_reply() {
  if [[ -t 0 ]]; then
    read -r -p "$1" "$2"
  else
    read -r -p "$1" "$2" </dev/tty
  fi
}

if [[ "$FETCH_ONLY" == false && " ${STACK_UP_ARGS[*]} " != *' --write-env-only '* ]]; then
  manage_stack
fi

read_channel() {
  [[ -f "$1" ]] || return 0
  sed -nE "s/^[[:space:]]*TAG[[:space:]]*=[[:space:]]*['\"]?(dev|stable)['\"]?([[:space:]]*#.*|[[:space:]]*)$/\1/p" "$1" | tail -1
}

channel=$(read_channel "$PWD/.env")
[[ -n "$channel" ]] || channel=$(read_channel "$INSTALL_DIR/.env")
if [[ -z "$channel" && -n "$repo_root" ]]; then
  channel=$(read_channel "$repo_root/.env")
fi
VERSION=${VERSION#v}
if [[ "$channel" == dev || "$VERSION" == *-dev ]]; then
  channel=dev
  FETCH_RELEASE=true
elif [[ -f "$INSTALL_DIR/.oktopus-bundle-version" &&
        "$(cat "$INSTALL_DIR/.oktopus-bundle-version")" == *-dev ]]; then
  if [[ "$channel" == stable ]]; then
    FETCH_RELEASE=true
  elif [[ -z "$channel" ]]; then
    channel=dev
  fi
fi
export OKTOPUS_STACK_CHANNEL="$channel"
export OKTOPUS_STACK_IMAGE_TAG=latest

if [[ -n "$repo_root" && "$FETCH_ONLY" == false ]] &&
   [[ "$FETCH_RELEASE" == false && ( "${OKTOPUS_LOCAL_STACK:-}" == 1 || ( -z "$VERSION" && "$FORCE" == false ) ) ]]; then
  echo "Using deployment at $repo_root (pass --fetch to download a release bundle instead)."
  cd "$repo_root"
  export OKTOPUS_STACK_ROOT="$repo_root"
  exec bash "${local_runner:-$repo_root/scripts/stack-up.sh}" "${STACK_UP_ARGS[@]+"${STACK_UP_ARGS[@]}"}"
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
  if [[ "$channel" == dev ]]; then
    local page=1 response tags tag
    while :; do
      response=$(curl -fsSL "${api%/latest}?per_page=100&page=$page") || return 1
      tags=$(grep -oE '"tag_name"[[:space:]]*:[[:space:]]*"[^"]+"' <<<"$response" | cut -d '"' -f 4 || true)
      tag=$(awk '/^v.*-dev$/ {print; exit}' <<<"$tags")
      if [[ -n "$tag" ]]; then
        echo "${tag#v}"
        return
      fi
      [[ $(wc -l <<<"$tags") -ge 100 ]] || break
      page=$((page + 1))
    done
    echo 'No dev prerelease found.' >&2
    return 1
  fi
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
  echo "Resolving latest ${channel:-stable} release..."
  VERSION="$(resolve_latest_version)"
elif [[ "$channel" == dev ]]; then
  VERSION="${VERSION%-dev}-dev"
  export OKTOPUS_STACK_IMAGE_TAG="$VERSION"
fi

INSTALL_DIR="$(mkdir -p "$INSTALL_DIR" && cd "$INSTALL_DIR" && pwd)"
MARKER="$INSTALL_DIR/.oktopus-bundle-version"
BUNDLE_NAME="oktopus-${VERSION}-deploy.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${BUNDLE_NAME}"

should_download=true
if [[ -f "$INSTALL_DIR/docker-compose.yml" && "$FORCE" == false ]]; then
  if [[ -f "$MARKER" ]] && [[ "$(cat "$MARKER")" == "$VERSION" ]]; then
    should_download=false
  elif [[ ! -f "$MARKER" && "$channel" == dev && -e "$INSTALL_DIR/.git" ]]; then
    echo 'Use --dir with a separate directory, or --force to replace files in this checkout.' >&2
    exit 1
  elif [[ ! -f "$MARKER" && "$channel" != dev ]]; then
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
export OKTOPUS_STACK_ROOT="$INSTALL_DIR"
exec bash "${local_runner:-$INSTALL_DIR/scripts/stack-up.sh}" "${STACK_UP_ARGS[@]+"${STACK_UP_ARGS[@]}"}"
