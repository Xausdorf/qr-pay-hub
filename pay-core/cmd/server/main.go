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
		cancel()
		return
	}
	defer pool.Close()

	uow := postgres.NewUnitOfWork(pool)
	transferUC := transfer.NewUseCase(uow)
	handler := grpchandler.NewHandler(transferUC)

	srv := grpc.NewServer()
	pb.RegisterPaymentProcessorServer(srv, handler)
	reflection.Register(srv)

	var lc net.ListenConfig
	lis, lisErr := lc.Listen(ctx, "tcp", cfg.GRPCAddr)
	if lisErr != nil {
		logger.Error("listen failed", "error", lisErr)
		return
	}

	go func() {
		logger.Info("gRPC server starting", "addr", cfg.GRPCAddr)
		if serveErr := srv.Serve(lis); serveErr != nil {
			logger.Error("serve failed", "error", serveErr)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down...")
	srv.GracefulStop()
}

func initDB(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pgCfg, parseErr := pgxpool.ParseConfig(url)
	if parseErr != nil {
		return nil, parseErr
	}

	pgCfg.MaxConns = dbMaxConns
	pgCfg.MinConns = dbMinConns
	pgCfg.MaxConnLifetime = dbMaxConnLifetime
	pgCfg.MaxConnIdleTime = dbMaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		return nil, err
	}

	if pingErr := pool.Ping(ctx); pingErr != nil {
		pool.Close()
		return nil, pingErr
	}

	return pool, nil
}
