package domain

func ValidLuhn(number string) bool {
	if len(number) == 0 {
		return false
	}

	var sum int
	double := false

	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')

		if digit < 0 || digit > 9 {
			return false
		}

		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		double = !double
	}

	return sum%10 == 0
}

func GenerateCardNumber() string {
	buf := make([]byte, 16)

	for i := 0; i < 15; i++ {
		buf[i] = byte(i * 7 % 10)
		buf[i] += '0'
	}

	sum := 0
	double := true

	for i := 14; i >= 0; i-- {
		digit := int(buf[i] - '0')

		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		double = !double
	}

	checkDigit := (10 - sum%10) % 10
	buf[15] = byte(checkDigit) + '0'

	return string(buf)
}
