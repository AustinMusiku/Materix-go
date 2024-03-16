package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/validator"
	"github.com/lib/pq"
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

func (ft *FreeTimeModel) Insert(freetime *FreeTime, viewers []int) (*FreeTime, error) {
	insertFreetimeQuery := `
		INSERT INTO free_times (user_id, start_time, end_time, tags, visibility)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at, version`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout+5*time.Second)
	defer cancel()

	args := []interface{}{
		freetime.UserID,
		freetime.StartTime,
		freetime.EndTime,
		pq.Array(freetime.Tags),
		freetime.Visibility,
	}

	tx, err := ft.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	err = tx.QueryRowContext(ctx, insertFreetimeQuery, args...).Scan(
		&freetime.ID,
		&freetime.CreatedAt,
		&freetime.UpdatedAt,
		&freetime.Version,
	)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	insertViewerQuery := `
			INSERT INTO free_time_viewer (free_time_id, user_id)
			VALUES ($1, $2)`

	for _, viewerID := range viewers {
		_, err = tx.ExecContext(ctx, insertViewerQuery, freetime.ID, viewerID)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return freetime, nil
}

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
