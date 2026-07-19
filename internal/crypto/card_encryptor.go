package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

type CardEncryptor struct {
	aesKey     []byte
	hmacSecret []byte
}

func NewCardEncryptor(aesSecret, hmacSecret string) *CardEncryptor {
	return &CardEncryptor{
		aesKey:     []byte(aesSecret),
		hmacSecret: []byte(hmacSecret),
	}
}

func (e *CardEncryptor) EncryptNumber(plaintext string) (hash string, encrypted []byte, hmacValue string, err error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, "", fmt.Errorf("encrypt number hash: %w", err)
	}

	block, err := aes.NewCipher(e.aesKey)
	if err != nil {
		return "", nil, "", fmt.Errorf("encrypt number aes: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", nil, "", fmt.Errorf("encrypt number gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", nil, "", fmt.Errorf("encrypt number nonce: %w", err)
	}

	encrypted = gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	mac := hmac.New(sha256.New, e.hmacSecret)
	mac.Write([]byte(plaintext))
	hmacValue = hex.EncodeToString(mac.Sum(nil))

	return string(hashedBytes), encrypted, hmacValue, nil
}

func (e *CardEncryptor) DecryptNumber(encrypted []byte) (string, error) {
	block, err := aes.NewCipher(e.aesKey)
	if err != nil {
		return "", fmt.Errorf("decrypt number aes: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("decrypt number gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(encrypted) < nonceSize {
		return "", fmt.Errorf("decrypt number: ciphertext too short")
	}

	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt number open: %w", err)
	}

	return string(plaintext), nil
}

func (e *CardEncryptor) HashCVV(cvv string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(cvv), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash cvv: %w", err)
	}

	return string(hashed), nil
}

func (e *CardEncryptor) VerifyCVV(hash string, cvv string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(cvv))
	if err != nil {
		return fmt.Errorf("cvv mismatch: %w", err)
	}

	return nil
}

func (e *CardEncryptor) ComputeHMAC(data string) string {
	mac := hmac.New(sha256.New, e.hmacSecret)
	mac.Write([]byte(data))

	return hex.EncodeToString(mac.Sum(nil))
}
