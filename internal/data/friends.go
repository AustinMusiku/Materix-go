package data

import (
	"context"
	"database/sql"
	"errors"

	"github.com/AustinMusiku/Materix-go/internal/validator"
)

var ErrDuplicateFriendRequest = errors.New("friend request already pending or accepted")

type FriendRequest struct {
	Id                int    `json:"id"`
	SourceUserId      int    `json:"source_user_id"`
	DestinationUserId int    `json:"destination_user_id"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
	Version           int    `json:"version,omitempty"`
}

type FriendPairModel struct {
	db *sql.DB
}

func (fp *FriendPairModel) GetRequest(friendRequestId int) (*FriendRequest, error) {
	query := `
		SELECT id, source_user_id, destination_user_id, status, created_at, updated_at, version
		FROM friends
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	var f FriendRequest

	err := fp.db.QueryRowContext(ctx, query, friendRequestId).Scan(
		&f.Id,
		&f.SourceUserId,
		&f.DestinationUserId,
		&f.Status,
		&f.CreatedAt,
		&f.UpdatedAt,
		&f.Version,
	)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

func (fp *FriendPairModel) Insert(friendRequest *FriendRequest) error {
	query := `
		INSERT INTO friends (source_user_id, destination_user_id, status)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at, version`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	err := fp.db.QueryRowContext(ctx, query, friendRequest.SourceUserId, friendRequest.DestinationUserId, friendRequest.Status).Scan(
		&friendRequest.Id,
		&friendRequest.CreatedAt,
		&friendRequest.UpdatedAt,
		&friendRequest.Version,
	)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "unique_friendship_pair"`:
			return ErrDuplicateFriendRequest
		default:
			return err
		}
	}

	return nil
}

func (fp *FriendPairModel) Accept(friendRequest *FriendRequest) error {
	query := `
		UPDATE friends
		SET status = 'accepted', updated_at = now(), version = version + 1
		WHERE id = $1 AND version = $2
		RETURNING updated_at, version`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	err := fp.db.QueryRowContext(ctx, query, friendRequest.Id, friendRequest.Version).Scan(
		&friendRequest.UpdatedAt,
		&friendRequest.Version,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (fp *FriendPairModel) GetFriendsFor(id int) ([]*User, error) {
	query := `
		SELECT 
			users.id, users.uuid, users.name, users.email, users.avatar_url
		FROM friends
		INNER JOIN users
		ON 
			(users.id = friends.source_user_id OR users.id = friends.destination_user_id) AND users.id != $1
		WHERE 
			(friends.source_user_id = $1 OR friends.destination_user_id = $1) AND friends.status = 'accepted'`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	rows, err := fp.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []*User

	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.Id,
			&user.Uuid,
			&user.Name,
			&user.Email,
			&user.AvatarUrl,
		)
		if err != nil {
			return nil, err
		}
		friends = append(friends, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return friends, nil
}

// Reject
// Get Accepted (Aka Get Friends For)
// Get Pending (Aka Get Received Friend Requests)
// Get Sent (Aka Get Sent Friend Requests)
// Delete
// Get Friends For

func ValidateFriendPair(v *validator.Validator, friendRequest *FriendRequest) {
	v.Check(friendRequest.SourceUserId > 0, "source_user_id", "must be valid")
	v.Check(friendRequest.DestinationUserId > 0, "destination_user_id", "must be valid")
	v.Check(friendRequest.SourceUserId != friendRequest.DestinationUserId, "destination_user_id", "cannot send friend request to self")
	v.Check(friendRequest.Status == "pending", "status", "must be pending")
}
