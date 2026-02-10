CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    balance BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT balance_non_negative CHECK (balance >= 0)
);

CREATE TYPE transaction_status AS ENUM ('pending', 'success', 'failed');

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_account UUID NOT NULL REFERENCES accounts(id),
    to_account UUID NOT NULL REFERENCES accounts(id),
    amount BIGINT NOT NULL,
    status transaction_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT amount_positive CHECK (amount > 0),
    CONSTRAINT different_accounts CHECK (from_account != to_account)
);

CREATE TABLE idempotency_keys (
    key VARCHAR(255) PRIMARY KEY,
    response_code INT NOT NULL,
    response_body JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_from_account ON transactions(from_account);
CREATE INDEX idx_transactions_to_account ON transactions(to_account);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_idempotency_keys_created_at ON idempotency_keys(created_at);
