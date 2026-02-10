package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/Xausdorf/qr-pay-hub/gen/pb"
	grpchandler "github.com/Xausdorf/qr-pay-hub/internal/delivery/grpc"
	"github.com/Xausdorf/qr-pay-hub/internal/infrastructure/config"
	"github.com/Xausdorf/qr-pay-hub/internal/infrastructure/postgres"
	"github.com/Xausdorf/qr-pay-hub/internal/usecase/transfer"
)

const (
	dbMaxConns        = 10
	dbMinConns        = 2
	dbMaxConnLifetime = 30 * time.Minute
	dbMaxConnIdleTime = 5 * time.Minute
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := config.Load()

	pool, err := initDB(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database init failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	uow := postgres.NewUnitOfWork(pool)
	transferUC := transfer.NewUseCase(uow)
	handler := grpchandler.NewHandler(transferUC)

	srv := grpc.NewServer()
	pb.RegisterPaymentProcessorServer(srv, handler)
	reflection.Register(srv)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Error("listen failed", "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("gRPC server starting", "addr", cfg.GRPCAddr)
		if err := srv.Serve(lis); err != nil {
			logger.Error("serve failed", "error", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down...")
	srv.GracefulStop()
}

func initDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = dbMaxConns
	cfg.MinConns = dbMinConns
	cfg.MaxConnLifetime = dbMaxConnLifetime
	cfg.MaxConnIdleTime = dbMaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
