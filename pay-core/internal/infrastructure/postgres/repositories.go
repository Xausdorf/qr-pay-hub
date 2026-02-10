package postgres

import (
	"context"
	"errors"
	"hash/fnv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Xausdorf/qr-pay-hub/internal/domain/entity"
	"github.com/Xausdorf/qr-pay-hub/internal/domain/repository"
)

type UnitOfWork struct {
	pool *pgxpool.Pool
	tx   pgx.Tx
}

func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

func (u *UnitOfWork) Begin(ctx context.Context) (repository.UnitOfWork, error) {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &UnitOfWork{pool: u.pool, tx: tx}, nil
}

func (u *UnitOfWork) Commit(ctx context.Context) error {
	if u.tx == nil {
		return nil
	}
	return u.tx.Commit(ctx)
}

func (u *UnitOfWork) Rollback(ctx context.Context) error {
	if u.tx == nil {
		return nil
	}
	return u.tx.Rollback(ctx)
}

func (u *UnitOfWork) Accounts() repository.AccountRepository {
	return &AccountRepo{tx: u.tx, pool: u.pool}
}

func (u *UnitOfWork) Transactions() repository.TransactionRepository {
	return &TransactionRepo{tx: u.tx}
}

func (u *UnitOfWork) Idempotency() repository.IdempotencyRepository {
	return &IdempotencyRepo{tx: u.tx, pool: u.pool}
}

type AccountRepo struct {
	tx   pgx.Tx
	pool *pgxpool.Pool
}

func (r *AccountRepo) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	var balance int64
	err := r.tx.QueryRow(ctx,
		`SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`,
		id,
	).Scan(&balance)
	if err != nil {
		return nil, err
	}
	return entity.NewAccount(id, balance), nil
}

func (r *AccountRepo) UpdateBalance(ctx context.Context, id uuid.UUID, newBalance int64) error {
	_, err := r.tx.Exec(ctx,
		`UPDATE accounts SET balance = $1 WHERE id = $2`,
		newBalance, id,
	)
	return err
}

type TransactionRepo struct {
	tx pgx.Tx
}

func (r *TransactionRepo) Create(ctx context.Context, t *entity.Transaction) error {
	_, err := r.tx.Exec(ctx,
		`INSERT INTO transactions (id, from_account, to_account, amount, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID(), t.FromAccount(), t.ToAccount(), t.Amount(), string(t.Status()), t.CreatedAt(),
	)
	return err
}

type IdempotencyRepo struct {
	tx   pgx.Tx
	pool *pgxpool.Pool
}

func (r *IdempotencyRepo) Find(ctx context.Context, key string) (*entity.IdempotencyRecord, error) {
	q := r.pool
	if r.tx != nil {
		return r.findWithTx(ctx, key)
	}

	var code int
	var body []byte
	err := q.QueryRow(ctx,
		`SELECT response_code, response_body FROM idempotency_keys WHERE key = $1`,
		key,
	).Scan(&code, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return entity.ReconstructIdempotencyRecord(key, code, body, time.Time{}), nil
}

func (r *IdempotencyRepo) findWithTx(ctx context.Context, key string) (*entity.IdempotencyRecord, error) {
	var code int
	var body []byte
	err := r.tx.QueryRow(ctx,
		`SELECT response_code, response_body FROM idempotency_keys WHERE key = $1`,
		key,
	).Scan(&code, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return entity.ReconstructIdempotencyRecord(key, code, body, time.Time{}), nil
}

func (r *IdempotencyRepo) Save(ctx context.Context, record *entity.IdempotencyRecord) error {
	_, err := r.tx.Exec(ctx,
		`INSERT INTO idempotency_keys (key, response_code, response_body, created_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (key) DO NOTHING`,
		record.Key(), record.ResponseCode(), record.ResponseBody(), record.CreatedAt(),
	)
	return err
}

func (r *IdempotencyRepo) Lock(ctx context.Context, key string) error {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	_, err := r.tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(h.Sum64()))
	return err
}
