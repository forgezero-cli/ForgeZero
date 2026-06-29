#!/bin/bash

go build -ldflags="-X github.com/forgezero-cli/ForgeZero/cmd/fz/cli.BuildDate=$(date +%Y-%m-%d)" -o fz cmd/fz/main.go
