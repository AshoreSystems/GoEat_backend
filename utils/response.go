package utils

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/Backblaze/blazer/b2"
	"github.com/golang-jwt/jwt/v5"
	storage_go "github.com/supabase-community/storage-go"
)

var StorageClient *storage_go.Client
var B2Client *b2.Client

func InitB2() {
	ctx := context.Background()

	appKeyID := os.Getenv("B2_ACCOUNT_ID")
	appKey := os.Getenv("B2_APPLICATION_KEY_ID")

	if appKeyID == "" || appKey == "" {
		log.Fatalf("B2 APPLICATION KEYS missing. Check env variables.")
	}

	client, err := b2.NewClient(ctx, appKeyID, appKey)
	if err != nil {
		log.Fatalf("B2 init failed: %v", err)
	}

	B2Client = client
}

func InitSupabase() {
	StorageClient = storage_go.NewClient(
		os.Getenv("SUPABASE_URL"),
		os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		nil,
	)
	log.Println("Supabase initialized")
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
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

	json.NewEncoder(w).Encode(resp)
}
