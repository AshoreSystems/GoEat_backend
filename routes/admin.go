package routes

import (
	"net/http"

	"GoEatsapi/db"
	"fmt"
	"strings"
)

func AdimnLogin(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		JSON(w, 405, false, "Method not allowed", nil)
		return
	}
	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("ParseForm error:", err)
		JSON(w, 400, false, "Invalid form data", nil)
		return
	}
	r.ParseForm()

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate
	if email == "" || password == "" {
		fmt.Println("Email or password is empty", email, password)
		JSON(w, 400, false, "Email and password are required", nil)
		return
	}

	fmt.Println("Login request:", email, password)

	// ---------------------------------------------
	// 🔍 1. Find user by email
	// ---------------------------------------------
	var storedID int
	var storedName string
	var storedPassword string

	err = db.DB.QueryRow(`
		SELECT id, name, password
		FROM login
		WHERE email = ?
	`, email).Scan(&storedID, &storedName, &storedPassword)

	if err != nil {
		JSON(w, 400, false, "Invalid email or password", nil)
		return
	}

	if storedPassword != password {
		JSON(w, 400, false, "wrong password", nil)
		return
	}

	token, err := GenerateToken(storedID, email)
	if err != nil {
		JSON(w, 500, false, "Failed to generate token", nil)
		return
	}

	// ---------------------------------------------
	// 🎉 3. Successful login
	// ---------------------------------------------
	JSON(w, 200, true, "Login successful", map[string]interface{}{
		"id":    storedID,
		"name":  storedName,
		"email": email,
		"token": token,
	})
}

func Get_partners_list(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		JSON(w, 401, false, "Authorization header missing", nil)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		JSON(w, 401, false, "Invalid token format", nil)
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, COALESCE(first_name, ''),
		COALESCE(last_name, ''),
		COALESCE(email, ''),
		COALESCE(primary_mobile, ''),
		COALESCE(status, '')
		FROM delivery_partners 
		WHERE profile_completed = 1
	`)
	if err != nil {
		fmt.Println("Failed to fetch partners:", err)
		JSON(w, 500, false, "Failed to fetch partners", nil)
		return
	}
	defer rows.Close()

	type Partner struct {
		ID        int    `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Phone     string `json:"primary_mobile"`
		Status    string `json:"status"`
	}

	var partners []Partner

	for rows.Next() {
		var p Partner
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Email, &p.Phone, &p.Status); err != nil {
			fmt.Println("Failed to scan partner:", err)
			JSON(w, 500, false, "Failed to scan partner", nil)
			return
		}
		partners = append(partners, p)
	}

	JSON(w, 200, true, "Partners list", partners)

}

func Update_request_status_of_partner(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		JSON(w, 405, false, "Method not allowed", nil)
		return
	}
	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("ParseForm error:", err)
		JSON(w, 400, false, "Invalid form data", nil)
		return
	}
	r.ParseForm()

	id := r.FormValue("id")
	status := r.FormValue("status")

	// Validate
	if id == "" || status == "" {
		fmt.Println("id or status is empty", id, status)
		JSON(w, 400, false, "id and status are required", nil)
		return
	}
	fmt.Println("Update request status of partner:", id, status)
	_, err = db.DB.Exec(`
		UPDATE delivery_partners
		SET status = ?
		WHERE id = ?
	`, status, id)
	if err != nil {
		fmt.Println("Failed to update request status of partner:", err)
		JSON(w, 500, false, "Failed to update request status of partner", nil)
		return
	}

	JSON(w, 200, true, "Request status updated", nil)
}
