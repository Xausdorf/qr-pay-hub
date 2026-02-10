package payment

import (
	"context"

	"github.com/google/uuid"
)

type Request struct {
	IdempotencyKey string
	FromAccountID  uuid.UUID
	ToAccountID    uuid.UUID
	Amount         int64
}

type Response struct {
	TransactionID string
	Status        string
	ErrorMessage  string
}

type Client interface {
	ProcessPayment(ctx context.Context, req Request) (*Response, error)
}
