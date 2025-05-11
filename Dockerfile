# Этап сборки (builder)
FROM golang:1.23.1-alpine AS builder

# Установка зависимостей для сборки
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Копируем модули для кэширования
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем статический бинарник с оптимизацией
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o /app/bin/grpc-service ./cmd

# Этап запуска (минимальный образ)
FROM alpine:3.18

# Аргумент для переменной окружения (можно переопределить при сборке)
ARG PORT=50051

# Переменная окружения для порта (можно переопределить при запуске)
ENV GRPC_PORT=$PORT

# Сертификаты и непривилегированный пользователь
RUN apk --no-cache add ca-certificates && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup

WORKDIR /app

# Копируем бинарник из builder
COPY --from=builder --chown=appuser:appgroup /app/bin/grpc-service .
COPY --from=builder --chown=appuser:appgroup /app/.env . 

# Настройки безопасности
USER appuser

# Экспортируем порт из переменной окружения и запускаем
EXPOSE $GRPC_PORT
CMD ["./grpc-service"]