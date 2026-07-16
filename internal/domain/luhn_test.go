package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidLuhn(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   bool
	}{
		{
			name:   "валидный номер Visa (тестовый)",
			number: "4111111111111111",
			want:   true,
		},
		{
			name:   "валидный номер MasterCard (тестовый)",
			number: "5555555555554444",
			want:   true,
		},
		{
			name:   "невалидный номер - последняя цифра неверная",
			number: "4111111111111112",
			want:   false,
		},
		{
			name:   "невалидный номер - содержит буквы",
			number: "4532abcd13416220",
			want:   false,
		},
		{
			name:   "невалидный номер - пустая строка",
			number: "",
			want:   false,
		},
		{
			name:   "номер из одного символа",
			number: "1",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidLuhn(tt.number)
			require.Equal(t, tt.want, got, "ValidLuhn(%s) = %v, want %v", tt.number, got, tt.want)
		})
	}
}

func TestGenerateCardNumber(t *testing.T) {
	number := GenerateCardNumber()

	require.Len(t, number, 16, "номер карты должен быть 16 цифр")

	valid := ValidLuhn(number)
	require.True(t, valid, "сгенерированный номер должен быть валидным по алгоритму Луна")

	for i, ch := range number {
		require.True(t, ch >= '0' && ch <= '9',
			"символ на позиции %d должен быть цифрой, получен '%c'", i, ch)
	}
}
