package passw

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const DefaultCost = 10

var ErrMismatchedHashAndPassword = errors.New("mismatched hash and password")

func HashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
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
