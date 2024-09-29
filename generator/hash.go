package generator

import (
	"crypto/rand"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func VerifyPassword(hashedStr, plainStr string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedStr), []byte(plainStr))
	return err == nil
}

func GeneratePlainPassword(length int) (string, error) {
	const charset = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	var password []byte
	for i := 0; i < length; i++ {
		charIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password = append(password, charset[charIndex.Int64()])
	}
	return string(password), nil
}
