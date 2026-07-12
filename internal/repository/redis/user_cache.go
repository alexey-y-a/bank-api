package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alexey-y-a/bank-api/internal/domain"
	goredis "github.com/redis/go-redis/v9"
)

type UserCache struct {
	client *goredis.Client
}

func NewUserCache(addr, password string) *UserCache {
	client := goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	return &UserCache{client: client}
}

func buildUserKey(userID int64) string {
	return fmt.Sprintf("user:%d", userID)
}

func (c *UserCache) Set(ctx context.Context, userID int64, user *domain.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("user_cache.marshal: %w", err)
	}

	err = c.client.Set(ctx, buildUserKey(userID), data, 15*time.Minute).Err()
	if err != nil {
		return fmt.Errorf("user_cache.set: %w", err)
	}

	return nil
}

func (c *UserCache) Get(ctx context.Context, userID int64) (*domain.User, error) {
	data, err := c.client.Get(ctx, buildUserKey(userID)).Bytes()
	if err != nil {
		if err == goredis.Nil {
			return nil, nil
		}

		return nil, fmt.Errorf("user_cache.get redis: %w", err)
	}

	var user domain.User
	err = json.Unmarshal(data, &user)
	if err != nil {
		return nil, fmt.Errorf("user_cache.get unmarshal: %w", err)
	}

	return &user, nil
}

func (c *UserCache) Delete(ctx context.Context, userID int64) error {
	err := c.client.Del(ctx, buildUserKey(userID)).Err()
	if err != nil {
		return fmt.Errorf("user_cache.delete: %w", err)
	}

	return nil
}

func (c *UserCache) Close() error {
	return c.client.Close()
}
