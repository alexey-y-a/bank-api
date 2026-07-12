package user

import (
	"context"
	"errors"
	"testing"

	"github.com/alexey-y-a/bank-api/internal/domain"
	"github.com/alexey-y-a/bank-api/internal/repository/mocks"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
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
			svc := NewService(mockRepo, nil, "test-secret", 24)
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

func TestService_Login(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		password    string
		setupMock   func(repo *mocks.MockUserRepository)
		expectedErr error
		checkToken  func(t *testing.T, token string)
	}{
		{
			name:     "успешный логин",
			email:    "test@example.com",
			password: "securepass123",
			setupMock: func(repo *mocks.MockUserRepository) {
				hashed, _ := bcrypt.GenerateFromPassword([]byte("securepass123"), bcrypt.DefaultCost)
				repo.EXPECT().FindByEmail(gomock.Any(), "test@example.com").
					Return(&domain.User{
						ID:       123,
						Email:    "test@example.com",
						Username: "testuser",
						Password: string(hashed),
					}, nil)
			},
			expectedErr: nil,
			checkToken: func(t *testing.T, token string) {
				parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{},
					func(t *jwt.Token) (interface{}, error) { return []byte("test-secret"), nil })
				require.NoError(t, err)
				require.True(t, parsed.Valid)
				claims := parsed.Claims.(*jwt.RegisteredClaims)
				require.Equal(t, "123", claims.Subject)
			},
		},
		{
			name:     "пользователь не найден",
			email:    "notfound@example.com",
			password: "anypass",
			setupMock: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().FindByEmail(gomock.Any(), "notfound@example.com").
					Return(nil, nil)
			},
			expectedErr: ErrInvalidCredentials,
		},
		{
			name:     "ошибка неверный пароль",
			email:    "test@example.com",
			password: "wrongpass",
			setupMock: func(repo *mocks.MockUserRepository) {
				hashed, _ := bcrypt.GenerateFromPassword([]byte("correctpass"), bcrypt.DefaultCost)
				repo.EXPECT().FindByEmail(gomock.Any(), "test@example.com").
					Return(&domain.User{
						ID:       123,
						Email:    "test@example.com",
						Password: string(hashed),
					}, nil)
			},
			expectedErr: ErrInvalidCredentials,
		},
		{
			name:     "ошибка: сбой БД при поиске",
			email:    "test@example.com",
			password: "anypass",
			setupMock: func(repo *mocks.MockUserRepository) {
				repo.EXPECT().FindByEmail(gomock.Any(), "test@example.com").
					Return(nil, errors.New("db error"))
			},
			expectedErr: errors.New("find by email: db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockUserRepository(ctrl)
			tt.setupMock(mockRepo)

			svc := NewService(mockRepo, nil, "test-secret", 24)

			token, resultUser, err := svc.Login(context.Background(), tt.email, tt.password)

			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedErr.Error())
				require.Empty(t, token)
				require.Nil(t, resultUser)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, token)
				require.NotNil(t, resultUser)
				require.Equal(t, "test@example.com", resultUser.Email)
				require.Empty(t, resultUser.Password)

				if tt.checkToken != nil {
					tt.checkToken(t, token)
				}
			}
		})
	}
}
