package transfer

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/Xausdorf/qr-pay-hub/internal/domain/entity"
	"github.com/Xausdorf/qr-pay-hub/internal/domain/repository"
)

type Request struct {
	IdempotencyKey string
	FromAccountID  uuid.UUID
	ToAccountID    uuid.UUID
	Amount         int64
}

type Response struct {
	TransactionID string
	Status        entity.TransactionStatus
	ErrorMessage  string
}

type responseCache struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	ErrorMessage  string `json:"error_message"`
}

type UseCase struct {
	uow repository.UnitOfWork
}

func NewUseCase(uow repository.UnitOfWork) *UseCase {
	return &UseCase{uow: uow}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) (*Response, error) {
	cached, err := uc.uow.Idempotency().Find(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return uc.parseCache(cached.ResponseBody())
	}

	tx, err := uc.uow.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := tx.Idempotency().Lock(ctx, req.IdempotencyKey); err != nil {
		return nil, err
	}

	cached, err = tx.Idempotency().Find(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return uc.parseCache(cached.ResponseBody())
	}

	sender, err := tx.Accounts().FindByIDForUpdate(ctx, req.FromAccountID)
	if err != nil {
		return nil, err
	}

	if err := sender.Debit(req.Amount); err != nil {
		return uc.saveAndReturn(ctx, tx, req.IdempotencyKey, uuid.Nil, entity.StatusFailed, err.Error())
	}

	receiver, err := tx.Accounts().FindByIDForUpdate(ctx, req.ToAccountID)
	if err != nil {
		return nil, err
	}

	if err := receiver.Credit(req.Amount); err != nil {
		return nil, err
	}

	if err := tx.Accounts().UpdateBalance(ctx, sender.ID(), sender.Balance()); err != nil {
		return nil, err
	}

	if err := tx.Accounts().UpdateBalance(ctx, receiver.ID(), receiver.Balance()); err != nil {
		return nil, err
	}

	txn := entity.NewTransaction(req.FromAccountID, req.ToAccountID, req.Amount, entity.StatusSuccess)
	if err := tx.Transactions().Create(ctx, txn); err != nil {
		return nil, err
	}

	return uc.saveAndReturn(ctx, tx, req.IdempotencyKey, txn.ID(), entity.StatusSuccess, "")
}

func (uc *UseCase) saveAndReturn(
	ctx context.Context,
	tx repository.UnitOfWork,
	key string,
	txID uuid.UUID,
	status entity.TransactionStatus,
	errMsg string,
) (*Response, error) {
	cache := responseCache{
		TransactionID: txID.String(),
		Status:        string(status),
		ErrorMessage:  errMsg,
	}
	body, err := json.Marshal(cache)
	if err != nil {
		return nil, err
	}

	record := entity.NewIdempotencyRecord(key, int(statusToCode(status)), body)
	if err := tx.Idempotency().Save(ctx, record); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &Response{
		TransactionID: txID.String(),
		Status:        status,
		ErrorMessage:  errMsg,
	}, nil
}

func (uc *UseCase) parseCache(body []byte) (*Response, error) {
	var cache responseCache
	if err := json.Unmarshal(body, &cache); err != nil {
		return nil, err
	}
	return &Response{
		TransactionID: cache.TransactionID,
		Status:        entity.TransactionStatus(cache.Status),
		ErrorMessage:  cache.ErrorMessage,
	}, nil
}

const (
	statusCodeUnspecified = 0
	statusCodePending     = 1
	statusCodeSuccess     = 2
	statusCodeFailed      = 3
)

func statusToCode(s entity.TransactionStatus) int {
	switch s {
	case entity.StatusPending:
		return statusCodePending
	case entity.StatusSuccess:
		return statusCodeSuccess
	case entity.StatusFailed:
		return statusCodeFailed
	default:
		return statusCodeUnspecified
	}
}
