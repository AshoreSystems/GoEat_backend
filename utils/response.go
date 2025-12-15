package utils

import (
	"encoding/json"

	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func GenerateToken(loginID int, email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login_id": loginID,
		"email":    email,
		// "exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	return token.SignedString([]byte("goeats-v01"))
}

func ParseToken(tokenString string) (int, string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte("goeats-v01"), nil
	})

	if err != nil || !token.Valid {
		return 0, "", err
	}

	claims := token.Claims.(jwt.MapClaims)

	loginID := int(claims["login_id"].(float64))
	email := claims["email"].(string)

	return loginID, email, nil
}

func JSON(w http.ResponseWriter, status int, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := APIResponse{
		Success: success,
		Message: message,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
