package user

import (
	"regexp"

	"github.com/alexey-y-a/bank-api/internal/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResponse struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func (r RegisterRequest) Validate() error {
	return validation.ValidateStruct(
		&r,
		validation.Field(
			&r.Email,
			validation.Required.Error("email is required"),
			is.EmailFormat.Error("invalid email format"),
			validation.Length(5, 255).Error("email must be between 5 and 255 characters"),
		),
		validation.Field(
			&r.Username,
			validation.Required.Error("username is required"),
			validation.Length(3, 50).Error("username mast be between 3 and 50 characters"),
			validation.Match(regexp.MustCompile(`^[a-zA-Z0-9_]+$`)).Error("username can only contain letters, digits and underscores"),
		),
		validation.Field(
			&r.Password,
			validation.Required.Error("password is required"),
			validation.Length(8, 128).Error("password must be between 8 and 128 characters"),
		),
	)
}

func (r LoginRequest) Validate() error {
	return validation.ValidateStruct(
		&r,
		validation.Field(
			&r.Email,
			validation.Required.Error("email is required"),
		),
		validation.Field(
			&r.Password,
			validation.Required.Error("password is required"),
		),
	)
}
