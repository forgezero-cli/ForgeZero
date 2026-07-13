#!/bin/bash
# Copyright (c) 2026 ForgeZero-cli
# SPDX-License-Identifier: GPL-3.0-or-later
set -euo pipefail

export GOMAXPROCS=14
export GOGC=150

HOST_GOOS=$(go env GOOS)
HOST_GOARCH=$(go env GOARCH)

run_tests() {
    local target_goos="$1"
    local target_goarch="$2"

    echo "[*] Testing $target_goos/$target_goarch"

    if [ "$target_goos" = "$HOST_GOOS" ] && [ "$target_goarch" = "$HOST_GOARCH" ]; then
        go test -race -benchmem -p 8 -parallel 8 ./...
    else
        go test -run '^$' -exec /bin/true ./...
    fi
}

for GOOS in linux windows darwin; do
    for GOARCH in amd64 arm64; do
        [[ "$GOOS" == "windows" && "$GOARCH" == "arm64" ]] && continue
        run_tests "$GOOS" "$GOARCH" &
    done
done

wait