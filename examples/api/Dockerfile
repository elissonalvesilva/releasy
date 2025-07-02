# ------------------------------------------
# 🚀 Stage 1 — Build Go binary
# ------------------------------------------
FROM golang:1.23.10-alpine3.22 AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o api .

# ------------------------------------------
# 🚀 Stage 2 — Image final minimalista
# ------------------------------------------
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/api .

EXPOSE 8080

ENTRYPOINT ["./api"]
