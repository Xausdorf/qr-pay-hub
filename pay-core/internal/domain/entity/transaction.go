package entity

import (
	"time"

	"github.com/google/uuid"
)

type TransactionStatus string

const (
	StatusPending TransactionStatus = "pending"
	StatusSuccess TransactionStatus = "success"
	StatusFailed  TransactionStatus = "failed"
)

type Transaction struct {
	id          uuid.UUID
	fromAccount uuid.UUID
	toAccount   uuid.UUID
	amount      int64
	status      TransactionStatus
	createdAt   time.Time
}

func NewTransaction(from, to uuid.UUID, amount int64, status TransactionStatus) *Transaction {
	return &Transaction{
		id:          uuid.New(),
		fromAccount: from,
		toAccount:   to,
		amount:      amount,
		status:      status,
		createdAt:   time.Now(),
	}
}

func ReconstructTransaction(
	id, from, to uuid.UUID,
	amount int64,
	status TransactionStatus,
	createdAt time.Time,
) *Transaction {
	return &Transaction{
		id:          id,
		fromAccount: from,
		toAccount:   to,
		amount:      amount,
		status:      status,
		createdAt:   createdAt,
	}
}

func (t *Transaction) ID() uuid.UUID {
	return t.id
}

func (t *Transaction) FromAccount() uuid.UUID {
	return t.fromAccount
}

func (t *Transaction) ToAccount() uuid.UUID {
	return t.toAccount
}

func (t *Transaction) Amount() int64 {
	return t.amount
}

func (t *Transaction) Status() TransactionStatus {
	return t.status
}

func (t *Transaction) CreatedAt() time.Time {
	return t.createdAt
}
