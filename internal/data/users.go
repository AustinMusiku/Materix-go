package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

const QueryTimeout = 5 * time.Second

var ErrDuplicateEmail = errors.New("email already exists")

type User struct {
	Id         int      `json:"id"`
	Uuid       string   `json:"uuid"`
	Name       string   `json:"last_name"`
	Email      string   `json:"email"`
	Password   password `json:"-"`
	Avatar_url string   `json:"avatar"`
	Provider   string   `json:"provider"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
	Activated  bool     `json:"activated"`
	Version    int      `json:"-"`
}

type password struct {
	plainText *string
	hash      []byte
}

type UserModel struct {
	db *sql.DB
}

func NewUserModel(db *sql.DB) *UserModel {
	return &UserModel{
		db,
	}
}

func (u *UserModel) Insert(user *User) error {
	query := `
		INSERT INTO users (name, email, password, avatar_url, provider, activated)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, uuid, created_at, updated_at, version`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	args := []interface{}{
		user.Name,
		user.Email,
		user.Password.hash,
		user.Avatar_url,
		user.Provider,
		user.Activated,
	}

	err := u.db.QueryRowContext(ctx, query, args...).Scan(
		&user.Id,
		&user.Uuid,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Version,
	)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func (u *UserModel) GetById(id int) (*User, error) {
	query := `
		SELECT id, uuid, name, email, password, provider, avatar_url, created_at, updated_at, activated, version
		FROM users
		WHERE id = $1`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	err := u.db.QueryRowContext(ctx, query, id).Scan(
		&user.Id,
		&user.Uuid,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Provider,
		&user.Avatar_url,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return &user, ErrRecordNotFound
		default:
			return &user, err
		}
	}

	return &user, nil
}

func (u *UserModel) GetByName(name string) (*User, error) {
	query := `
		SELECT id, uuid, name, email, password, provider, avatar_url, created_at, updated_at, activated, version
		FROM users
		WHERE name = $1`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	err := u.db.QueryRowContext(ctx, query, name).Scan(
		&user.Id,
		&user.Uuid,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Provider,
		&user.Avatar_url,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return &user, ErrRecordNotFound
		default:
			return &user, err
		}
	}

	return &user, nil
}

func (u *UserModel) GetByEmail(email string) (*User, error) {
	query := `
		SELECT id, uuid, name, email, password, provider, avatar_url, created_at, updated_at, activated, version
		FROM users
		WHERE email = $1`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	err := u.db.QueryRowContext(ctx, query, email).Scan(
		&user.Id,
		&user.Uuid,
		&user.Name,
		&user.Email,
		&user.Password.hash,
		&user.Provider,
		&user.Avatar_url,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return &user, ErrRecordNotFound
		default:
			return &user, err
		}
	}

	return &user, nil
}

func (u *UserModel) Update(user *User) error {
	return nil
}

func (u *UserModel) Delete(id int) error {
	return nil
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
