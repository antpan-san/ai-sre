package utils

import (
	"errors"
	"time"

	"ft-backend/common/logger"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims contains the custom claims for JWT tokens.
// UserID is stored as a string (UUID) to support PostgreSQL UUID primary keys.
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed JWT access token.
func GenerateAccessToken(userID, username, email, role, secretKey string, expiresIn int) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresIn) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// GenerateRefreshToken creates a signed JWT refresh token.
func GenerateRefreshToken(userID, username, secretKey string, expiresIn int) (string, error) {
	claims := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresIn) * time.Minute)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Subject:   username,
		ID:        userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}

// ValidateToken parses and validates a JWT token string.
func ValidateToken(tokenString, secretKey string) (*JWTClaims, error) {
	logger.Debug("Validating JWT token")

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil {
		logger.Error("Token parse error: %v", err)
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		logger.Debug("JWT claims: UserID=%s Username=%s Role=%s", claims.UserID, claims.Username, claims.Role)
		return claims, nil
	}

	logger.Warn("Invalid token claims")
	return nil, errors.New("invalid token")
}

// ExtractUserIDFromToken extracts the user ID (UUID string) from a token.
func ExtractUserIDFromToken(tokenString, secretKey string) (string, error) {
	claims, err := ValidateToken(tokenString, secretKey)
	if err != nil {
		return "", err
	}

	return claims.UserID, nil
}
