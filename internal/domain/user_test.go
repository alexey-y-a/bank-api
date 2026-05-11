package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		username    string
		password    string
		expectedErr error
	}{
		{
			name:        "успешное создание валидного пользователя",
			email:       "test@example.com",
			username:    "ivan_ivanov",
			password:    "secure_password_123",
			expectedErr: nil,
		},
		{
			name:        "ошибка при пустом email",
			email:       "",
			username:    "ivan_ivanov",
			password:    "secure_password_123",
			expectedErr: ErrEmptyFields,
		},
		{
			name:        "ошибка при пустом username",
			email:       "test@example.com",
			username:    "",
			password:    "secure_password_123",
			expectedErr: ErrEmptyFields,
		},
		{
			name:        "ошибка при пустом пароле",
			email:       "test@example.com",
			username:    "ivan_ivanov",
			password:    "",
			expectedErr: ErrEmptyFields,
		},
		{
			name:        "ошибка при слишком коротком email (менее 5 символов)",
			email:       "a@b",
			username:    "ivan_ivanov",
			password:    "secure_password_123",
			expectedErr: ErrInvalidEmail,
		},
		{
			name:        "ошибка при слишком коротком username (менее 3 символов)",
			email:       "test@example.com",
			username:    "ab",
			password:    "secure_password_123",
			expectedErr: ErrInvalidUsername,
		},
		{
			name:        "ошибка при слабом пароле (менее 8 символов)",
			email:       "test@example.com",
			username:    "ivan_ivanov",
			password:    "123456",
			expectedErr: ErrWeakPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := NewUser(tt.email, tt.username, tt.password)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)

				require.Nil(t, user, "при ошибке пользователь должен быть nil")

				return
			}

			require.NoError(t, err, "ожидалось успешное создание пользователя")

			require.NotNil(t, user, "пользователь должен быть создан")

			require.Equal(t, tt.email, user.Email, "email должен совпадать")

			require.Equal(t, tt.username, user.Username, "username должен совпадать")

			require.Equal(t, tt.password, user.Password, "пароль должен совпадать")

			require.False(t, user.CreatedAt.IsZero(), "CreatedAt должен быть установлен")

			require.False(t, user.UpdatedAt.IsZero(), "UpdatedAt должен быть установлен")
		})
	}
}
