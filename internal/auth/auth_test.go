package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-for-unit-tests"

// токен, выпущенный сервисом, должен им же успешно проверяться
func TestTokenRoundTrip(t *testing.T) {
	s := NewAuthService(nil, []byte(testSecret))

	token, err := s.GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims, err := s.CheckToken(token)
	if err != nil {
		t.Fatalf("CheckToken: %v", err)
	}
	if claims.UserID != 42 {
		t.Fatalf("UserID: получили %d, ожидали 42", claims.UserID)
	}
}

// токен, подписанный чужим секретом, проходить не должен
func TestCheckTokenWrongSecret(t *testing.T) {
	issuer := NewAuthService(nil, []byte(testSecret))
	verifier := NewAuthService(nil, []byte("completely-different-secret"))

	token, err := issuer.GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	if _, err := verifier.CheckToken(token); err == nil {
		t.Fatal("токен прошёл проверку с чужим секретом — подпись не проверяется")
	}
}

// протухший токен должен отвергаться
func TestCheckTokenExpired(t *testing.T) {
	s := NewAuthService(nil, []byte(testSecret))

	claims := CustomClaims{
		UserID: 42,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("подпись тестового токена: %v", err)
	}

	if _, err := s.CheckToken(expired); err == nil {
		t.Fatal("протухший токен прошёл проверку — exp не проверяется")
	}
}

// подделка с alg=none не должна проходить (проверка t.Method в keyFunc)
func TestCheckTokenNoneAlgorithm(t *testing.T) {
	s := NewAuthService(nil, []byte(testSecret))

	claims := CustomClaims{
		UserID: 1337,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	forged, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("подпись тестового токена: %v", err)
	}

	if _, err := s.CheckToken(forged); err == nil {
		t.Fatal("токен с alg=none прошёл проверку — алгоритм подписи не проверяется")
	}
}

func TestAuthMiddleware(t *testing.T) {
	s := NewAuthService(nil, []byte(testSecret))

	validToken, err := s.GenerateToken(99)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	tests := []struct {
		name       string
		cookie     *http.Cookie
		wantStatus int
	}{
		{"без cookie", nil, http.StatusUnauthorized},
		{"мусор вместо токена", &http.Cookie{Name: "token", Value: "not-a-jwt"}, http.StatusUnauthorized},
		{"чужое имя cookie", &http.Cookie{Name: "session", Value: validToken}, http.StatusUnauthorized},
		{"валидный токен", &http.Cookie{Name: "token", Value: validToken}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUserID int64
			var called bool

			next := func(w http.ResponseWriter, r *http.Request) {
				called = true
				id, ok := UserIDFromContext(r.Context())
				if !ok {
					t.Error("userID отсутствует в контексте")
				}
				gotUserID = id
				w.WriteHeader(http.StatusOK)
			}

			req := httptest.NewRequest(http.MethodGet, "/folders", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()

			s.AuthMiddleware(next)(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("статус: получили %d, ожидали %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusUnauthorized && called {
				t.Fatal("next вызван несмотря на отказ авторизации")
			}
			if tt.wantStatus == http.StatusOK && gotUserID != 99 {
				t.Fatalf("userID из контекста: получили %d, ожидали 99", gotUserID)
			}
		})
	}
}