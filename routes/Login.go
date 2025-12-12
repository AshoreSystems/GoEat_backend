package routes

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"GoEatsapi/db"
	"GoEatsapi/utils"
	"fmt"
	"math/rand"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// var smtpCfg = mailer.Config{
// 	APIKey: "api-F5E04BD8DD2843099F47E0D3D4FB16AF",
// 	From:   "kadir.pathan@aviontechnology.us",
// }

func JSON(w http.ResponseWriter, status int, success bool, msg string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"message": msg,
		"data":    data,
	})
}

func GenerateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(999999))
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		JSON(w, 405, false, "Method not allowed", nil)
		return
	}
	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {

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

	// ---------------------------------------------
	// 🔍 1. Find user by email
	// ---------------------------------------------
	var storedID int
	var storedName string
	var storedHashedPassword string

	err = db.DB.QueryRow(`
		SELECT id, name, password
		FROM login
		WHERE email = ?
	`, email).Scan(&storedID, &storedName, &storedHashedPassword)

	if err != nil {
		JSON(w, 400, false, "Invalid email or password", nil)
		return
	}

	// ---------------------------------------------
	// 🔐 2. Compare hashed password
	// ---------------------------------------------
	err = bcrypt.CompareHashAndPassword([]byte(storedHashedPassword), []byte(password))
	if err != nil {
		JSON(w, 400, false, "Invalid email or password", nil)
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

func RegisterHandler(w http.ResponseWriter, r *http.Request) {

	// Parse form-data
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		JSON(w, 400, false, "Invalid form data", nil)
		return
	}

	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	userType := strings.ToLower(strings.TrimSpace(r.FormValue("type"))) // partner or user

	if email == "" {
		JSON(w, 400, false, "Email is required", nil)
		return
	}

	if userType == "" {
		userType = "user" // default
	}

	otp := GenerateOTP()

	// Send email in a goroutine (non-blocking)

	//-----------------------------------
	// CHECK IF LOGIN ALREADY EXISTS
	//-----------------------------------
	var loginID int
	err := db.DB.QueryRow(`
		SELECT id FROM login WHERE email = ?
	`, email).Scan(&loginID)

	// ❗ ERROR CHECK: If DB returned error but not "no rows", stop here
	if err != nil && err != sql.ErrNoRows {
		JSON(w, 500, false, "Database error", nil)
		return
	}

	//-----------------------------------
	// CASE 1 → LOGIN EXISTS → UPDATE OTP
	//-----------------------------------
	if err == nil { // meaning record FOUND

		_, err := db.DB.Exec(`
			UPDATE login
			SET verification_code = ?, updated_at = NOW()
			WHERE id = ?
		`, otp, loginID)

		if err != nil {
			JSON(w, 500, false, "Failed to update OTP", nil)
			return
		}

		// If partner registration → ensure delivery_partners entry
		if userType == "partner" {
			_, err := db.DB.Exec(`
				INSERT INTO delivery_partners (login_id, email, status, created_at, updated_at)
				VALUES (?, ?, 'pending', NOW(), NOW())
				ON DUPLICATE KEY UPDATE email = email
			`, loginID, email)

			if err != nil {
				fmt.Println("PARTNER INSERT ERROR:", err)
				JSON(w, 500, false, "Failed to update partner record", nil)
				return
			}
		}

		// send OTP email

		// err = mailer.SendOTPviaSMTP(sendgridKey, fromEmail, email, subject, body)
		// if err != nil {
		// 	fmt.Println("SMTP ERROR:", err)
		// 	JSON(w, 500, false, "Failed to send OTP email", nil)
		// 	return
		// }

		JSON(w, 200, true, "OTP sent", map[string]string{"otp": otp})
		return
	}

	//-----------------------------------
	// CASE 2 → NEW LOGIN → INSERT LOGIN
	//-----------------------------------

	res, err := db.DB.Exec(`
		INSERT INTO login (email, type, status, verification_code, email_verified, created_at, updated_at)
		VALUES (?, ?, 'pending', ?, 0, NOW(), NOW())
	`, email, userType, otp)

	if err != nil {
		fmt.Println("SQL ERROR:", err)
		JSON(w, 500, false, "Failed to create login record", nil)
		return
	}

	loginID64, _ := res.LastInsertId()
	loginID = int(loginID64)

	//-----------------------------------
	// INSERT delivery_partners IF PARTNER
	//-----------------------------------
	if userType == "partner" {

		_, err := db.DB.Exec(`
			INSERT INTO delivery_partners (login_id, email, status, created_at, updated_at)
			VALUES (?, ?, 'pending', NOW(), NOW())
		`, loginID, email)

		if err != nil {
			fmt.Println("PARTNER INSERT ERROR:", err)
			JSON(w, 500, false, "Failed to create partner record", nil)
			return
		}
	}

	// send OTP email
	// err = mailer.SendEmail(smtpCfg, []string{email}, subject, body)
	// if err != nil {
	// 	fmt.Println("Failed to send OTP email:", err)
	// 	JSON(w, 500, false, "Failed to send OTP email", nil)
	// 	return
	// }

	JSON(w, 200, true, "Registration successful. OTP sent.", map[string]string{"otp": otp})
}

func GenerateToken(loginID int, email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login_id": loginID,
		"email":    email,
		// "exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	return token.SignedString([]byte("goeats-v01"))
}

func VerifyEmailHandler(w http.ResponseWriter, r *http.Request) {

	r.ParseMultipartForm(10 << 20)
	r.ParseForm()

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	otp := strings.TrimSpace(r.FormValue("otp")) // DO NOT lowercase OTP

	if email == "" {
		JSON(w, 400, false, "Email is required", nil)
		return
	}

	var loginID int
	var emailVerified int
	var dbOTP sql.NullString

	err := db.DB.QueryRow(`
        SELECT id, verification_code, email_verified
        FROM login
        WHERE email = ?
    `, email).Scan(&loginID, &dbOTP, &emailVerified)

	if err != nil {
		JSON(w, 400, false, "Email not found", nil)
		return
	}

	// -------------------------------------------------------------
	// Fetch partner profile status from delivery_partners table
	// -------------------------------------------------------------
	var partnerStatus string
	err = db.DB.QueryRow(`
        SELECT status
        FROM delivery_partners
        WHERE login_id = ?
    `, loginID).Scan(&partnerStatus)

	if err != nil && err != sql.ErrNoRows {
		JSON(w, 500, false, "Failed to fetch partner status", nil)
		return
	}
	// -------------------------------
	// If already verified but OTP entered → still validate OTP
	// -------------------------------
	if emailVerified == 1 {

		if otp != "" { // User entered OTP, must validate it
			if !dbOTP.Valid || dbOTP.String != otp {
				JSON(w, 400, false, "Invalid OTP", nil)
				return
			}
		}

		token, err := GenerateToken(loginID, email)
		if err != nil {
			JSON(w, 500, false, "Token generation failed", nil)
			return
		}

		JSON(w, 200, true, "Email already verified", map[string]string{
			"token":  token,
			"status": partnerStatus,
		})
		return
	}

	// -------------------------------
	// Not verified → normal OTP check
	// -------------------------------
	if otp == "" {
		JSON(w, 400, false, "OTP is required", nil)
		return
	}

	if !dbOTP.Valid {
		JSON(w, 400, false, "OTP not generated", nil)
		return
	}

	if dbOTP.String != otp {
		JSON(w, 400, false, "Invalid OTP", nil)
		return
	}

	// Update verification
	_, err = db.DB.Exec(`
        UPDATE login
        SET email_verified = 1,
            verification_code = NULL,
            status = 'active',
            updated_at = NOW()
        WHERE id = ?
    `, loginID)

	if err != nil {
		JSON(w, 500, false, "Failed to update login", nil)
		return
	}

	// Generate token
	token, err := GenerateToken(loginID, email)
	if err != nil {
		JSON(w, 500, false, "Token generation failed", nil)
		return
	}

	JSON(w, 200, true, "Email verified successfully", map[string]string{
		"token":  token,
		"status": partnerStatus,
	})
}

func GetEmailStatusHandler(w http.ResponseWriter, r *http.Request) {

	// -------------------------------
	// 1. Read Bearer Token
	// -------------------------------
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

	tokenString := parts[1]

	// -------------------------------
	// 2. Parse Token → loginID + email
	// -------------------------------
	loginID, email, err := utils.ParseToken(tokenString)
	if err != nil {
		JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 3. Fetch email_verified from login table + password
	// -------------------------------
	var emailVerified int
	var password sql.NullString

	err = db.DB.QueryRow(`
        SELECT email_verified, password
        FROM login
        WHERE id = ?
    `, loginID).Scan(&emailVerified, &password)

	if err != nil {
		JSON(w, 500, false, "Failed to fetch login info", nil)
		return
	}

	// -------------------------------
	// 4. Fetch personal info from delivery_partners
	// -------------------------------
	var firstName, lastName, dateOfBirth, primaryMobile, gender, profilePhoto sql.NullString
	var profileCompleted int
	var partnerprofilereqStatus string

	err = db.DB.QueryRow(`
        SELECT first_name, last_name, date_of_birth, primary_mobile, gender, profile_photo_url,profile_completed,status
        FROM delivery_partners
        WHERE login_id = ?
    `, loginID).Scan(
		&firstName,
		&lastName,
		&dateOfBirth,
		&primaryMobile,
		&gender,
		&profilePhoto,
		&profileCompleted,
		&partnerprofilereqStatus,
	)

	if err != nil && err != sql.ErrNoRows {
		JSON(w, 500, false, "Failed to fetch partner info", nil)
		return
	}

	// -------------------------------
	// 5. STEP COMPLETION LOGIC
	// -------------------------------

	// Step 1 → Basic profile details
	step1Completed := firstName.String != "" &&
		lastName.String != "" &&
		dateOfBirth.String != "" &&
		primaryMobile.String != "" &&
		gender.String != ""

	// Step 2 → Password set in login table
	step2Completed := password.String != ""

	// Step 3 → Profile photo uploaded
	step3Completed := profilePhoto.String != ""

	// -------------------------------
	// 6. Final Response
	// -------------------------------
	JSON(w, 200, true, "Success", map[string]interface{}{
		"email":             email,
		"email_verified":    emailVerified,
		"first_name":        firstName.String,
		"last_name":         lastName.String,
		"date_of_birth":     dateOfBirth.String,
		"primary_mobile":    primaryMobile.String,
		"profile_photo_url": profilePhoto.String,
		"gender":            gender.String,
		"profile_completed": profileCompleted,

		// ✔ ADD NEW FLAGS
		"step_1_completed": step1Completed,
		"step_2_completed": step2Completed,
		"step_3_completed": step3Completed,

		"partner_profilereq_Status": partnerprofilereqStatus,
	})
}
