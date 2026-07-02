#!/usr/bin/env bash
set -euo pipefail
DEST=""
OS=""
ARCH=""
VER="latest"
FORCE=0
REPO="forgezero-cli/ForgeZero"
DRY=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    -d|--dest) DEST="$2"; shift 2;;
    -o|--os) OS="$2"; shift 2;;
    -a|--arch) ARCH="$2"; shift 2;;
    -v|--version) VER="$2"; shift 2;;
    -r|--repo) REPO="$2"; shift 2;;
    -f|--force) FORCE=1; shift 1;;
    --dry-run) DRY=1; shift 1;;
    --) shift; break;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done
uname_s=$(uname -s 2>/dev/null || echo unknown)
uname_m=$(uname -m 2>/dev/null || echo unknown)
if [ -z "$OS" ]; then
  case "$uname_s" in
    Darwin*) OS=darwin;;
    Linux*) OS=linux;;
    MINGW*|MSYS*|CYGWIN*) OS=windows;;
    *) OS=linux;;
  esac
fi
if [ -z "$ARCH" ]; then
  case "$uname_m" in
    x86_64|amd64) ARCH=amd64;;
    aarch64|arm64) ARCH=arm64;;
    armv7l|armv6l) ARCH=arm;;
    *) ARCH=amd64;;
  esac
fi
BIN_NAME="fz"
if [ "$OS" = "windows" ]; then
  BIN_FILE="${BIN_NAME}-${OS}-${ARCH}.exe"
else
  BIN_FILE="${BIN_NAME}-${OS}-${ARCH}"
fi
SRC_PATH=""
if [ -n "$REPO" ]; then
  if [ "$VER" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BIN_FILE}"
  else
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VER}/${BIN_FILE}"
  fi
  TMPF=$(mktemp -u)
  if command -v curl >/dev/null 2>&1; then
    DL_CMD="curl -fL -o \"${TMPF}\" \"${DOWNLOAD_URL}\""
  elif command -v wget >/dev/null 2>&1; then
    DL_CMD="wget -O \"${TMPF}\" \"${DOWNLOAD_URL}\""
  else
    echo "curl or wget required" >&2
    exit 1
  fi
  if [ "$DRY" -eq 1 ]; then
    echo "Would download: ${DOWNLOAD_URL} -> ${TMPF}"
    exit 0
  fi
  eval $DL_CMD
  SRC_PATH="$TMPF"
else
  REL_PATH="release/${BIN_FILE}"
  if [ -f "$REL_PATH" ]; then
    SRC_PATH="$REL_PATH"
  else
    echo "Local release not found: $REL_PATH" >&2
    exit 1
  fi
fi
if [ ! -f "$SRC_PATH" ]; then
  echo "Download failed or source missing: $SRC_PATH" >&2
  exit 1
fi
if [ "$FORCE" -eq 0 ]; then
  if command -v sha256sum >/dev/null 2>&1; then
    if [ -f "release/${BIN_FILE}.sha256" ]; then
      exp=$(cut -d' ' -f1 "release/${BIN_FILE}.sha256")
      got=$(sha256sum "$SRC_PATH" | cut -d' ' -f1)
      if [ "$exp" != "$got" ]; then
        echo "checksum mismatch" >&2
        exit 1
      fi
    fi
  fi
fi
if [ -z "$DEST" ]; then
  if [ "$OS" = "windows" ]; then
    DEST="$HOME/bin"
  else
    if [ -w "/usr/local/bin" ]; then
      DEST="/usr/local/bin"
    else
      DEST="$HOME/.local/bin"
    fi
  fi
fi
mkdir -p "$DEST"
TARGET_PATH="$DEST/$BIN_NAME"
if [ "$OS" = "windows" ]; then
  TARGET_PATH="$DEST/${BIN_NAME}.exe"
fi
if [ -f "$TARGET_PATH" ] && [ "$FORCE" -eq 0 ]; then
  echo "Target exists: $TARGET_PATH (use --force to overwrite)" >&2
  exit 1
fi
if [ "$DRY" -eq 1 ]; then
  echo "Would install $SRC_PATH -> $TARGET_PATH"
  exit 0
fi
if [ -w "$DEST" ]; then
  mv -f "$SRC_PATH" "$TARGET_PATH"
else
  if command -v sudo >/dev/null 2>&1; then
    sudo mv -f "$SRC_PATH" "$TARGET_PATH"
  else
    echo "Need sudo to move to $DEST" >&2
    exit 1
  fi
fi
chmod +x "$TARGET_PATH"
if [ "$DEST" = "$HOME/.local/bin" ] || [ "$DEST" = "$HOME/bin" ]; then
  case ":$PATH:" in
    *":$DEST:"*) :;;
    *) echo "Add $DEST to PATH";;
  esac
fi
echo "Installed: $TARGET_PATH"