package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/usecase/generateqr"
	"github.com/Xausdorf/qr-pay-hub/pay-gateway/internal/usecase/pay"
)

type Handler struct {
	payUC        *pay.UseCase
	generateQRUC *generateqr.UseCase
}

func NewHandler(payUC *pay.UseCase, generateQRUC *generateqr.UseCase) *Handler {
	return &Handler{
		payUC:        payUC,
		generateQRUC: generateQRUC,
	}
}

type PayRequest struct {
	FromID string `json:"from_id"`
	ToID   string `json:"to_id"`
	Amount int64  `json:"amount"`
}

type PayResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Error         string `json:"error,omitempty"`
}

func (h *Handler) HandlePay(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("X-Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, `{"error":"X-Idempotency-Key header required"}`, http.StatusBadRequest)
		return
	}

	var req PayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	resp, err := h.payUC.Execute(r.Context(), pay.Request{
		IdempotencyKey: idempotencyKey,
		FromID:         req.FromID,
		ToID:           req.ToID,
		Amount:         req.Amount,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(PayResponse{
		TransactionID: resp.TransactionID,
		Status:        resp.Status,
		Error:         resp.Error,
	})
}

func (h *Handler) HandleQR(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "account_id")
	if accountID == "" {
		http.Error(w, `{"error":"account_id required"}`, http.StatusBadRequest)
		return
	}

	amountStr := r.URL.Query().Get("amount")
	if amountStr == "" {
		http.Error(w, `{"error":"amount query param required"}`, http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || amount <= 0 {
		http.Error(w, `{"error":"invalid amount"}`, http.StatusBadRequest)
		return
	}

	png, err := h.generateQRUC.Execute(generateqr.Request{
		AccountID: accountID,
		Amount:    amount,
	})
	if err != nil {
		http.Error(w, `{"error":"qr generation failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(png)
}
