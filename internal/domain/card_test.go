package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewCard(t *testing.T) {
	t.Run("успешное создание карты", func(t *testing.T) {
		card, err := NewCard(1)

		require.NoError(t, err, "не должно быть ошибки")
		require.Equal(t, int64(1), card.AccountID, "счет должен совпадать")
		require.Len(t, card.Number, 16, "номер должен быть 16 цифр")

		valid := ValidLuhn(card.Number)
		require.True(t, valid, "номер карты должен проходить проверку Луна")

		require.Equal(t, CardStatusActive, card.Status, "новая карта должна быть активной")

		expectedExpiry := time.Date(time.Now().Year()+3, time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
		require.Equal(t, expectedExpiry.Year(), card.ExpiresAt.Year(), "год срока должен быть + 3")
		require.Equal(t, expectedExpiry.Month(), card.ExpiresAt.Month(), "месяц срока должен совпадать")
	})
}

func TestCard_MaskNumber(t *testing.T) {
	t.Run("маскировка полного номера из 16 цифр", func(t *testing.T) {
		card := &Card{Number: "4532148813416220"}
		masked := card.MaskNumber()

		require.Equal(t, "**** **** **** 6220", masked, "должны видеть последние 4 цифры")
	})

	t.Run("маскировка при пустом номере", func(t *testing.T) {
		card := &Card{Number: ""}
		masked := card.MaskNumber()

		require.Equal(t, "**** **** **** ****", masked, "при пустом номере полная маска")
	})
}

func TestCard_CanUse(t *testing.T) {
	t.Run("активная карта с неистекшим сроком - можно использовать", func(t *testing.T) {
		card := &Card{
			Status:    CardStatusActive,
			ExpiresAt: time.Now().AddDate(1, 0, 0),
		}

		err := card.CanUse()
		require.NoError(t, err, "активную карту можно использовать")
	})

	t.Run("заблокированная карта - ошибка", func(t *testing.T) {
		card := &Card{
			Status:    CardStatusBlocked,
			ExpiresAt: time.Now().AddDate(1, 0, 0),
		}

		err := card.CanUse()
		require.ErrorIs(t, err, ErrCardBlocked, "заблокированную карту нельзя использовать")
	})

	t.Run("просроченная карта - ошибка", func(t *testing.T) {
		card := &Card{
			Status:    CardStatusActive,
			ExpiresAt: time.Now().AddDate(-1, 0, 0),
		}

		err := card.CanUse()
		require.ErrorIs(t, err, ErrCardExpired, "просроченную карту нельзя использовать")
	})
}
