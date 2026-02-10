package repository

import "context"

type UnitOfWork interface {
	Begin(ctx context.Context) (UnitOfWork, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	Accounts() AccountRepository
	Transactions() TransactionRepository
	Idempotency() IdempotencyRepository
}
