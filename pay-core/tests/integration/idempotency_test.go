package integration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Xausdorf/qr-pay-hub/gen/pb"
)

func TestIdempotencyNetworkRetryStorm(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	conn, connErr := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, connErr)
	defer conn.Close()

	client := pb.NewPaymentProcessorClient(conn)

	senderID := uuid.New()
	receiverID := uuid.New()
	idempotencyKey := uuid.New().String()

	_, err = pool.Exec(ctx, `INSERT INTO accounts (id, balance) VALUES ($1, 5000), ($2, 0)`, senderID, receiverID)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM transactions WHERE from_account = $1`, senderID)
		pool.Exec(context.Background(), `DELETE FROM idempotency_keys WHERE key = $1`, idempotencyKey)
		pool.Exec(context.Background(), `DELETE FROM accounts WHERE id IN ($1, $2)`, senderID, receiverID)
	})

	makeRequest := func() *pb.PaymentResponse {
		resp, rpcErr := client.ProcessPayment(ctx, &pb.PaymentRequest{
			IdempotencyKey: idempotencyKey,
			FromAccountId:  senderID.String(),
			ToAccountId:    receiverID.String(),
			Amount:         1000,
		})
		require.NoError(t, rpcErr)
		return resp
	}

	t.Run("sequential_retries", func(t *testing.T) {
		responses := make([]*pb.PaymentResponse, 10)
		for i := range 10 {
			responses[i] = makeRequest()
			t.Logf("sequential %d: tx=%s, status=%s", i, responses[i].GetTransactionId(), responses[i].GetStatus())
		}

		firstTxID := responses[0].GetTransactionId()
		for i, resp := range responses {
			require.Equal(t, firstTxID, resp.GetTransactionId(), "response %d has different tx_id", i)
			require.Equal(t, pb.TransactionStatus_TRANSACTION_STATUS_SUCCESS, resp.GetStatus())
		}
	})

	var senderBalance int64
	err = pool.QueryRow(ctx, `SELECT balance FROM accounts WHERE id = $1`, senderID).Scan(&senderBalance)
	require.NoError(t, err)
	require.Equal(t, int64(4000), senderBalance, "balance should decrease exactly once: 5000 - 1000 = 4000")

	newIdempotencyKey := uuid.New().String()
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM idempotency_keys WHERE key = $1`, newIdempotencyKey)
	})

	t.Run("concurrent_retries", func(t *testing.T) {
		const workers = 10
		responses := make([]*pb.PaymentResponse, workers)
		var wg sync.WaitGroup
		var mu sync.Mutex

		wg.Add(workers)
		for i := range workers {
			go func(idx int) {
				defer wg.Done()
				resp, rpcErr := client.ProcessPayment(ctx, &pb.PaymentRequest{
					IdempotencyKey: newIdempotencyKey,
					FromAccountId:  senderID.String(),
					ToAccountId:    receiverID.String(),
					Amount:         500,
				})
				assert.NoError(t, rpcErr)

				mu.Lock()
				responses[idx] = resp
				mu.Unlock()
				t.Logf("concurrent %d: tx=%s, status=%s", idx, resp.GetTransactionId(), resp.GetStatus())
			}(i)
		}
		wg.Wait()

		firstTxID := ""
		for i, resp := range responses {
			if resp == nil {
				continue
			}
			if firstTxID == "" {
				firstTxID = resp.GetTransactionId()
			}
			require.Equal(t, firstTxID, resp.GetTransactionId(), "concurrent response %d has different tx_id", i)
		}
	})

	err = pool.QueryRow(ctx, `SELECT balance FROM accounts WHERE id = $1`, senderID).Scan(&senderBalance)
	require.NoError(t, err)
	require.Equal(t, int64(3500), senderBalance, "balance should decrease exactly once more: 4000 - 500 = 3500")

	var txCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM transactions WHERE from_account = $1`, senderID).Scan(&txCount)
	require.NoError(t, err)
	require.Equal(t, 2, txCount, "exactly 2 transactions should exist")

	t.Logf("Final sender balance: %d, total transactions: %d", senderBalance, txCount)
}

func TestIdempotencyPreservesFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	conn, connErr := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, connErr)
	defer conn.Close()

	client := pb.NewPaymentProcessorClient(conn)

	senderID := uuid.New()
	receiverID := uuid.New()
	idempotencyKey := uuid.New().String()

	_, err = pool.Exec(ctx, `INSERT INTO accounts (id, balance) VALUES ($1, 100), ($2, 0)`, senderID, receiverID)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM transactions WHERE from_account = $1`, senderID)
		pool.Exec(context.Background(), `DELETE FROM idempotency_keys WHERE key = $1`, idempotencyKey)
		pool.Exec(context.Background(), `DELETE FROM accounts WHERE id IN ($1, $2)`, senderID, receiverID)
	})

	responses := make([]*pb.PaymentResponse, 5)
	for i := range 5 {
		resp, rpcErr := client.ProcessPayment(ctx, &pb.PaymentRequest{
			IdempotencyKey: idempotencyKey,
			FromAccountId:  senderID.String(),
			ToAccountId:    receiverID.String(),
			Amount:         500,
		})
		require.NoError(t, rpcErr)
		responses[i] = resp
		t.Logf("retry %d: status=%s, error=%s", i, resp.GetStatus(), resp.GetErrorMessage())
	}

	for i, resp := range responses {
		require.Equal(
			t,
			pb.TransactionStatus_TRANSACTION_STATUS_FAILED,
			resp.GetStatus(),
			"retry %d should be FAILED",
			i,
		)
		require.Contains(t, resp.GetErrorMessage(), "insufficient funds")
	}

	var balance int64
	err = pool.QueryRow(ctx, `SELECT balance FROM accounts WHERE id = $1`, senderID).Scan(&balance)
	require.NoError(t, err)
	require.Equal(t, int64(100), balance, "balance should remain unchanged")

	t.Logf("Idempotency correctly preserves FAILED status across retries")
}
