package data

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	Uuid      string `json:"uuid"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
	Provider  string `json:"provider"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	StandardClaims
}

type StandardClaims struct {
	Iat jwt.NumericDate `json:"iat"`
	Exp jwt.NumericDate `json:"exp"`
	Iss string          `json:"iss"`
	Aud string          `json:"aud"`
	Nbf jwt.NumericDate `json:"nbf"`
	Sub string          `json:"sub"`
}

func NewTokenPair(user User) (map[string]string, error) {
	claims := UserClaims{
		Uuid:      user.Uuid,
		Username:  user.Name,
		Email:     user.Email,
		AvatarUrl: user.Avatar_url,
		Provider:  user.Provider,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		StandardClaims: StandardClaims{
			Iat: jwt.NumericDate{Time: time.Now()},
			Exp: jwt.NumericDate{Time: time.Now().Add(time.Minute * 30)},
			Iss: "https://materix.app",
			Aud: "materix",
			Sub: fmt.Sprintf("%d", user.Id),
			Nbf: jwt.NumericDate{Time: time.Now()},
		},
	}

	at, err := NewAccessToken(claims)
	if err != nil {
		return map[string]string{}, err
	}

	rt, err := NewRefreshToken(claims.StandardClaims)
	if err != nil {
		return map[string]string{}, err
	}

	return map[string]string{
		"access_token":  at,
		"refresh_token": rt,
	}, nil
}

func NewAccessToken(claims UserClaims) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uuid":      claims.Uuid,
		"username":  claims.Username,
		"email":     claims.Email,
		"avatarUrl": claims.AvatarUrl,
		"provider":  claims.Provider,
		"createdAt": claims.CreatedAt,
		"updatedAt": claims.UpdatedAt,
		// std claims
		"sub": claims.Sub,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute * 30).Unix(),
		"aud": "materix",
		"iss": "https://materix.app",
		"nbf": time.Now().Unix(),
	})

	at, err := t.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", fmt.Errorf("error signing access token: %w", err)
	}

	return at, nil
}

func NewRefreshToken(claims StandardClaims) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": claims.Iat,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
		"aud": "materix",
		"iss": "https://materix.app",
		"sub": claims.Sub,
		"nbf": claims.Nbf,
	})

	rt, err := t.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", fmt.Errorf("error signing refresh token: %w", err)
	}

	return rt, nil
}

func ParseAccessToken(at string) (*UserClaims, error) {
	pat, err := jwt.ParseWithClaims(at, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := pat.Claims.(*UserClaims)
	if !ok {
		return nil, errors.New("invalid access token")
	}

	return claims, nil
}

func ParseRefreshToken(rt string) (*StandardClaims, error) {
	rat, err := jwt.ParseWithClaims(rt, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := rat.Claims.(*UserClaims)
	if !ok {
		return nil, errors.New("invalid refresh token")
	}

	return &StandardClaims{
		Iat: claims.Iat,
		Exp: claims.Exp,
		Iss: claims.Iss,
		Aud: claims.Aud,
		Nbf: claims.Nbf,
		Sub: claims.Sub,
	}, nil
}

func (u *UserClaims) GetExpirationTime() (*jwt.NumericDate, error) {
	return &u.Exp, nil
}

func (u *UserClaims) GetIssuedAt() (*jwt.NumericDate, error) {
	return &u.Iat, nil
}

func (u *UserClaims) GetNotBefore() (*jwt.NumericDate, error) {
	return &u.Nbf, nil

}

func (u *UserClaims) GetIssuer() (string, error) {
	return u.Iss, nil

}

func (u *UserClaims) GetSubject() (string, error) {
	return u.Sub, nil

}

func (u *UserClaims) GetAudience() (jwt.ClaimStrings, error) {
	return jwt.ClaimStrings{u.Aud}, nil
}
