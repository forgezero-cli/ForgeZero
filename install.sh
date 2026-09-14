#!/bin/sh
set -eu

# Copyright (c) 2026 forgezero-cli
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

die() {
    printf '%s\n' "forgezero-install: $1" >&2
    exit 1
}

usage() {
    cat <<'EOF'
Usage: install.sh [options]

  -d, --dest DIR          installation directory
  -o, --os OS             linux, darwin, or windows
  -a, --arch ARCH         amd64 or arm64
  -v, --version VERSION   release tag, or latest
  -r, --repo OWNER/REPO   GitHub repository, empty for local release
  -f, --force             replace an existing binary
      --dry-run           print the operation without installing
  -h, --help              show this help
EOF
}

need_value() {
    [ "$#" -ge 2 ] || die "missing value for $1"
}

OS=
ARCH=
DEST=
VERSION=latest
REPO=forgezero-cli/ForgeZero
FORCE=0
DRY_RUN=0

while [ "$#" -gt 0 ]; do
    case "$1" in
        -d|--dest)
            need_value "$@"
            DEST=$2
            shift 2
            ;;
        -o|--os)
            need_value "$@"
            OS=$2
            shift 2
            ;;
        -a|--arch)
            need_value "$@"
            ARCH=$2
            shift 2
            ;;
        -v|--version)
            need_value "$@"
            VERSION=$2
            shift 2
            ;;
        -r|--repo)
            need_value "$@"
            REPO=$2
            shift 2
            ;;
        -f|--force)
            FORCE=1
            shift
            ;;
        --dry-run)
            DRY_RUN=1
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        --)
            shift
            break
            ;;
        *)
            die "unknown argument: $1"
            ;;
    esac
done

UNAME_S=$(uname -s 2>/dev/null || printf 'unknown')
UNAME_M=$(uname -m 2>/dev/null || printf 'unknown')

if [ -z "$OS" ]; then
    case "$UNAME_S" in
        Linux*) OS=linux ;;
        Darwin*) OS=darwin ;;
        MINGW*|MSYS*|CYGWIN*) OS=windows ;;
        *) die "unsupported operating system: $UNAME_S" ;;
    esac
fi

case "$OS" in
    linux|darwin|windows) ;;
    *) die "unsupported operating system: $OS" ;;
esac

if [ -z "$ARCH" ]; then
    case "$UNAME_M" in
        x86_64|amd64) ARCH=amd64 ;;
        aarch64|arm64) ARCH=arm64 ;;
        *) die "unsupported architecture: $UNAME_M" ;;
    esac
fi

case "$ARCH" in
    amd64|arm64) ;;
    *) die "unsupported architecture: $ARCH" ;;
esac

BIN_FILE="fz-$OS-$ARCH"
if [ "$OS" = windows ]; then
    BIN_FILE="$BIN_FILE.exe"
fi

TMP_DIR=
cleanup() {
    if [ -n "$TMP_DIR" ]; then
        rm -rf "$TMP_DIR"
    fi
}
trap cleanup 0
trap 'cleanup; exit 1' 1 2 15

download() {
    url=$1
    output=$2
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL --retry 3 --connect-timeout 15 -o "$output" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -q --tries=3 --timeout=15 -O "$output" "$url"
    else
        die "curl or wget is required"
    fi
}

verify_checksum() {
    source=$1
    checksum=$2
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$source" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$source" | awk '{print $1}')
    else
        die "sha256sum or shasum is required"
    fi
    expected=$(awk 'NF {print $1; exit}' "$checksum")
    [ -n "$expected" ] || die "empty checksum file: $checksum"
    [ "$actual" = "$expected" ] || die "checksum mismatch for $source"
}

if [ -n "$REPO" ]; then
    TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/forgezero-install.XXXXXX") || die "cannot create temporary directory"
    SOURCE_PATH="$TMP_DIR/$BIN_FILE"
    if [ "$VERSION" = latest ]; then
        BASE_URL="https://github.com/$REPO/releases/latest/download"
    else
        BASE_URL="https://github.com/$REPO/releases/download/$VERSION"
    fi
    DOWNLOAD_URL="$BASE_URL/$BIN_FILE"
    CHECKSUM_URL="$DOWNLOAD_URL.sha256"
    CHECKSUM_PATH="$TMP_DIR/$BIN_FILE.sha256"
    if [ "$DRY_RUN" -eq 1 ]; then
        printf 'Would download: %s\n' "$DOWNLOAD_URL"
        printf 'Would verify:   %s\n' "$CHECKSUM_URL"
    else
        download "$DOWNLOAD_URL" "$SOURCE_PATH" || die "download failed: $DOWNLOAD_URL"
        download "$CHECKSUM_URL" "$CHECKSUM_PATH" || die "checksum download failed: $CHECKSUM_URL"
        verify_checksum "$SOURCE_PATH" "$CHECKSUM_PATH"
    fi
else
    SOURCE_PATH="release/$BIN_FILE"
    CHECKSUM_PATH="$SOURCE_PATH.sha256"
    [ -f "$SOURCE_PATH" ] || die "local release not found: $SOURCE_PATH"
    [ -f "$CHECKSUM_PATH" ] || die "local checksum not found: $CHECKSUM_PATH"
    [ "$DRY_RUN" -eq 1 ] || verify_checksum "$SOURCE_PATH" "$CHECKSUM_PATH"
fi

if [ -z "$DEST" ]; then
    if [ "$OS" = windows ]; then
        DEST=${HOME:-.}/bin
    elif [ -w /usr/local/bin ]; then
        DEST=/usr/local/bin
    else
        DEST=${XDG_BIN_HOME:-${HOME:-.}/.local/bin}
    fi
fi

TARGET="$DEST/fz"
[ "$OS" = windows ] && TARGET="$TARGET.exe"

if [ -e "$TARGET" ] && [ "$FORCE" -eq 0 ]; then
    die "target exists: $TARGET (use --force to replace it)"
fi

if [ "$DRY_RUN" -eq 1 ]; then
    printf 'Would install: %s -> %s\n' "$SOURCE_PATH" "$TARGET"
    exit 0
fi

if [ ! -e "$DEST" ]; then
    mkdir -p "$DEST" 2>/dev/null || :
fi
if [ -d "$DEST" ] && [ -w "$DEST" ]; then
    INSTALL_TMP="$DEST/.fz.$$.tmp"
    cp "$SOURCE_PATH" "$INSTALL_TMP"
    chmod 0755 "$INSTALL_TMP"
    mv -f "$INSTALL_TMP" "$TARGET"
else
    command -v sudo >/dev/null 2>&1 || die "cannot write $DEST and sudo is unavailable"
    sudo mkdir -p "$DEST"
    sudo install -m 0755 "$SOURCE_PATH" "$TARGET"
fi

case ":${PATH:-}:" in
    *:"$DEST":*) ;;
    *) printf 'Add %s to PATH\n' "$DEST" ;;
esac
printf 'Installed ForgeZero: %s\n' "$TARGET"
