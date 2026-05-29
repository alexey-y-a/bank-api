package user

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alexey-y-a/bank-api/internal/domain"
	userservice "github.com/alexey-y-a/bank-api/internal/service/user"
)

type UserService interface {
	Register(ctx context.Context, email, username, password string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (string, *domain.User, error)
}

type Handler struct {
	service UserService
}

func NewHandler(service UserService) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err = req.Validate()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(err)
		return
	}

	user, err := h.service.Register(r.Context(), req.Email, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, userservice.ErrUserAlreadyExists) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toUserResponse(user))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	err = req.Validate()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(err)
		return
	}

	token, user, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, userservice.ErrInvalidCredentials) {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := LoginResponse{
		Token: token,
		User:  toUserResponse(user),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
