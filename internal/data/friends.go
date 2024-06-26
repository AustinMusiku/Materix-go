package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/AustinMusiku/Materix-go/internal/validator"
)

var ErrDuplicateFriendRequest = errors.New("friend request already pending or accepted")

type FriendRequest struct {
	Id                int    `json:"id"`
	SourceUserId      int    `json:"source_user_id,omitempty"`
	DestinationUserId int    `json:"destination_user_id,omitempty"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
	Version           int    `json:"version,omitempty"`
}

type DetailedFriendRequest struct {
	SourceUser      *User          `json:"source_user,omitempty"`
	DestinationUser *User          `json:"destination_user,omitempty"`
	RequestDetails  *FriendRequest `json:"request_details,omitempty"`
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

func (fp *FriendPairModel) GetFriend(id, friendId int) (*FriendRequest, error) {
	query := `
		SELECT id, source_user_id, destination_user_id, status, created_at, updated_at, version
		FROM friends
		WHERE 
			(friends.source_user_id = $1 AND friends.destination_user_id = $2) 
				OR 
			(friends.source_user_id = $2 AND friends.destination_user_id = $1)`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	var friend FriendRequest

	err := fp.db.QueryRowContext(ctx, query, id, friendId).Scan(
		&friend.Id,
		&friend.SourceUserId,
		&friend.DestinationUserId,
		&friend.Status,
		&friend.CreatedAt,
		&friend.UpdatedAt,
		&friend.Version,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &friend, nil
}

func (fp *FriendPairModel) GetFriendsFor(id int, filters Filters) ([]*User, Meta, error) {
	query := fmt.Sprintf(`
		SELECT 
			count(*) OVER(), users.id, users.uuid, users.name, users.email, users.avatar_url
		FROM friends
		INNER JOIN users
		ON 
			(users.id = friends.source_user_id OR users.id = friends.destination_user_id) AND users.id != $1
		WHERE 
			(friends.source_user_id = $1 OR friends.destination_user_id = $1) AND friends.status = 'accepted'
		ORDER BY friends.%s %s, users.id ASC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	rows, err := fp.db.QueryContext(ctx, query, id, filters.PageSize, filters.offset())
	if err != nil {
		return nil, Meta{}, err
	}
	defer rows.Close()

	friends := []*User{}
	totalRecords := 0

	for rows.Next() {
		var user User
		err := rows.Scan(
			&totalRecords,
			&user.Id,
			&user.Uuid,
			&user.Name,
			&user.Email,
			&user.AvatarUrl,
		)
		if err != nil {
			return nil, Meta{}, err
		}
		friends = append(friends, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, Meta{}, err
	}

	meta := calculateMeta(totalRecords, filters.Page, filters.PageSize)
	return friends, meta, nil
}

func (fp *FriendPairModel) GetSentFor(id int, filters Filters) ([]*DetailedFriendRequest, Meta, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), 
			friends.id, users.id as user_id, users.name as user_name, users.email, users.avatar_url, 
			friends.status, friends.created_at 
		FROM friends
		INNER JOIN users
		ON users.id = friends.destination_user_id
		WHERE source_user_id = $1 AND status = 'pending'
		ORDER BY friends.%s %s, users.id ASC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	rows, err := fp.db.QueryContext(ctx, query, id, filters.PageSize, filters.offset())
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Meta{}, ErrRecordNotFound
		default:
			return nil, Meta{}, err
		}
	}
	defer rows.Close()

	friendRequests := []*DetailedFriendRequest{}
	totalRecords := 0

	for rows.Next() {
		var du User
		var fr FriendRequest
		err := rows.Scan(
			&totalRecords,
			&fr.Id,
			&du.Id,
			&du.Name,
			&du.Email,
			&du.AvatarUrl,
			&fr.Status,
			&fr.CreatedAt,
		)
		if err != nil {
			return nil, Meta{}, err
		}
		friendRequests = append(friendRequests, &DetailedFriendRequest{
			DestinationUser: &du,
			RequestDetails: &FriendRequest{
				Id:        fr.Id,
				Status:    fr.Status,
				CreatedAt: fr.CreatedAt,
			},
		})
	}

	meta := calculateMeta(totalRecords, filters.Page, filters.PageSize)
	return friendRequests, meta, nil
}

func (fp *FriendPairModel) GetReceivedFor(id int, filters Filters) ([]*DetailedFriendRequest, Meta, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(),
			friends.id, users.id as user_id, users.name as user_name, users.email, users.avatar_url, friends.status, friends.created_at 
		FROM friends
		INNER JOIN users
		ON users.id = friends.source_user_id
		WHERE destination_user_id = $1 AND status = 'pending'
		ORDER BY friends.%s %s, users.id ASC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	rows, err := fp.db.QueryContext(ctx, query, id, filters.PageSize, filters.offset())
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Meta{}, ErrRecordNotFound
		default:
			return nil, Meta{}, err
		}
	}
	defer rows.Close()

	friendRequests := []*DetailedFriendRequest{}
	totalRecords := 0

	for rows.Next() {
		var su User
		var fr FriendRequest
		err := rows.Scan(
			&totalRecords,
			&fr.Id,
			&su.Id,
			&su.Name,
			&su.Email,
			&su.AvatarUrl,
			&fr.Status,
			&fr.CreatedAt,
		)
		if err != nil {
			return nil, Meta{}, err
		}
		friendRequests = append(friendRequests, &DetailedFriendRequest{
			SourceUser: &su,
			RequestDetails: &FriendRequest{
				Id:        fr.Id,
				Status:    fr.Status,
				CreatedAt: fr.CreatedAt,
			},
		})
	}

	meta := calculateMeta(totalRecords, filters.Page, filters.PageSize)
	return friendRequests, meta, nil
}

func (fp *FriendPairModel) Delete(friendRequest *FriendRequest) error {
	query := `
		DELETE FROM friends
		WHERE id = $1 AND version = $2`

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	result, err := fp.db.ExecContext(ctx, query, friendRequest.Id, friendRequest.Version)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrEditConflict
	}

	return nil
}

func (fp *FriendPairModel) SearchFor(id int, q string, filters Filters) ([]*User, Meta, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), 
			users.id, users.name, users.email, users.avatar_url 
		FROM friends
		INNER JOIN users
		ON (users.id = friends.source_user_id OR users.id = friends.destination_user_id) AND users.id != $1
		WHERE 
			(friends.source_user_id = $1 OR friends.destination_user_id = $1) AND friends.status = 'accepted'
			AND search @@ plainto_tsquery($2)
		ORDER BY ts_rank(search, plainto_tsquery($2)), friends.%s %s, users.id ASC
		LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), QueryTimeout)
	defer cancel()

	args := []interface{}{id, q, filters.PageSize, filters.offset()}

	rows, err := fp.db.QueryContext(ctx, query, args...)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Meta{}, ErrRecordNotFound
		default:
			return nil, Meta{}, err
		}
	}
	defer rows.Close()

	friends := []*User{}
	totalRecords := 0

	for rows.Next() {
		var u User
		err := rows.Scan(
			&totalRecords,
			&u.Id,
			&u.Name,
			&u.Email,
			&u.AvatarUrl,
		)
		if err != nil {
			return nil, Meta{}, err
		}
		friends = append(friends, &u)
	}

	meta := calculateMeta(totalRecords, filters.Page, filters.PageSize)
	return friends, meta, nil

}

func ValidateFriendPair(v *validator.Validator, friendRequest *FriendRequest) {
	v.Check(friendRequest.SourceUserId > 0, "source_user_id", "must be valid")
	v.Check(friendRequest.DestinationUserId > 0, "destination_user_id", "must be valid")
	v.Check(friendRequest.SourceUserId != friendRequest.DestinationUserId, "destination_user_id", "cannot send friend request to self")
	v.Check(friendRequest.Status == "pending", "status", "must be pending")
}
