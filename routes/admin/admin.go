package Admin

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"GoEatsapi/db"
	"GoEatsapi/utils"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func AdimnLogin(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Method not allowed", nil)
		return
	}
	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("ParseForm error:", err)
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}
	r.ParseForm()

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate
	if email == "" || password == "" {
		fmt.Println("Email or password is empty", email, password)
		utils.JSON(w, 400, false, "Email and password are required", nil)
		return
	}

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
		utils.JSON(w, 400, false, "Invalid email or password", nil)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		utils.JSON(w, 400, false, "Invalid email or password", nil)
		return
	}

	token, err := utils.GenerateToken(storedID, email)
	if err != nil {
		utils.JSON(w, 500, false, "Failed to generate token", nil)
		return
	}

	// ---------------------------------------------
	// 🎉 3. Successful login
	// ---------------------------------------------
	utils.JSON(w, 200, true, "Login successful", map[string]interface{}{
		"token": token,
	})
}

func Get_partners_list(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.JSON(w, 401, false, "Authorization header missing", nil)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		utils.JSON(w, 401, false, "Invalid token format", nil)
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
		utils.JSON(w, 500, false, "Failed to fetch partners", nil)
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

	partners := []Partner{}

	for rows.Next() {
		var p Partner
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Email, &p.Phone, &p.Status); err != nil {
			fmt.Println("Failed to scan partner:", err)
			utils.JSON(w, 500, false, "Failed to scan partner", nil)
			return
		}
		partners = append(partners, p)
	}

	utils.JSON(w, 200, true, "Partners list", partners)

}

func Get_restaurants_list(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.JSON(w, 401, false, "Authorization header missing", nil)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		utils.JSON(w, 401, false, "Invalid token format", nil)
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, restaurant_name, email, phone_number, status
		FROM restaurants
	`)
	if err != nil {
		fmt.Println("Failed to fetch restaurants:", err)
		utils.JSON(w, 500, false, "Failed to fetch restaurants", nil)
		return
	}
	defer rows.Close()

	type Restaurant struct {
		ID     int    `json:"id"`
		Name   string `json:"restaurant_name"`
		Email  string `json:"email"`
		Phone  string `json:"phone_number"`
		Status string `json:"status"`
	}

	var restaurants []Restaurant

	for rows.Next() {
		var r Restaurant
		if err := rows.Scan(&r.ID, &r.Name, &r.Email, &r.Phone, &r.Status); err != nil {
			fmt.Println("Failed to scan restaurant:", err)
			utils.JSON(w, 500, false, "Failed to scan restaurant", nil)
			return
		}
		restaurants = append(restaurants, r)
	}

	utils.JSON(w, 200, true, "Restaurants list", restaurants)
}

func Update_request_status_of_restaurant(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Method not allowed", nil)
		return
	}
	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("ParseForm error:", err)
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}
	r.ParseForm()

	id := r.FormValue("id")
	status := r.FormValue("status")

	// Validate
	if id == "" || status == "" {
		fmt.Println("id or status is empty", id, status)
		utils.JSON(w, 400, false, "id and status are required", nil)
		return
	}
	fmt.Println("Update request status of partner:", id, status)
	_, err = db.DB.Exec(`
		UPDATE restaurants
		SET status = ?
		WHERE id = ?
	`, status, id)
	if err != nil {
		fmt.Println("Failed to update request status of partner:", err)
		utils.JSON(w, 500, false, "Failed to update request status of partner", nil)
		return
	}

	utils.JSON(w, 200, true, "Request status updated", nil)
}

func Update_request_status_of_partner(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Method not allowed", nil)
		return
	}
	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("ParseForm error:", err)
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}
	r.ParseForm()

	id := r.FormValue("id")
	status := r.FormValue("status")

	// Validate
	if id == "" || status == "" {
		fmt.Println("id or status is empty", id, status)
		utils.JSON(w, 400, false, "id and status are required", nil)
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
		utils.JSON(w, 500, false, "Failed to update request status of partner", nil)
		return
	}

	utils.JSON(w, 200, true, "Request status updated", nil)
}

func GetPartnerDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Method check
	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Invalid request method", nil)
		return
	}

	// Auth check
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.JSON(w, 401, false, "Authorization header missing", nil)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		utils.JSON(w, 401, false, "Invalid token format", nil)
		return
	}

	// Request body
	var req struct {
		PartnerID uint64 `json:"partner_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, 400, false, "Invalid request body", nil)
		return
	}

	if req.PartnerID == 0 {
		utils.JSON(w, 400, false, "partner_id is required", nil)
		return
	}

	// Response struct
	type PartnerDetails struct {
		ID                   uint64 `json:"id"`
		FirstName            string `json:"first_name"`
		LastName             string `json:"last_name"`
		PrimaryMobile        string `json:"primary_mobile"`
		Email                string `json:"email"`
		Gender               string `json:"gender"`
		City                 string `json:"city"`
		FullAddress          string `json:"full_address"`
		LanguagesKnown       string `json:"languages_known"`
		Status               string `json:"status"`
		ProfileCompleted     int    `json:"profile_completed"`
		ProfilePhotoURL      string `json:"profile_photo_url"`
		DrivingLicenseURL    string `json:"driving_license_url"`
		DrivingLicenseNumber string `json:"driving_license_number"`
		DrivingLicenseExpire string `json:"driving_license_expire"`
		CreatedAt            string `json:"created_at"`
	}

	var partner PartnerDetails

	query := `
		SELECT 
			id,
			COALESCE(first_name,''),
			COALESCE(last_name,''),
			COALESCE(primary_mobile,''),
			COALESCE(email,''),
			COALESCE(gender,''),
			COALESCE(city,''),
			COALESCE(full_address,''),
			COALESCE(languages_known,''),
			status,
			profile_completed,
			COALESCE(profile_photo_url,''),
			COALESCE(driving_license_url,''),
			COALESCE(driving_license_number,''),
			COALESCE(driving_license_expire,''),
			created_at
		FROM delivery_partners
		WHERE id = ?
	`

	err := db.DB.QueryRow(query, req.PartnerID).Scan(
		&partner.ID,
		&partner.FirstName,
		&partner.LastName,
		&partner.PrimaryMobile,
		&partner.Email,
		&partner.Gender,
		&partner.City,
		&partner.FullAddress,
		&partner.LanguagesKnown,
		&partner.Status,
		&partner.ProfileCompleted,
		&partner.ProfilePhotoURL,
		&partner.DrivingLicenseURL,
		&partner.DrivingLicenseNumber,
		&partner.DrivingLicenseExpire,
		&partner.CreatedAt,
	)

	if err == sql.ErrNoRows {
		utils.JSON(w, 404, false, "Partner not found", nil)
		return
	} else if err != nil {
		fmt.Println("DB Error:", err)
		utils.JSON(w, 500, false, "Failed to fetch partner details", nil)
		return
	}

	utils.JSON(w, 200, true, "Partner details", partner)
}
