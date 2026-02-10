# Pay-Core

gRPC сервис обработки платежей — ядро платёжной системы QR-Pay-Hub.

## Архитектура (Clean Architecture)

```
pay-core/
├── cmd/server/main.go                     # Точка входа, DI
├── gen/pb/                                # Сгенерированный protobuf
└── internal/
    ├── domain/                            # СЛОЙ ДОМЕНА
    │   ├── entity/
    │   │   ├── account.go                 # Account entity
    │   │   ├── transaction.go             # Transaction entity
    │   │   └── idempotency.go             # IdempotencyRecord entity
    │   └── repository/
    │       ├── repository.go              # Repository интерфейсы
    │       └── unit_of_work.go            # UnitOfWork интерфейс
    │
    ├── usecase/                           # СЛОЙ USE CASES
    │   └── transfer/
    │       └── transfer.go                # TransferUseCase
    │
    ├── infrastructure/                    # СЛОЙ ИНФРАСТРУКТУРЫ
    │   ├── postgres/
    │   │   └── repositories.go            # PostgreSQL реализации
    │   └── config/
    │       └── config.go                  # Конфигурация
    │
    └── delivery/                          # СЛОЙ ДОСТАВКИ
        └── grpc/
            └── handler.go                 # gRPC хендлер
```

## Запуск

```bash
go run ./cmd/server
```

## Конфигурация

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `DATABASE_URL` | `postgres://qrpay:qrpay_secret@localhost:5432/qrpay?sslmode=disable` | Строка подключения |
| `GRPC_ADDR` | `:50051` | Адрес gRPC сервера |

## gRPC API

### PaymentProcessor.ProcessPayment

```protobuf
message PaymentRequest {
  string idempotency_key = 1;
  string from_account_id = 2;
  string to_account_id = 3;
  int64 amount = 4;
}
```

## Логика TransferUseCase

1. Проверка идемпотентности
2. Начало UnitOfWork
3. Блокировка отправителя (`SELECT ... FOR UPDATE`)
4. `Account.Debit()` — проверка и списание
5. Блокировка получателя
6. `Account.Credit()` — зачисление
7. Создание Transaction entity
8. Сохранение IdempotencyRecord
9. Commit
