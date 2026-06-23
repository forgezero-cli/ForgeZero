FROM golang:1.24-alpine AS builder
ARG VERSION
ARG BUILD_DATE
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -buildvcs=false \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildDate=${BUILD_DATE}" \
    -o fz ./cmd/fz

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/fz .
ENTRYPOINT ["./fz"]
