package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/nxvrlxv/independent_cloud/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo         *repository.FileRepository
	jwtSecretKey []byte
}

var ErrInvalidCredentials = errors.New("invalid credentials")

type ctxkey string

const userIDkey ctxkey = "userID"

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDkey).(int64)
	return id, ok
}

func NewAuthService(repo *repository.FileRepository, jwtSecretKey []byte) *AuthService {
	return &AuthService{repo: repo, jwtSecretKey: jwtSecretKey}
}

func (service *AuthService) RegisterUser(ctx context.Context, username string, password string) (int64, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return -1, fmt.Errorf("error occured while hashing passwd^ %w", err)
	}

	id, err := service.repo.CreateUser(ctx, username, string(hash))
	if err != nil {
		return -1, fmt.Errorf("error in creating user %w", err)
	}

	return id, nil
}

func (service *AuthService) LoginUser(ctx context.Context, username string, password string) (int64, error) {
	user, err := service.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return -1, fmt.Errorf("Error! There is no this user: %w", ErrInvalidCredentials)
	}

	err1 := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err1 != nil {
		return -1, fmt.Errorf("Failed to login: %w", ErrInvalidCredentials)
	}

	return user.UserID, nil
}

func (service *AuthService) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "token is not set:", http.StatusUnauthorized)
			return
		}

		tokenString := authHeader.Value
		claims, err := service.CheckToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token:", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDkey, claims.UserID)
		next(w, r.WithContext(ctx))

	}
}
