package transfer_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/Xausdorf/qr-pay-hub/internal/domain/entity"
	"github.com/Xausdorf/qr-pay-hub/internal/usecase/transfer"
	"github.com/Xausdorf/qr-pay-hub/internal/usecase/transfer/mocks"
)

func TestTransferUseCase_Execute_Idempotency(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uow := mocks.NewMockUnitOfWork(ctrl)
	idempotencyRepo := mocks.NewMockIdempotencyRepository(ctrl)

	uc := transfer.NewUseCase(uow)

	cachedBody := []byte(`{"transaction_id":"cached-tx-id","status":"success","error_message":""}`)
	record := entity.ReconstructIdempotencyRecord("test-key", 2, cachedBody, time.Time{})

	uow.EXPECT().Idempotency().Return(idempotencyRepo)
	idempotencyRepo.EXPECT().Find(gomock.Any(), "test-key").Return(record, nil)

	resp, err := uc.Execute(context.Background(), transfer.Request{
		IdempotencyKey: "test-key",
		FromAccountID:  uuid.New(),
		ToAccountID:    uuid.New(),
		Amount:         1000,
	})

	require.NoError(t, err)
	assert.Equal(t, "cached-tx-id", resp.TransactionID)
	assert.Equal(t, entity.StatusSuccess, resp.Status)
}

func TestTransferUseCase_Execute_SuccessfulTransfer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uow := mocks.NewMockUnitOfWork(ctrl)
	txUow := mocks.NewMockUnitOfWork(ctrl)
	accountRepo := mocks.NewMockAccountRepository(ctrl)
	txnRepo := mocks.NewMockTransactionRepository(ctrl)
	idempotencyRepo := mocks.NewMockIdempotencyRepository(ctrl)

	uc := transfer.NewUseCase(uow)

	fromID := uuid.New()
	toID := uuid.New()

	uow.EXPECT().Idempotency().Return(idempotencyRepo)
	idempotencyRepo.EXPECT().Find(gomock.Any(), "new-key").Return(nil, nil)

	uow.EXPECT().Begin(gomock.Any()).Return(txUow, nil)
	txUow.EXPECT().Rollback(gomock.Any()).Return(nil)

	txUow.EXPECT().Idempotency().Return(idempotencyRepo).Times(3)
	idempotencyRepo.EXPECT().Lock(gomock.Any(), "new-key").Return(nil)
	idempotencyRepo.EXPECT().Find(gomock.Any(), "new-key").Return(nil, nil)

	txUow.EXPECT().Accounts().Return(accountRepo).Times(4)
	txUow.EXPECT().Transactions().Return(txnRepo)
	txUow.EXPECT().Commit(gomock.Any()).Return(nil)

	accountRepo.EXPECT().FindByIDForUpdate(gomock.Any(), fromID).Return(entity.NewAccount(fromID, 5000), nil)
	accountRepo.EXPECT().FindByIDForUpdate(gomock.Any(), toID).Return(entity.NewAccount(toID, 1000), nil)
	accountRepo.EXPECT().UpdateBalance(gomock.Any(), fromID, int64(4000)).Return(nil)
	accountRepo.EXPECT().UpdateBalance(gomock.Any(), toID, int64(2000)).Return(nil)
	txnRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
	idempotencyRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	resp, err := uc.Execute(context.Background(), transfer.Request{
		IdempotencyKey: "new-key",
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         1000,
	})

	require.NoError(t, err)
	assert.Equal(t, entity.StatusSuccess, resp.Status)
}

func TestTransferUseCase_Execute_InsufficientFunds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uow := mocks.NewMockUnitOfWork(ctrl)
	txUow := mocks.NewMockUnitOfWork(ctrl)
	accountRepo := mocks.NewMockAccountRepository(ctrl)
	idempotencyRepo := mocks.NewMockIdempotencyRepository(ctrl)

	uc := transfer.NewUseCase(uow)

	fromID := uuid.New()
	toID := uuid.New()

	uow.EXPECT().Idempotency().Return(idempotencyRepo)
	idempotencyRepo.EXPECT().Find(gomock.Any(), "insufficient-key").Return(nil, nil)

	uow.EXPECT().Begin(gomock.Any()).Return(txUow, nil)
	txUow.EXPECT().Rollback(gomock.Any()).Return(nil)

	txUow.EXPECT().Idempotency().Return(idempotencyRepo).Times(3)
	idempotencyRepo.EXPECT().Lock(gomock.Any(), "insufficient-key").Return(nil)
	idempotencyRepo.EXPECT().Find(gomock.Any(), "insufficient-key").Return(nil, nil)

	txUow.EXPECT().Accounts().Return(accountRepo)
	txUow.EXPECT().Commit(gomock.Any()).Return(nil)

	accountRepo.EXPECT().FindByIDForUpdate(gomock.Any(), fromID).Return(entity.NewAccount(fromID, 500), nil)
	idempotencyRepo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)

	resp, err := uc.Execute(context.Background(), transfer.Request{
		IdempotencyKey: "insufficient-key",
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         1000,
	})

	require.NoError(t, err)
	assert.Equal(t, entity.StatusFailed, resp.Status)
	assert.Equal(t, "insufficient funds", resp.ErrorMessage)
}

func TestTransferUseCase_Execute_SenderNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	uow := mocks.NewMockUnitOfWork(ctrl)
	txUow := mocks.NewMockUnitOfWork(ctrl)
	accountRepo := mocks.NewMockAccountRepository(ctrl)
	idempotencyRepo := mocks.NewMockIdempotencyRepository(ctrl)

	uc := transfer.NewUseCase(uow)

	fromID := uuid.New()
	toID := uuid.New()

	uow.EXPECT().Idempotency().Return(idempotencyRepo)
	idempotencyRepo.EXPECT().Find(gomock.Any(), "notfound-key").Return(nil, nil)

	uow.EXPECT().Begin(gomock.Any()).Return(txUow, nil)
	txUow.EXPECT().Rollback(gomock.Any()).Return(nil)

	txUow.EXPECT().Idempotency().Return(idempotencyRepo).Times(2)
	idempotencyRepo.EXPECT().Lock(gomock.Any(), "notfound-key").Return(nil)
	idempotencyRepo.EXPECT().Find(gomock.Any(), "notfound-key").Return(nil, nil)

	txUow.EXPECT().Accounts().Return(accountRepo)

	accountRepo.EXPECT().FindByIDForUpdate(gomock.Any(), fromID).Return(nil, errors.New("account not found"))

	_, err := uc.Execute(context.Background(), transfer.Request{
		IdempotencyKey: "notfound-key",
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         1000,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "account not found")
}
