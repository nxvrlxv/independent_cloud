package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	UserID int64
	jwt.RegisteredClaims
}

func (service *AuthService) GenerateToken(userID int64) (string, error) {
	if string(service.jwtSecretKey) == "" {
		return "", fmt.Errorf("JWT_SECRET is not set")
	}
	customClaims := CustomClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)
	tokenString, err := token.SignedString(service.jwtSecretKey)

	if err != nil {
		return "", fmt.Errorf("Invalid token: %w", err)
	}

	return tokenString, nil

}

func (service *AuthService) CheckToken(tokenString string) (*CustomClaims, error) {
	var MyClaims CustomClaims

	_, err := jwt.ParseWithClaims(tokenString, &MyClaims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}

		return service.jwtSecretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("Not valid token: %w", err)
	}

	return &MyClaims, nil
}
