package passw

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const DefaultCost = 10

var ErrMismatchedHashAndPassword = errors.New("mismatched hash and password")

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func VerifyPassword(hashedPassword, password []byte) error {
	if err := bcrypt.CompareHashAndPassword(hashedPassword, password); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrMismatchedHashAndPassword
		}
		return err
	}
	return nil
}
