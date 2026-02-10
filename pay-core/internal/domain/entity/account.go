package entity

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrNegativeAmount    = errors.New("amount must be positive")
)

type Account struct {
	id      uuid.UUID
	balance int64
}

func NewAccount(id uuid.UUID, balance int64) *Account {
	return &Account{
		id:      id,
		balance: balance,
	}
}

func (a *Account) ID() uuid.UUID {
	return a.id
}

func (a *Account) Balance() int64 {
	return a.balance
}

func (a *Account) Debit(amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	if a.balance < amount {
		return ErrInsufficientFunds
	}
	a.balance -= amount
	return nil
}

func (a *Account) Credit(amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	a.balance += amount
	return nil
}
