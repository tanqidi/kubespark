package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// DefaultJWTSecret is the default secret key for signing JWT tokens
	// In production, this should be overridden by KUBESPARK_JWT_SECRET environment variable
	DefaultJWTSecret = "kubespark-secret-key-change-in-production"
	// TokenExpiration is the expiration time for JWT tokens (24 hours)
	TokenExpiration = 24 * time.Hour
)

// getJWTSecret returns the JWT secret key from environment variable,
// or the default value if not set
func getJWTSecret() string {
	secret := os.Getenv("KUBESPARK_JWT_SECRET")
	if secret == "" {
		return DefaultJWTSecret
	}
	return secret
}

// getConfiguredUsername returns the configured username from environment variable,
// or the default value if not set
func getConfiguredUsername() string {
	username := os.Getenv("KUBESPARK_USERNAME")
	if username == "" {
		return "admin"
	}
	return username
}

// getConfiguredPassword returns the configured password from environment variable,
// or the default value if not set
func getConfiguredPassword() string {
	password := os.Getenv("KUBESPARK_PASSWORD")
	if password == "" {
		return "123456"
	}
	return password
}

// computePasswordVerifier computes HMAC-SHA256 of password using JWT secret as key
// This creates a verifier that cannot be brute-forced without knowing the JWT secret
// Even if attackers obtain the token and see the verifier, they cannot crack the password
// because they need the JWT secret to verify any password guess
func computePasswordVerifier(password string) string {
	secret := getJWTSecret()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(password))
	return hex.EncodeToString(mac.Sum(nil))
}

// Claims represents JWT claims
type Claims struct {
	Username         string `json:"username"`
	PasswordVerifier string `json:"password_verifier"` // HMAC-SHA256(secret, password) - cannot be brute-forced without secret
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for the given username and password
// The password verifier (HMAC-SHA256) is included in the token to prevent attackers
// who obtained JWT secret and username from generating valid tokens without knowing the password.
// Using HMAC instead of plain hash prevents offline brute-force attacks: even if attackers
// obtain the token and see the verifier, they cannot crack the password without the JWT secret.
func GenerateToken(username, password string) (string, error) {
	claims := Claims{
		Username:         username,
		PasswordVerifier: computePasswordVerifier(password),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "kubespark",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(getJWTSecret()))
}

// ValidateToken validates a JWT token and returns the claims
// It performs three-level validation:
// 1. Signature and expiration validation (JWT standard)
// 2. Username validation (must match configured username)
// 3. Password verifier validation (must match configured password verifier)
// This prevents attackers who obtained JWT secret and username from generating
// valid tokens without knowing the password.
// Using HMAC prevents offline brute-force: attackers cannot crack the password
// from the verifier without the JWT secret.
func ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(getJWTSecret()), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Secondary validation: verify that the username in token matches the configured username
		configuredUsername := getConfiguredUsername()
		if claims.Username != configuredUsername {
			//return nil, errors.New("token username does not match configured username")
			return nil, errors.New("invalid token")
		}

		// Tertiary validation: verify that the password verifier in token matches the configured password verifier
		// This prevents attackers who obtained JWT secret and username from generating valid tokens
		// without knowing the actual password.
		// Using HMAC-SHA256 instead of plain hash prevents offline brute-force attacks:
		// even if attackers see the verifier in the token, they cannot crack the password
		// without the JWT secret to verify their guesses.
		configuredPassword := getConfiguredPassword()
		expectedPasswordVerifier := computePasswordVerifier(configuredPassword)
		if claims.PasswordVerifier != expectedPasswordVerifier {
			//return nil, errors.New("token password verifier does not match configured password")
			return nil, errors.New("invalid token")
		}

		return claims, nil
	}

	return nil, errors.New("invalid token")
}
