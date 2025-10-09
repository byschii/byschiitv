
# Multi-stage Dockerfile for iptvsim
#  - builder: uses the official golang image to compile the binary
#  - final: small alpine image that runs the compiled binary

FROM golang:1.23-alpine AS builder

WORKDIR /src

# Cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the project and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o /iptvsim main.go

FROM alpine:3.18

# non-root user
RUN addgroup -S app && adduser -S -G app app

COPY --from=builder /iptvsim /usr/local/bin/iptvsim
RUN apk add --no-cache ffmpeg
EXPOSE 8080
USER app

ENTRYPOINT ["/usr/local/bin/iptvsim"]

