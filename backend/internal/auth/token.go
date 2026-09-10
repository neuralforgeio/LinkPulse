package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/golang-jwt/jwt/v5"
)

func NewAccessToken(secret string, userID uuid.UUID, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(ttl)

	claims := jwt.RegisteredClaims{
		Subject: userID.String(),
		IssuedAt: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return  "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return signed, expiresAt, nil
}

func ParseAccessToken(secret, token string) (uuid.UUID, error) {
	parsed, err := jwt.Parse(token, func (*jwt.Token) (any, error)  {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !parsed.Valid {
		return uuid.Nil, errors.New("invalid access token")
	}

	sub, err := parsed.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, errors.New("token has no subject")
	}
	userID, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, errors.New("token subject is not a user id")
	}
	return userID, nil
}

func NewRefreshToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("read random bytes: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, HashRefreshToken(raw), nil
}

func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
