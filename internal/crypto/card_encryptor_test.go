package crypto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	enc := NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")

	plaintext := "4532148813416220"

	hash, encrypted, hmacValue, err := enc.EncryptNumber(plaintext)
	require.NoError(t, err, "EncryptNumber не должна возвращать ошибку")
	require.NotEmpty(t, hash, "bcrypt-хеш не должен быть пустым")
	require.NotEmpty(t, encrypted, "шифротекст не должен быть пустым")
	require.NotEmpty(t, hmacValue, "HMAC не должен быть пустым")
	require.NotEqual(t, plaintext, hash, "bcrypt-хеш не должен совпадать с исходным номером")
	require.NotEqual(t, plaintext, string(encrypted), "шифротекст не должен совпадать с исходным номером")

	decrypted, err := enc.DecryptNumber(encrypted)
	require.NoError(t, err, "DecryptNumber не должна возвращать ошибку")
	require.Equal(t, plaintext, decrypted, "расшифрованный номер должен совпадать с исходным")
}

func TestEncryptTwice_DifferentCiphertext(t *testing.T) {
	enc := NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")

	plaintext := "5555555555554444"

	_, encrypted1, _, err := enc.EncryptNumber(plaintext)
	require.NoError(t, err)

	_, encrypted2, _, err := enc.EncryptNumber(plaintext)
	require.NoError(t, err)

	require.NotEqual(t, encrypted1, encrypted2, "каждое шифрование должно давать уникальный шифротекст из-за nonce")

	dec1, err := enc.DecryptNumber(encrypted1)
	require.NoError(t, err)
	require.Equal(t, plaintext, dec1)

	dec2, err := enc.DecryptNumber(encrypted2)
	require.NoError(t, err)
	require.Equal(t, plaintext, dec2)
}

func TestDecrypt_InvalidData(t *testing.T) {
	enc := NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")

	_, err := enc.DecryptNumber([]byte{})
	require.Error(t, err, "расшифровка пустых данных должна вернуть ошибку")

	_, err = enc.DecryptNumber([]byte{1, 2, 3})
	require.Error(t, err, "расшифровка коротких данных должна вернуть ошибку")
}

func TestHashCVV_VerifyCVV(t *testing.T) {
	enc := NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")

	cvv := "123"

	hash, err := enc.HashCVV(cvv)
	require.NoError(t, err, "HashCVV не должна возвращать ошибку")
	require.NotEmpty(t, hash, "хеш CVV не должен быть пустым")
	require.NotEqual(t, cvv, hash, "хеш не должен совпадать с исходным CVV")

	err = enc.VerifyCVV(hash, cvv)
	require.NoError(t, err, "VerifyCVV с правильным CVV должна проходить")

	err = enc.VerifyCVV(hash, "999")
	require.Error(t, err, "VerifyCVV с неправильным CVV должна вернуть ошибку")
}

func TestComputeHMAC(t *testing.T) {
	enc := NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")

	data := "4532148813416220"

	hmac1 := enc.ComputeHMAC(data)
	hmac2 := enc.ComputeHMAC(data)

	require.NotEmpty(t, hmac1, "HMAC не должен быть пустым")
	require.Equal(t, hmac1, hmac2, "одинаковые данные должны давать одинаковый HMAC")
}

func TestComputeHMAC_DifferentData(t *testing.T) {
	enc := NewCardEncryptor("test-aes-secret-32byte!!", "test-hmac-secret")

	hmac1 := enc.ComputeHMAC("4532148813416220")
	hmac2 := enc.ComputeHMAC("4532148813416221")

	require.NotEqual(t, hmac1, hmac2, "разные данные должны давать разные HMAC")
}
