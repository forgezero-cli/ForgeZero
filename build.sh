#!/bin/bash

go build -ldflags="-X github.com/forgezero-cli/ForgeZero/cmd/fz/cli.BuildDate=$(date +%Y-%m-%d) -X github.com/forgezero-cli/ForgeZero/cmd/fz/cli.VersionCore=v5.3.0" -o fz cmd/fz/main.go
