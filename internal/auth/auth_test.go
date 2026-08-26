package auth

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	//arrange
	userUUID := uuid.New()
	secret := "123"
	var expDur time.Duration
	expDur = time.Minute * 20

	//act
	tokenString, err := MakeJWT(userUUID, secret, expDur)
	if err != nil {
		t.Fatalf("operation failed: %v", err)
	}
	newUserUUID, err := ValidateJWT(tokenString, secret)
	if err != nil {
		t.Fatalf("operation failed: %v", err)
	}

	//assert
	if newUserUUID != userUUID {
		t.Errorf("Test failed: got %v, want %v", newUserUUID, userUUID)

	}
}

func TestGetBearerToken(t *testing.T) {
	// arrange
	req, err := http.NewRequest("GET", "api/users", nil)
	if err != nil {
		t.Fatalf("operation failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/plaintext")
	req.Header.Set("Authorization", "Bearer 1234567890")
	// act with func
	tokenString, err := GetBearerToken(req.Header)
	if err != nil {
		t.Fatalf("operation failed: %v", err)
	}
	// act manually
	token := strings.Split(tokenString, " ")[1]
	// assert
	if tokenString != token {
		t.Fatalf("invalid token")
	}

}

/*
// Set standard or custom headers
req.Header.Set("Authorization", "Bearer your_token_here")
req.Header.Set("Content-Type", "application/json")

// For cookies, you can set the Cookie header directly or use AddCookie:
req.AddCookie(&http.Cookie{
    Name:  "session_token",
    Value: "xyz123",
})
*/
