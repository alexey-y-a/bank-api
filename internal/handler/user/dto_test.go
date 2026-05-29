package user

import (
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/require"
)

func TestRegisterRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		req         RegisterRequest
		expectError bool
		errorFields []string
	}{
		{
			name: "все поля валидны",
			req: RegisterRequest{
				Email:    "valid@example.com",
				Username: "valid_user",
				Password: "securepass123",
			},
			expectError: false,
		},
		{
			name:        "пустой запрос",
			req:         RegisterRequest{},
			expectError: true,
			errorFields: []string{"email", "username", "password"},
		},
		{
			name: "невалидный email: нет @",
			req: RegisterRequest{
				Email:    "invalid-email",
				Username: "valid_user",
				Password: "securepass123",
			},
			expectError: true,
			errorFields: []string{"email"},
		},
		{
			name: "слишком короткий username",
			req: RegisterRequest{
				Email:    "valid@example.com",
				Username: "ab",
				Password: "securepass123",
			},
			expectError: true,
			errorFields: []string{"username"},
		},
		{
			name: "username с запрещенными символами",
			req: RegisterRequest{
				Email:    "valid@example.com",
				Username: "invalid user!",
				Password: "securepass123",
			},
			expectError: true,
			errorFields: []string{"username"},
		},
		{
			name: "слишком короткий пароль",
			req: RegisterRequest{
				Email:    "valid@example.com",
				Username: "valid_user",
				Password: "short",
			},
			expectError: true,
			errorFields: []string{"password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()

			if !tt.expectError {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)

			errs, ok := err.(validation.Errors)
			require.True(t, ok, "ошибка должна быть validation.Errors")

			for _, field := range tt.errorFields {
				require.Contains(t, errs, field, "ошибка должна содержать поле %s", field)
			}
		})
	}
}

func TestLoginRequest_Validate(t *testing.T) {
	tests := []struct {
		name        string
		req         LoginRequest
		expectError bool
		errorFields []string
	}{
		{
			name: "все поля валидны",
			req: LoginRequest{
				Email:    "test@example.com",
				Password: "anypassword",
			},
			expectError: false,
		},
		{
			name:        "пустой запрос",
			req:         LoginRequest{},
			expectError: true,
			errorFields: []string{"email", "password"},
		},
		{
			name: "отсутствует email",
			req: LoginRequest{
				Password: "anypassword",
			},
			expectError: true,
			errorFields: []string{"email"},
		},
		{
			name: "отсутствует password",
			req: LoginRequest{
				Email: "test@example.com",
			},
			expectError: true,
			errorFields: []string{"password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()

			if !tt.expectError {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)

			errs, ok := err.(validation.Errors)
			require.True(t, ok, "ошибка должна быть validation.Errors")

			for _, field := range tt.errorFields {
				require.Contains(t, errs, field, "ошибка должна содержать поле %s", field)
			}
		})
	}
}
