package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {
    expiresIn := time.Hour
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, 
		jwt.RegisteredClaims{
			Issuer: "chirpy",
			IssuedAt: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
			Subject: userID.String(),
		},
	)

	return t.SignedString([]byte (tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return []byte(tokenSecret), nil
    })
    if err != nil {
        return uuid.UUID{}, err
    }

    if !token.Valid {
        return uuid.UUID{}, errors.New("invalid token")
    }

    // Extract Subject → UUID
    userID, err := uuid.Parse(claims.Subject)
    if err != nil {
        return uuid.UUID{}, err
    }

    return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
    bearer := headers.Get("Authorization")
    bearer, found := strings.CutPrefix(bearer, "Bearer ")
    if bearer == "" {
        return "", errors.New("JWT not found")
    }
    if !found {
        return "", errors.New("wrong JWT prefix")
    }

    return bearer, nil
}