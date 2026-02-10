package http //nolint:revive // directory-based package name, imported with alias

import (
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const requestTimeout = 30 * time.Second

func NewRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(requestTimeout))

	r.Post("/api/pay", h.HandlePay)
	r.Get("/api/qr/{account_id}", h.HandleQR)

	return r
}
