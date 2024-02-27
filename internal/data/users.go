package data

import (
	"errors"

	"github.com/AustinMusiku/Materix-go/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Email      string   `json:"email"`
	FirstName  string   `json:"first_name"`
	LastName   string   `json:"last_name"`
	Password   password `json:"-"`
	Activated  bool     `json:"activated"`
	Avatar_url string   `json:"avatar"`
	Provider   string   `json:"provider"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

type password struct {
	plainText *string
	hash      []byte
}

func (p *password) Set(text string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), 12)
	if err != nil {
		return err
	}

	p.hash = hash
	p.plainText = &text

	return nil
}

func (p *password) Compare(text string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(text))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.FirstName != "", "firstName", "must be provided")
	v.Check(len(user.FirstName) <= 500, "firstName", "must not be more than 500 bytes long")

	v.Check(user.LastName != "", "lastName", "must be provided")
	v.Check(len(user.LastName) <= 500, "lastName", "must not be more than 500 bytes long")

	ValidateEmail(v, user.Email)

	if user.Password.plainText != nil {
		ValidatePasswordPlaintext(v, *user.Password.plainText)
	}
}
