package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/alvesantos/financas-backend/internal/domain"
)

// JWTIssuer implementa domain.TokenIssuer com HS256.
type JWTIssuer struct {
	secret     []byte
	issuer     string
	expiration time.Duration
}

var _ domain.TokenIssuer = (*JWTIssuer)(nil)

func NewJWTIssuer(secret, issuer string, expiration time.Duration) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), issuer: issuer, expiration: expiration}
}

func (s *JWTIssuer) Issue(user *domain.User) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.expiration)

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(user.ID, 10),
		Issuer:    s.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, domain.ErrInternal.Wrap(err)
	}

	return signed, expiresAt, nil
}

func (s *JWTIssuer) Verify(tokenString string) (int64, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", t.Header["alg"])
		}
		return s.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.issuer),
	)
	if err != nil || !token.Valid {
		return 0, domain.ErrUnauthenticated.Wrap(err)
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, domain.ErrUnauthenticated.Wrap(err)
	}

	return userID, nil
}
