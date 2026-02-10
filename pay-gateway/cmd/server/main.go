package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpdelivery "github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/delivery/http"
	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/infrastructure/config"
	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/infrastructure/grpcclient"
	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/infrastructure/qrgenerator"
	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/usecase/generateqr"
	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/usecase/pay"
)

const (
	qrCodeSize            = 256
	readHeaderTimeout     = 5 * time.Second
	gracefulShutdownDelay = 5 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg := config.Load()

	paymentClient, err := grpcclient.NewClient(cfg.CoreGRPCAddr)
	if err != nil {
		logger.Error("grpc client init failed", "error", err)
		cancel()
		return
	}
	defer paymentClient.Close()

	qrGen := qrgenerator.NewGenerator(qrCodeSize)

	payUC := pay.NewUseCase(paymentClient)
	generateQRUC := generateqr.NewUseCase(qrGen)

	handler := httpdelivery.NewHandler(payUC, generateQRUC)
	router := httpdelivery.NewRouter(handler)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		logger.Info("HTTP server starting", "addr", cfg.HTTPAddr)
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			logger.Error("http serve failed", "error", serveErr)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), gracefulShutdownDelay)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}
