package data

import (
	"database/sql"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/validator"
)

type FreeTime struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Visibility string    `json:"visibility,omitempty"`
	Version    int       `json:"version,omitempty"`
}

type FreeTimeModel struct {
	db *sql.DB
}

func NewFreeTimeModel(db *sql.DB) *FreeTimeModel {
	return &FreeTimeModel{db: db}
}

//	func (ft *FreeTimeModel) Insert(freetime *FreeTime, viewers []int) (*FreeTime, error) {
//		return freetime, nil
//	}
//
//	func (ft *FreeTimeModel) Get(freetimeId int) (*FreeTime, error) {
//		return freetime, nil
//	}
//
//	func (ft *FreeTimeModel) Update(freetime *FreeTime) (*FreeTime, error) {
//		return freetime, nil
//	}
//
//	func (ft *FreeTimeModel) Delete(freetime *FreeTime) (*FreeTime, error) {
//		return freetime, nil
//	}
func ValidateFreeTime(v *validator.Validator, freetime *FreeTime) bool {
	v.Check(freetime.UserID > 0, "user_id", "must be valid")
	v.Check(freetime.StartTime.After(time.Now()), "start_time", "must be in the future")
	v.Check(freetime.StartTime.Before(freetime.EndTime), "end_time", "must be after start time")
	v.Check(freetime.Visibility == "public" || freetime.Visibility == "private", "visibility", "must be either public or private")

	return v.Valid()
}
