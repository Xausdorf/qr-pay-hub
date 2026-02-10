package pay

import (
	"context"

	"github.com/google/uuid"

	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/domain/payment"
)

type Request struct {
	IdempotencyKey string
	FromID         string
	ToID           string
	Amount         int64
}

type Response struct {
	TransactionID string
	Status        string
	Error         string
}

type UseCase struct {
	client payment.Client
}

func NewUseCase(client payment.Client) *UseCase {
	return &UseCase{client: client}
}

func (uc *UseCase) Execute(ctx context.Context, req Request) (*Response, error) {
	fromID, err := uuid.Parse(req.FromID)
	if err != nil {
		return nil, err
	}

	toID, err := uuid.Parse(req.ToID)
	if err != nil {
		return nil, err
	}

	resp, err := uc.client.ProcessPayment(ctx, payment.Request{
		IdempotencyKey: req.IdempotencyKey,
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         req.Amount,
	})
	if err != nil {
		return nil, err
	}

	return &Response{
		TransactionID: resp.TransactionID,
		Status:        resp.Status,
		Error:         resp.ErrorMessage,
	}, nil
}
