package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Replace this with your secret key
// var JwtSecret = []byte("GOEATS_SECRET_KEY")
var JwtKey = []byte(os.Getenv("e4d1c038b9b3b00b1681d92c1310afb8880d0a61e99f004e96d750f37f3ab085"))

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"status":false,"message":"Authorization header missing"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"status":false,"message":"Invalid token format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// Validate Token
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return JwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, `{"status":false,"message":"Invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}
