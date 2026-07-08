#!/bin/bash
# Copyright (c) 2026 ForgeZero-cli
# SPDX-License-Identifier: GPL-3.0-or-later
set -e

HOST_GOOS=$(go env GOOS)
HOST_GOARCH=$(go env GOARCH)

run_tests() {
  local target_goos="$1"
  local target_goarch="$2"
  local exec_cmd="/bin/true"

  echo "Testing GOOS=$target_goos GOARCH=$target_goarch"
  if [ "$target_goos" = "$HOST_GOOS" ] && [ "$target_goarch" = "$HOST_GOARCH" ]; then
    env GOOS="$target_goos" GOARCH="$target_goarch" go test -race -benchmem ./...
  else
    env GOOS="$target_goos" GOARCH="$target_goarch" go test -run '^$' -exec "$exec_cmd" ./...
  fi
}

for GOOS in linux windows darwin; do
  for GOARCH in amd64 arm64; do
    if [ "$GOOS" = "windows" ] && [ "$GOARCH" = "arm64" ]; then
      continue
    fi
    run_tests "$GOOS" "$GOARCH"
  done
done