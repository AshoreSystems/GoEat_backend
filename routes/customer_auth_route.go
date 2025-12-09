package routes

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"GoEatsapi/db"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	FullName     string `json:"full_name"`
	Password     string `json:"password"`
	Email        string `json:"email"`
	CountryCode  string `json:"country_code"`
	PhoneNumber  string `json:"phone_number"`
	Dob          string `json:"dob"`
	ProfileImage string `json:"profile_image"`
	LoginID      string `json:"login_id"`
}

func sendErrorResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  false,
		"message": message,
	})
}

func SingUp_Customer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		//http.Error(w, "Invalid request method", http.StatusBadRequest)
		sendErrorResponse(w, "Invalid request method")
		return
	}

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		//http.Error(w, "Invalid JSON", http.StatusBadRequest)
		sendErrorResponse(w, "Invalid JSON format")
		return
	}

	// ---------- VALIDATIONS ----------
	validationErr := validateUser(user)
	if validationErr != "" {
		//http.Error(w, validationErr, http.StatusBadRequest)
		sendErrorResponse(w, validationErr)
		return
	}

	// check duplicate email
	var exists string
	err = db.DB.QueryRow("SELECT email FROM login WHERE email = ?", user.Email).Scan(&exists)
	if err == nil {
		sendErrorResponse(w, "Email already exists")
		return
	}
	// -------- PASSWORD HASHING ----------
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		//http.Error(w, "Password encryption failed", http.StatusInternalServerError)
		sendErrorResponse(w, "Password encryption failed")
		return
	}
	// ------ INSERT INTO LOGIN TABLE ------
	loginQuery := `INSERT INTO login (name, email, phone, type, status, email_verified, verification_code, password)
               VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.DB.Exec(loginQuery,
		user.FullName, user.Email, user.PhoneNumber, "customer", "inactive", 0, "", string(hashedPassword))

	if err != nil {
		sendErrorResponse(w, "Error inserting login: "+err.Error())
		return
	}

	// Get login id
	loginID, err := result.LastInsertId()
	if err != nil {
		sendErrorResponse(w, "Error getting login ID")
		return
	}

	// Insert customer record
	customerQuery := `INSERT INTO customer (login_id, full_name,password,email,country_code, phone_number, dob, profile_image)
                  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.DB.Exec(customerQuery,
		loginID, user.FullName, string(hashedPassword), user.Email, user.CountryCode, user.PhoneNumber, user.Dob, user.ProfileImage)

	if err != nil {
		sendErrorResponse(w, "Error inserting customer: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"status":  true,
		"message": "User created successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Validate function
func validateUser(u User) string {

	if strings.TrimSpace(u.FullName) == "" {
		return "Full name is required"
	}
	if len(u.Password) < 6 {
		return "Password must be at least 6 characters"
	}
	if !isValidEmail(u.Email) {
		return "Invalid email format"
	}
	if strings.TrimSpace(u.CountryCode) == "" {
		return "Country code is required"
	}
	if len(u.PhoneNumber) < 10 {
		return "Phone number must be at least 10 digits"
	}
	return ""
}

// Email format check
func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

var JwtKey = []byte(os.Getenv("e4d1c038b9b3b00b1681d92c1310afb8880d0a61e99f004e96d750f37f3ab085"))

func GenerateJWT(email string, userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		//"exp":     time.Now().Add(24 * time.Hour).Unix(),
		//"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtKey)
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func LoginCustomer(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Invalid request method")
		return
	}

	var loginReq LoginRequest
	err := json.NewDecoder(r.Body).Decode(&loginReq)
	if err != nil {
		sendErrorResponse(w, "Invalid JSON format")
		return
	}

	// Fetch login user
	var storedPassword string
	var userID int
	var email string
	var email_verified int

	query := "SELECT id, email, password, email_verified FROM login WHERE email = ?"
	err = db.DB.QueryRow(query, loginReq.Email).Scan(&userID, &email, &storedPassword, &email_verified)
	if err != nil {
		sendErrorResponse(w, "Invalid email or password")
		return
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(loginReq.Password))
	if err != nil {
		sendErrorResponse(w, "Invalid email or password")
		return
	}
	// Generate OTP
	otp := generateOTP()

	// Update login table with OTP
	_, err = db.DB.Exec("UPDATE login SET verification_code = ? WHERE id = ?", otp, userID)
	if err != nil {
		sendErrorResponse(w, "Failed to update OTP: "+err.Error())
		return
	}

	// Generate JWT token
	token, err := GenerateJWT(email, userID)
	if err != nil {
		sendErrorResponse(w, "Failed to generate token")
		return
	}

	// ------- FETCH CUSTOMER DETAILS -------
	var fullName, phone, countryCode, dob, profileImage, login_id string
	var user_id int
	customerQuery := `SELECT id, full_name, phone_number, country_code, dob, profile_image,login_id  FROM customer WHERE login_id = ?`
	err = db.DB.QueryRow(customerQuery, userID).Scan(&user_id, &fullName, &phone, &countryCode, &dob, &profileImage, &login_id)
	if err != nil {
		sendErrorResponse(w, "Customer details not found")
		return
	}

	// Response payload
	data := map[string]interface{}{
		"user_id":       user_id,
		"full_name":     fullName,
		"phone_number":  phone,
		"country_code":  countryCode,
		"dob":           dob,
		"profile_image": profileImage,
	}
	response := LoginResponse{
		Status:        true,
		Message:       "Login successful",
		Token:         token,
		Otp:           otp,
		EmailVerified: email_verified,
		Data:          data,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000)) // 6 digit OTP
}

type LoginResponse struct {
	Status        bool        `json:"status"`
	Message       string      `json:"message"`
	Token         string      `json:"token"`
	Otp           string      `json:"otp"`
	EmailVerified int         `json:"email_verified"`
	Data          interface{} `json:"data"`
}

type VerifyOTPRequest struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func CustomerVerifyOTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Invalid request method")
		return
	}

	var req VerifyOTPRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		sendErrorResponse(w, "Invalid JSON payload")
		return
	}

	// Fetch login user
	var storedOTP string
	var userID int
	var email string

	query := "SELECT id, email, verification_code FROM login WHERE email = ?"
	err = db.DB.QueryRow(query, req.Email).Scan(&userID, &email, &storedOTP)
	if err != nil {
		sendErrorResponse(w, "Invalid email or OTP")
		return
	}

	// Compare OTP
	if req.OTP != storedOTP {
		sendErrorResponse(w, "Incorrect OTP")
		return
	}

	// Update verification flag
	_, err = db.DB.Exec("UPDATE login SET email_verified = 1 WHERE id = ?", userID)
	if err != nil {
		sendErrorResponse(w, "Error updating verification status")
		return
	}

	// Generate JWT token
	token, err := GenerateJWT(email, userID)
	if err != nil {
		sendErrorResponse(w, "Failed to generate token")
		return
	}

	// Fetch customer profile
	var fullName, phone, countryCode, dob, profileImage string
	var user_id int
	customerQuery := `SELECT id,full_name, phone_number, country_code, dob, profile_image 
                      FROM customer WHERE login_id = ?`
	err = db.DB.QueryRow(customerQuery, userID).Scan(&user_id, &fullName, &phone, &countryCode, &dob, &profileImage)
	if err != nil {
		sendErrorResponse(w, "Customer details not found")
		return
	}

	// Data object
	data := map[string]interface{}{
		"user_id":       user_id,
		"full_name":     fullName,
		"phone_number":  phone,
		"country_code":  countryCode,
		"dob":           dob,
		"profile_image": profileImage,
	}

	// // Final response
	// response := map[string]interface{}{
	//     "status":  true,
	//     "message": "OTP verified successfully",
	//     "token":   token,
	//     "email":   email,
	//     "data":    data,
	// }
	response := LoginResponse{
		Status:  true,
		Message: "OTP verified successfully",
		Token:   token,
		Data:    data,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type ResendOTPRequest struct {
	Email string `json:"email"`
}

func CustomerResendOTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Invalid request method")
		return
	}

	var req ResendOTPRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		sendErrorResponse(w, "Invalid JSON format")
		return
	}

	// Check if user exists
	var userID int
	var email string

	query := "SELECT id, email FROM login WHERE email = ?"
	err = db.DB.QueryRow(query, req.Email).Scan(&userID, &email)
	if err != nil {
		sendErrorResponse(w, "Email not found")
		return
	}

	// Generate new OTP
	otp := generateOTP()

	// Update login table with new OTP
	_, err = db.DB.Exec("UPDATE login SET verification_code = ? WHERE id = ?", otp, userID)
	if err != nil {
		sendErrorResponse(w, "Failed to update OTP: "+err.Error())
		return
	}

	// Response format
	response := map[string]interface{}{
		"status":  true,
		"message": "OTP resent successfully",
		// "email":   email,
		// "otp":     otp, // remove later if sending via email/sms
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
