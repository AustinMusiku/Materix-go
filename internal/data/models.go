package data

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
	QueryTimeout      = 5 * time.Second
)

type Models struct {
	Users     UserModel
	Friends   FriendPairModel
	FreeTimes FreeTimeModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Users:     UserModel{db: db},
		Friends:   FriendPairModel{db: db},
		FreeTimes: FreeTimeModel{db: db},
	}
}
