package resto

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"net/http"
)

func RestoLogin(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	// if r.Method != http.MethodPost {
	// 	utils.JSON(w, 405, false, "Method not allowed", nil)
	// 	return
	// }

	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate
	if email == "" || password == "" {
		utils.JSON(w, 400, false, "Email and password are required", nil)
		return
	}

	// ⭐ STATIC PASSWORD CHECK
	if password != "123456" {
		utils.JSON(w, 400, false, "Wrong password", nil)
		return
	}

	// ---------------------------------------------
	// 🔍 1. Get user by email (no password validate)
	// ---------------------------------------------
	var storedID int
	var storedName string

	err = db.DB.QueryRow(`
    SELECT id, restaurant_name 
    FROM restaurants 
    WHERE email = ?
`, email).Scan(&storedID, &storedName)

	if err != nil {
		utils.JSON(w, 400, false, "Invalid email", nil)
		return
	}

	// Generate token
	token, err := utils.GenerateToken(storedID, email)
	if err != nil {
		utils.JSON(w, 500, false, "Failed to generate token", nil)
		return
	}

	// ---------------------------------------------
	// 🎉 Successful login
	// ---------------------------------------------
	utils.JSON(w, 200, true, "Login successful", map[string]interface{}{
		"id":    storedID,
		"name":  storedName,
		"email": email,
		"token": token,
	})
}
