package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/AustinMusiku/Materix-go/internal/validator"
	"github.com/lib/pq"
)

type FreeTime struct {
	Id         int       `json:"id"`
	UserId     int       `json:"user_id"`
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
		freetime.UserId,
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
		&freetime.Id,
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
		_, err = tx.ExecContext(ctx, insertViewerQuery, freetime.Id, viewerID)
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

func (ft *FreeTimeModel) Get(freetimeId int) (*FreeTime, error) {
	query := `
		SELECT id, user_id, start_time, end_time, created_at, updated_at, tags, visibility, version
		FROM free_times
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	var freetime FreeTime

	err := ft.db.QueryRowContext(ctx, query, freetimeId).Scan(
		&freetime.Id,
		&freetime.UserId,
		&freetime.StartTime,
		&freetime.EndTime,
		&freetime.CreatedAt,
		&freetime.UpdatedAt,
		pq.Array(&freetime.Tags),
		&freetime.Visibility,
		&freetime.Version,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &freetime, nil
}

func (ft *FreeTimeModel) GetAllFor(userId int) ([]*FreeTime, error) {
	query := `
		SELECT id, user_id, start_time, end_time, created_at, updated_at, tags, visibility
		FROM free_times
		WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	rows, err := ft.db.QueryContext(ctx, query, userId)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	defer rows.Close()

	var freetimes []*FreeTime

	for rows.Next() {
		var ft FreeTime
		err = rows.Scan(
			&ft.Id,
			&ft.UserId,
			&ft.StartTime,
			&ft.EndTime,
			&ft.CreatedAt,
			&ft.UpdatedAt,
			pq.Array(&ft.Tags),
			&ft.Visibility,
		)
		if err != nil {
			return nil, err
		}

		freetimes = append(freetimes, &ft)
	}

	return freetimes, nil
}

func (ft *FreeTimeModel) Update(freetime *FreeTime) (*FreeTime, error) {
	query := `
		UPDATE free_times
		SET start_time = $1, end_time = $2, tags = $3, visibility = $4, updated_at = now(), version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	args := []interface{}{
		freetime.StartTime,
		freetime.EndTime,
		pq.Array(freetime.Tags),
		freetime.Visibility,
		freetime.Id,
		freetime.Version,
	}

	err := ft.db.QueryRowContext(ctx, query, args...).Scan(&freetime.Version)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return freetime, nil
}

func (ft *FreeTimeModel) Delete(freetime *FreeTime) error {
	query := `
		DELETE FROM free_times
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	_, err := ft.db.ExecContext(ctx, query, freetime.Id)
	if err != nil {
		return err
	}

	return nil
}

type FriendFreeTime struct {
	FreetimeId      int       `json:"id"`
	FriendId        int       `json:"user_id"`
	FriendName      string    `json:"name"`
	FriendEmail     string    `json:"email"`
	FriendAvatarUrl string    `json:"avatar_url"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Tags            []string  `json:"tags"`
}

func (ft *FreeTimeModel) GetAllForFriendsOf(userId int) ([]*FriendFreeTime, error) {
	query := `
		SELECT 
			ft.id as free_time_id, 
			u.id as friend_id, 
			u.name as friend_name, 
			u.email as friend_email,
			u.avatar_url,
			ft.start_time, 
			ft.end_time, 
			ft.tags
		FROM free_times ft
		INNER JOIN friends f 
		ON 
			(ft.user_id = f.source_user_id AND f.destination_user_id = $1)
			OR
			(ft.user_id = f.destination_user_id AND f.source_user_id = $1)
		INNER JOIN users u
		ON (ft.user_id = u.id)
		WHERE (f.source_user_id = $1 OR f.destination_user_id = $1)
		AND f.status = 'accepted'`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	rows, err := ft.db.QueryContext(ctx, query, userId)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	defer rows.Close()

	freetimes := []*FriendFreeTime{}

	for rows.Next() {
		var ft FriendFreeTime
		err = rows.Scan(
			&ft.FreetimeId,
			&ft.FriendId,
			&ft.FriendName,
			&ft.FriendEmail,
			&ft.FriendAvatarUrl,
			&ft.StartTime,
			&ft.EndTime,
			pq.Array(&ft.Tags),
		)
		if err != nil {
			return nil, err
		}

		freetimes = append(freetimes, &ft)
	}

	return freetimes, nil
}

func ValidateFreeTime(v *validator.Validator, freetime *FreeTime) bool {
	v.Check(freetime.UserId > 0, "user_id", "must be valid")
	v.Check(freetime.StartTime.After(time.Now()), "start_time", "must be in the future")
	v.Check(freetime.StartTime.Before(freetime.EndTime), "end_time", "must be after start time")
	v.Check(freetime.Visibility == "public" || freetime.Visibility == "private", "visibility", "must be either public or private")

	return v.Valid()
}
