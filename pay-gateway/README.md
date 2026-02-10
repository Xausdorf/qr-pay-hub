# Pay-Gateway

HTTP API шлюз для платёжной системы QR-Pay-Hub.

## Архитектура (Clean Architecture)

```
pay-gateway/
├── cmd/server/main.go                    # Точка входа, DI
├── gen/pb/                               # Сгенерированный protobuf
└── internal/
    ├── domain/                           # СЛОЙ ДОМЕНА
    │   ├── payment/
    │   │   └── payment.go                # Payment типы и Client интерфейс
    │   └── qrcode/
    │       └── qrcode.go                 # QRData и Generator интерфейс
    │
    ├── usecase/                          # СЛОЙ USE CASES
    │   ├── pay/
    │   │   └── pay.go                    # PayUseCase
    │   └── generateqr/
    │       └── generateqr.go             # GenerateQRUseCase
    │
    ├── infrastructure/                   # СЛОЙ ИНФРАСТРУКТУРЫ
    │   ├── grpcclient/
    │   │   └── client.go                 # gRPC клиент к pay-core
    │   ├── qrgenerator/
    │   │   └── generator.go              # QR генератор (skip2/go-qrcode)
    │   └── config/
    │       └── config.go                 # Конфигурация
    │
    └── delivery/                         # СЛОЙ ДОСТАВКИ
        └── http/
            ├── handler.go                # HTTP хендлеры
            └── router.go                 # Chi роутер
```

## Запуск

```bash
go run ./cmd/server
```

## Конфигурация

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `CORE_GRPC_ADDR` | `localhost:50051` | Адрес gRPC сервиса pay-core |
| `HTTP_ADDR` | `:8080` | Адрес HTTP сервера |

## HTTP API

### POST /api/pay

```bash
curl -X POST http://localhost:8080/api/pay \
  -H "Content-Type: application/json" \
  -H "X-Idempotency-Key: payment-123" \
  -d '{"from_id": "uuid1", "to_id": "uuid2", "amount": 500}'
```

### GET /api/qr/{account_id}?amount=1000

```bash
curl "http://localhost:8080/api/qr/550e8400-e29b-41d4-a716-446655440000?amount=1000" -o qr.png
```
