# ── Build stage ────────────────────────────────────────────────────────────────
FROM golang:1-alpine AS builder

ENV GOTOOLCHAIN=auto

ARG VERSION=dev

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /bin/scaloo \
    ./cmd

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S scaloo && adduser -S scaloo -G scaloo

COPY --from=builder /bin/scaloo /usr/local/bin/scaloo

USER scaloo

EXPOSE 8080

ENTRYPOINT ["scaloo"]
