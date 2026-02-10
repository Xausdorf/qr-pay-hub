package transfer

import (
	"context"
	"encoding/json"
	"errors"

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
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
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

	if lockErr := tx.Idempotency().Lock(ctx, req.IdempotencyKey); lockErr != nil {
		return nil, lockErr
	}

	cached, err = tx.Idempotency().Find(ctx, req.IdempotencyKey)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if cached != nil {
		return uc.parseCache(cached.ResponseBody())
	}

	sender, err := tx.Accounts().FindByIDForUpdate(ctx, req.FromAccountID)
	if err != nil {
		return nil, err
	}

	if debitErr := sender.Debit(req.Amount); debitErr != nil {
		return uc.saveAndReturn(ctx, tx, req.IdempotencyKey, uuid.Nil, entity.StatusFailed, debitErr.Error())
	}

	receiver, err := tx.Accounts().FindByIDForUpdate(ctx, req.ToAccountID)
	if err != nil {
		return nil, err
	}

	if creditErr := receiver.Credit(req.Amount); creditErr != nil {
		return nil, creditErr
	}

	if updErr := tx.Accounts().UpdateBalance(ctx, sender.ID(), sender.Balance()); updErr != nil {
		return nil, updErr
	}

	if updErr := tx.Accounts().UpdateBalance(ctx, receiver.ID(), receiver.Balance()); updErr != nil {
		return nil, updErr
	}

	txn := entity.NewTransaction(req.FromAccountID, req.ToAccountID, req.Amount, entity.StatusSuccess)
	if createErr := tx.Transactions().Create(ctx, txn); createErr != nil {
		return nil, createErr
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

	record := entity.NewIdempotencyRecord(key, statusToCode(status), body)
	if saveErr := tx.Idempotency().Save(ctx, record); saveErr != nil {
		return nil, saveErr
	}

	if commitErr := tx.Commit(ctx); commitErr != nil {
		return nil, commitErr
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
