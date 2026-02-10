package integration_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Xausdorf/qr-pay-hub/gen/pb"
)

const (
	dbURL    = "postgres://qrpay:qrpay_secret@localhost:5432/qrpay?sslmode=disable"
	grpcAddr = "localhost:50051"
)

func TestDoubleSpendingAttack(t *testing.T) {
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

	_, err = pool.Exec(ctx, `INSERT INTO accounts (id, balance) VALUES ($1, 1000), ($2, 0)`, senderID, receiverID)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM transactions WHERE from_account = $1`, senderID)
		pool.Exec(
			context.Background(),
			`DELETE FROM idempotency_keys WHERE key LIKE $1`,
			fmt.Sprintf("double-spend-%s-%%", senderID),
		)
		pool.Exec(context.Background(), `DELETE FROM accounts WHERE id IN ($1, $2)`, senderID, receiverID)
	})

	const goroutines = 10
	var wg sync.WaitGroup
	var successCount atomic.Int32
	var failedCount atomic.Int32

	wg.Add(goroutines)
	for i := range goroutines {
		go func(idx int) {
			defer wg.Done()

			resp, rpcErr := client.ProcessPayment(ctx, &pb.PaymentRequest{
				IdempotencyKey: fmt.Sprintf("double-spend-%s-%d", senderID, idx),
				FromAccountId:  senderID.String(),
				ToAccountId:    receiverID.String(),
				Amount:         1000,
			})

			if rpcErr != nil {
				t.Logf("goroutine %d: error %v", idx, rpcErr)
				return
			}

			if resp.GetStatus() == pb.TransactionStatus_TRANSACTION_STATUS_SUCCESS {
				successCount.Add(1)
				t.Logf("goroutine %d: SUCCESS, tx=%s", idx, resp.GetTransactionId())
			} else {
				failedCount.Add(1)
				t.Logf("goroutine %d: FAILED, reason=%s", idx, resp.GetErrorMessage())
			}
		}(i)
	}

	wg.Wait()

	require.Equal(t, int32(1), successCount.Load(), "exactly 1 request should succeed")
	require.Equal(t, int32(9), failedCount.Load(), "exactly 9 requests should fail")

	var senderBalance, receiverBalance int64
	err = pool.QueryRow(ctx, `SELECT balance FROM accounts WHERE id = $1`, senderID).Scan(&senderBalance)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `SELECT balance FROM accounts WHERE id = $1`, receiverID).Scan(&receiverBalance)
	require.NoError(t, err)

	require.Equal(t, int64(0), senderBalance, "sender balance must be 0")
	require.Equal(t, int64(1000), receiverBalance, "receiver balance must be 1000")

	t.Logf("Final balances: sender=%d, receiver=%d", senderBalance, receiverBalance)
}
