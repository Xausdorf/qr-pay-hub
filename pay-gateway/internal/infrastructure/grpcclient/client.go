package grpcclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/Xausdorf/qr-pay-hub/pay-gateway/gen/pb"
	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/domain/payment"
)

type Client struct {
	client pb.PaymentProcessorClient
	conn   *grpc.ClientConn
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: pb.NewPaymentProcessorClient(conn),
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) ProcessPayment(ctx context.Context, req payment.Request) (*payment.Response, error) {
	resp, err := c.client.ProcessPayment(ctx, &pb.PaymentRequest{
		IdempotencyKey: req.IdempotencyKey,
		FromAccountId:  req.FromAccountID.String(),
		ToAccountId:    req.ToAccountID.String(),
		Amount:         req.Amount,
	})
	if err != nil {
		return nil, err
	}

	return &payment.Response{
		TransactionID: resp.GetTransactionId(),
		Status:        resp.GetStatus().String(),
		ErrorMessage:  resp.GetErrorMessage(),
	}, nil
}
