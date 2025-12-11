package resto

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"database/sql"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
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

	// ---------------------------------------------
	// 🔍 1. Get user by email (no password validate)
	// ---------------------------------------------
	var storedID int
	var storedName string
	var storedHashedPassword string

	err = db.DB.QueryRow(`
    SELECT id, restaurant_name ,password
    FROM restaurants 
    WHERE email = ?
`, email).Scan(&storedID, &storedName, &storedHashedPassword)

	if err != nil {
		utils.JSON(w, 400, false, "Invalid email", nil)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHashedPassword), []byte(password))
	if err != nil {
		utils.JSON(w, 400, false, "Invalid email or password", nil)
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

func RestoRegister(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Method not allowed", nil)
		return
	}

	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}

	name := r.FormValue("name")
	restaurantName := r.FormValue("restaurant_name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	phoneNumber := r.FormValue("phone_number")
	businessAddress := r.FormValue("business_address")
	city := r.FormValue("city")
	state := r.FormValue("state")
	zipcode := r.FormValue("zipcode")
	latitude := r.FormValue("latitude")
	longitude := r.FormValue("longitude")

	// Validate
	if name == "" || email == "" || password == "" {
		utils.JSON(w, 400, false, "Name, email and password are required", nil)
		return
	}

	// 🔍 1. Check if email already exists
	var exists int
	err = db.DB.QueryRow("SELECT COUNT(*) FROM restaurants WHERE email = ?", email).Scan(&exists)
	if err != nil {
		utils.JSON(w, 500, false, "Database error", nil)
		return
	}

	if exists > 0 {
		utils.JSON(w, 400, false, "Can't use this email, already used", nil)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		utils.JSON(w, 500, false, "Failed to encrypt password", nil)
		return
	}

	// ---------------------------------------------
	// 📥 1. Insert user
	// ---------------------------------------------
	_, err = db.DB.Exec(`
    INSERT INTO restaurants (restaurant_name,business_owner_name, email,phone_number, password,business_address,city,state,zipcode,latitude,longitude)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, restaurantName, name, email, phoneNumber, hashed, businessAddress, city, state, zipcode, latitude, longitude)

	if err != nil {
		utils.JSON(w, 500, false, "Failed to register", nil)
		return
	}

	// ---------------------------------------------
	// 🎉 Successful registration
	// ---------------------------------------------
	utils.JSON(w, 200, true, "Registration successful", nil)
}

func RestoCheckDetails(w http.ResponseWriter, r *http.Request) {
	// -------------------------------
	// 1. Read Bearer Token
	// -------------------------------
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

	tokenString := parts[1]

	// -------------------------------
	// 2. Parse Token → loginID + email
	// -------------------------------
	loginID, email, err := utils.ParseToken(tokenString)
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 4. Fetch personal info from delivery_partners
	// -------------------------------
	var business_owner_name string
	var restaurantprofilereqStatus string

	err = db.DB.QueryRow(`
        SELECT business_owner_name, status
        FROM restaurants
        WHERE id = ?
    `, loginID).Scan(
		&business_owner_name,
		&restaurantprofilereqStatus,
	)

	if err != nil && err != sql.ErrNoRows {
		utils.JSON(w, 500, false, "Failed to fetch partner info", nil)
		return
	}

	utils.JSON(w, 200, true, "Success", map[string]interface{}{
		"email": email,
		// "email_verified":    emailVerified,
		"business_owner_name":          business_owner_name,
		"restaurant_profilereq_Status": restaurantprofilereqStatus,
		// ✔ ADD NEW FLAGS

	})
}
