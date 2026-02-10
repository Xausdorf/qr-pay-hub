# QR-Pay-Hub

Микросервисная платёжная система с поддержкой QR-переводов.

## Архитектура

```
┌─────────────┐     HTTP      ┌─────────────┐     gRPC      ┌─────────────┐
│   Client    │──────────────►│ pay-gateway │──────────────►│  pay-core   │
└─────────────┘               └─────────────┘               └──────┬──────┘
                                                                   │
                                                                   ▼
                                                            ┌─────────────┐
                                                            │ PostgreSQL  │
                                                            └─────────────┘
```

## Clean Architecture

Оба сервиса следуют принципам Clean Architecture:

```
cmd/server/main.go          # Точка входа, DI-контейнер
internal/
├── domain/                 # Слой домена (сущности, интерфейсы)
│   ├── entity/
│   └── repository/
├── usecase/                # Слой бизнес-логики
├── infrastructure/         # Слой инфраструктуры (БД, клиенты)
│   ├── postgres/
│   └── config/
└── delivery/               # Слой доставки (gRPC/HTTP)
    └── grpc/ или http/
```

## Сервисы

| Сервис | Описание | Порт |
|--------|----------|------|
| **pay-core** | gRPC сервис обработки платежей | 50051 |
| **pay-gateway** | HTTP API + генерация QR-кодов | 8080 |

## Стек технологий

- **Go 1.25**
- **gRPC** (google.golang.org/grpc)
- **PostgreSQL 18** (pgx/v5)
- **Docker Compose**
- **chi** (HTTP роутер)

## Быстрый старт

```bash
# 1. Запуск PostgreSQL
docker compose up -d

# 2. Запуск Core сервиса
cd pay-core && go run ./cmd/server

# 3. Запуск Gateway (в другом терминале)
cd pay-gateway && go run ./cmd/server
```

## API

### POST /api/pay
Выполнить платёж.

```bash
curl -X POST http://localhost:8080/api/pay \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: unique-key-123" \
  -d '{"from_id": "uuid", "to_id": "uuid", "amount": 1000}'
```

### GET /api/qr/{account_id}?amount=100
Сгенерировать QR-код для платежа.

```bash
curl http://localhost:8080/api/qr/550e8400-e29b-41d4-a716-446655440000?amount=1000 -o qr.png
```

## Ключевые особенности

- **Clean Architecture** — строгое разделение слоёв
- **Идемпотентность** — гарантия обработки запроса ровно 1 раз
- **Pessimistic Locking** — защита от double-spending через `SELECT ... FOR UPDATE`
- **UnitOfWork** — атомарные транзакции
