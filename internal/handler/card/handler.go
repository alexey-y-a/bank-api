package card

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/middleware"
	cardservice "github.com/alexey-y-a/bank-api/internal/service/card"
)

type Service interface {
	IssueCard(ctx context.Context, accountID, userID int64) (*domain.Card, error)
	GetUserCards(ctx context.Context, userID int64) ([]*domain.Card, error)
	BlockCard(ctx context.Context, cardID, userID int64) error
	PayWithCard(ctx context.Context, cardID, userID int64, cvv string, amount int64) error
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateCard(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user ID in token", http.StatusInternalServerError)
		return
	}

	var req CreateCardRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.AccountID == 0 {
		http.Error(w, "account_id is required", http.StatusBadRequest)
		return
	}

	card, err := h.service.IssueCard(r.Context(), req.AccountID, userID)
	if err != nil {
		if errors.Is(err, cardservice.ErrCardNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toCardResponse(card))
}

func (h *Handler) GetUserCards(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user ID in token", http.StatusInternalServerError)
	}

	cards, err := h.service.GetUserCards(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toCardListResponse(cards))
}

func (h *Handler) BlockCard(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user ID in token", http.StatusInternalServerError)
	}

	cardIDStr := r.PathValue("id")
	cardID, err := strconv.ParseInt(cardIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid card ID", http.StatusBadRequest)
		return
	}

	err = h.service.BlockCard(r.Context(), cardID, userID)
	if err != nil {
		if errors.Is(err, cardservice.ErrCardNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) PayWithCard(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user  ID in token", http.StatusInternalServerError)
		return
	}

	cardIDStr := r.PathValue("id")
	cardID, err := strconv.ParseInt(cardIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid card ID", http.StatusBadRequest)
		return
	}

	var req PayWithCardRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err = h.service.PayWithCard(r.Context(), cardID, userID, req.CVV, req.Amount)
	if err != nil {
		if errors.Is(err, cardservice.ErrCardNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, cardservice.ErrInvalidCVV) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, domain.ErrCardBlocked) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if errors.Is(err, domain.ErrInsufficientFunds) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}
