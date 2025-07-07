# ------------------------------------------
# 🚀 Stage 1 — Build Go binary
# ------------------------------------------
FROM golang:1.23.10-alpine3.22 AS builder

# Instala git e outras dependências básicas
RUN apk add --no-cache git

WORKDIR /app

# Copia go.mod + go.sum e faz download das deps
COPY go.mod go.sum ./
RUN go mod download

# Copia tudo
COPY . .

# Compila o binário
RUN go build -o controlplane ./cmd/controlplane

# ------------------------------------------
# 🚀 Stage 2 — Image final minimalista
# ------------------------------------------
FROM alpine:latest

WORKDIR /app

# Copia binário do builder
COPY --from=builder /app/controlplane .

# Expõe porta (opcional, se tiver porta http)
EXPOSE 3344


# Comando de inicialização
ENTRYPOINT ["./controlplane"]
