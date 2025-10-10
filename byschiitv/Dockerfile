# ---- build the Go binary (still fine to use Debian) ----
FROM golang:1.23-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# static binary, arm64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" -o /iptvsim .

# ---- runtime: Debian + Raspberry Pi ffmpeg ----
FROM debian:bookworm-slim

# Add Raspberry Pi repo and install their ffmpeg + v4l-utils
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates wget gnupg && \
    wget -qO /usr/share/keyrings/raspberrypi-archive-keyring.gpg \
      https://archive.raspberrypi.com/debian/raspberrypi.gpg.key && \
    echo "deb [arch=arm64 signed-by=/usr/share/keyrings/raspberrypi-archive-keyring.gpg] \
      http://archive.raspberrypi.com/debian/ bookworm main" \
      > /etc/apt/sources.list.d/raspi.list && \
    apt-get update && apt-get install -y --no-install-recommends \
      ffmpeg v4l-utils && \
    rm -rf /var/lib/apt/lists/*

# Create an app user that can access /dev/video* (match host 'video' GID)
ARG VIDEO_GID=44
RUN groupadd -g ${VIDEO_GID} video || true && \
    useradd -m -s /bin/bash -G video app

COPY --from=builder /iptvsim /usr/local/bin/iptvsim
EXPOSE 8080
USER app
ENTRYPOINT ["/usr/local/bin/iptvsim"]
