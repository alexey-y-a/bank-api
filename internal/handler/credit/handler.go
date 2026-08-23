package credit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/middleware"
	creditservice "github.com/alexey-y-a/bank-api/internal/service/credit"
)

type Service interface {
	CreateCredit(ctx context.Context, accountID, userID int64, amount int64, rate float64, term int64) (*domain.Credit, []*domain.CreditScheduleItem, error)
	GetSchedule(ctx context.Context, creditID, userID int64) ([]*domain.CreditScheduleItem, error)
	MakePayment(ctx context.Context, creditID, userID int64) error
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateCredit(w http.ResponseWriter, r *http.Request) {
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

	var req CreateCreditRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	credit, schedule, err := h.service.CreateCredit(r.Context(), req.AccountID, userID, req.Amount, req.Rate, req.Term)
	if err != nil {
		if errors.Is(err, creditservice.ErrCreditNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, creditservice.ErrForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"credit":   toCreditResponse(credit),
		"schedule": toScheduleListResponse(schedule),
	})
}

func (h *Handler) GetSchedule(w http.ResponseWriter, r *http.Request) {
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

	creditIDStr := r.PathValue("id")
	creditID, err := strconv.ParseInt(creditIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid credit ID", http.StatusBadRequest)
		return
	}

	schedule, err := h.service.GetSchedule(r.Context(), creditID, userID)
	if err != nil {
		if errors.Is(err, creditservice.ErrCreditNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, creditservice.ErrForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toScheduleListResponse(schedule))
}

func (h *Handler) MakePayment(w http.ResponseWriter, r *http.Request) {
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

	creditIDStr := r.PathValue("id")
	creditID, err := strconv.ParseInt(creditIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid credit ID", http.StatusBadRequest)
		return
	}

	err = h.service.MakePayment(r.Context(), creditID, userID)
	if err != nil {
		if errors.Is(err, creditservice.ErrCreditNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, creditservice.ErrForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if errors.Is(err, creditservice.ErrInsufficientFunds) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
