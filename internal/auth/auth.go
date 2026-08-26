package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func CheckPasswordHash(password string, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return match, err
	}
	return match, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeaderValue := headers.Get("Authorization")
	fmt.Printf("authHeader = %v", authHeaderValue)
	if authHeaderValue == "" {
		return " ", fmt.Errorf("header doesn't exist")
	}
	splitAuthHeader := strings.Split(authHeaderValue, " ")

	tokenString := splitAuthHeader[1]
	return tokenString, nil
}

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}

	return hash, nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {

	id := userID.String()
	method := jwt.SigningMethodHS256
	nowUTC := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		IssuedAt:  jwt.NewNumericDate(nowUTC),
		ExpiresAt: jwt.NewNumericDate(nowUTC.Add(expiresIn)),
		Subject:   id,
	}
	newToken := jwt.NewWithClaims(method, claims)
	secretByte := []byte(tokenSecret)
	signedString, err := newToken.SignedString(secretByte)
	if err != nil {
		return "", err
	}

	return signedString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {

	keyFunc := func(token *jwt.Token) (any, error) {
		secretByte := []byte(tokenSecret)
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("wrong signing method")
		}
		return secretByte, nil
	}

	var returnUUID uuid.UUID

	var claims jwt.RegisteredClaims

	token, err := jwt.ParseWithClaims(tokenString, &claims, keyFunc)
	if err != nil {
		return returnUUID, fmt.Errorf("error: token is invalid or has expired: %w", err)
	}

	id, err := token.Claims.GetSubject()
	if err != nil {
		return returnUUID, fmt.Errorf("error getting user's id from Subject field: %w", err)
	}

	returnUUID, err = uuid.Parse(id)
	if err != nil {
		return returnUUID, fmt.Errorf("error parsing string to uuid: %w", err)
	}

	return returnUUID, nil
}
