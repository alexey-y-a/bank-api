package user

import (
	"context"
	"errors"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestService_Register(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		username    string
		password    string
		setupMock   func(repo *mocks.MockUserRepository)
		expectedErr error
	}{
		{
			name:     "успешная регистрация",
			email:    "test@example.com",
			username: "testuser",
			password: "securepass123",
			setupMock: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().FindByEmail(gomock.Any(), "test@example.com").Return(nil, nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name:     "ошибка при регистрации: дубликат email",
			email:    "existing@example.com",
			username: "testuser",
			password: "securepass123",
			setupMock: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().FindByEmail(gomock.Any(), "existing@example.com").
					Return(&domain.User{ID: 1, Email: "existing@example.com"}, nil)
			},
			expectedErr: ErrUserAlreadyExists,
		},
		{
			name:     "ошибка при регистрации: сбой БД при поиске",
			email:    "test@example.com",
			username: "testuser",
			password: "securepass123",
			setupMock: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().FindByEmail(gomock.Any(), "test@example.com").
					Return(nil, errors.New("db error"))
			},
			expectedErr: errors.New("find by email: db error"),
		},
		{
			name:     "ошибка при регистрации: сбой БД при создании",
			email:    "test@example.com",
			username: "testuser",
			password: "securepass123",
			setupMock: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().FindByEmail(gomock.Any(), "test@example.com").Return(nil, nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("db error"))
			},
			expectedErr: errors.New("create user in db: db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockRepo := mocks.NewMockUserRepository(ctrl)
			tt.setupMock(mockRepo)
			svc := NewService(mockRepo, "test-secret", 24)
			result, err := svc.Register(context.Background(), tt.email, tt.username, tt.password)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr.Error())
				require.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, tt.email, result.Email)
				require.Equal(t, tt.username, result.Username)
				require.Empty(t, result.Password)
			}
		})
	}
}
