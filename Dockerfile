# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем статический бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /sprut ./cmd/sprut

# Runtime stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates

COPY --from=builder /sprut /usr/local/bin/sprut

# Создаём непривилегированного пользователя
RUN adduser -D -H -s /sbin/nologin sprut
USER sprut

EXPOSE 8443

ENTRYPOINT ["sprut"]
CMD ["-config", "/etc/sprut/config.yaml"]
