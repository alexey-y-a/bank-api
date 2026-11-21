package analytics

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/middleware"
)

const monthLayout = "2006-01"

type Service interface {
	GetIncomeExpense(ctx context.Context, userID int64, from, to time.Time) (domain.PeriodSummary, []*domain.Event, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetIncomeExpense(w http.ResponseWriter, r *http.Request) {
	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user id token", http.StatusInternalServerError)
		return
	}

	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().UTC().Format(monthLayout)
	}

	from, err := time.Parse(monthLayout, month)
	if err != nil {
		http.Error(w, "invalid month format, expected YYYY-MM", http.StatusBadRequest)
		return
	}

	to := from.AddDate(0, 1, 0)

	summary, topOperations, err := h.service.GetIncomeExpense(r.Context(), userID, from, to)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(toIncomeExpenseResponse(month, summary, topOperations))
}
