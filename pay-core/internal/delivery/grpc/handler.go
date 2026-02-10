package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/Xausdorf/qr-pay-hub/gen/pb"
	"github.com/Xausdorf/qr-pay-hub/internal/domain/entity"
	"github.com/Xausdorf/qr-pay-hub/internal/usecase/transfer"
)

type Handler struct {
	pb.UnimplementedPaymentProcessorServer

	transferUC *transfer.UseCase
}

func NewHandler(transferUC *transfer.UseCase) *Handler {
	return &Handler{transferUC: transferUC}
}

func (h *Handler) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {
	if req.GetIdempotencyKey() == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}
	if req.GetAmount() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be positive")
	}

	fromID, err := uuid.Parse(req.GetFromAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid from_account_id")
	}

	toID, err := uuid.Parse(req.GetToAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid to_account_id")
	}

	if fromID == toID {
		return nil, status.Error(codes.InvalidArgument, "from and to accounts must differ")
	}

	resp, err := h.transferUC.Execute(ctx, transfer.Request{
		IdempotencyKey: req.GetIdempotencyKey(),
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         req.GetAmount(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "transfer failed: %v", err)
	}

	return &pb.PaymentResponse{
		TransactionId: resp.TransactionID,
		Status:        mapStatus(resp.Status),
		ErrorMessage:  resp.ErrorMessage,
	}, nil
}

func mapStatus(s entity.TransactionStatus) pb.TransactionStatus {
	switch s {
	case entity.StatusPending:
		return pb.TransactionStatus_TRANSACTION_STATUS_PENDING
	case entity.StatusSuccess:
		return pb.TransactionStatus_TRANSACTION_STATUS_SUCCESS
	case entity.StatusFailed:
		return pb.TransactionStatus_TRANSACTION_STATUS_FAILED
	default:
		return pb.TransactionStatus_TRANSACTION_STATUS_UNSPECIFIED
	}
}
