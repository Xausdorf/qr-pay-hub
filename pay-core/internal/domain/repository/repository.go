package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/Xausdorf/qr-pay-hub/internal/domain/entity"
)

type AccountRepository interface {
	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*entity.Account, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, newBalance int64) error
}

type TransactionRepository interface {
	Create(ctx context.Context, tx *entity.Transaction) error
}

type IdempotencyRepository interface {
	Find(ctx context.Context, key string) (*entity.IdempotencyRecord, error)
	Save(ctx context.Context, record *entity.IdempotencyRecord) error
	Lock(ctx context.Context, key string) error
}
