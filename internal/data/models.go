package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Users   UserModel
	Friends FriendPairModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Users:   UserModel{db: db},
		Friends: FriendPairModel{db: db},
	}
}
