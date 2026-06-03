FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -ldflags="-s -w" -o fz ./cmd/fz

FROM alpine:latest

RUN apk add --no-cache \
    nasm \
    gcc \
    clang \
    zig \
    musl-dev \
    linux-headers

WORKDIR /app

COPY --from=builder /build/fz /usr/local/bin/fz

ENTRYPOINT ["fz"]
CMD ["--help"]
