package routes

import (
	"GoEatsapi/db"
	"GoEatsapi/mailer"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

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
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		sendErrorResponse(w, "Invalid JSON format")
		return
	}

	// Fetch login user
	var storedPassword string
	var userID int
	var email string
	var email_verified int

	query := `
		SELECT id, email, password, email_verified
		FROM login WHERE email = ?`

	err := db.DB.QueryRow(query, loginReq.Email).
		Scan(&userID, &email, &storedPassword, &email_verified)
	if err != nil {
		sendErrorResponse(w, "Invalid email or password")
		return
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword(
		[]byte(storedPassword),
		[]byte(loginReq.Password),
	); err != nil {
		sendErrorResponse(w, "Invalid email or password")
		return
	}

	_, err = db.DB.Exec(
		"UPDATE login SET status = 'active' WHERE id = ?",
		userID,
	)
	if err != nil {
		sendErrorResponse(w, "Failed to update login status")
		return
	}

	// 🔐 Send OTP only if email not verified
	if email_verified == 0 {

		otp := generateOTP()

		_, err = db.DB.Exec(
			"UPDATE login SET verification_code = ? WHERE id = ?",
			otp, userID,
		)
		if err != nil {
			sendErrorResponse(w, "Failed to update OTP")
			return
		}

		subject := "Your Login OTP"
		htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Your GoEats OTP</title>
</head>
<body style="margin:0; padding:0; font-family: Arial, sans-serif; background-color:#f4f4f4;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="padding:20px;">
		<tr>
			<td align="center">

				<table width="600" cellpadding="0" cellspacing="0"
					style="background:#ffffff; border-radius:8px; overflow:hidden;">

					<!-- HEADER -->
					<tr>
						<td style="background-color:#ff6b35; padding:20px; text-align:center;">
							<img src="https://yourdomain.com/assets/logo.png"
								alt="GoEats"
								style="max-height:50px;">
						</td>
					</tr>

					<!-- TITLE -->
					<tr>
						<td style="padding:20px; text-align:center;">
							<h2 style="color:#2c3e50; margin:0;">🔐 Login Verification</h2>
						</td>
					</tr>

					<!-- CONTENT -->
					<tr>
						<td style="padding:0 20px 10px; color:#555;">
							Dear User,
						</td>
					</tr>

					<tr>
						<td style="padding:0 20px 15px; color:#555;">
							Your OTP for login is:
						</td>
					</tr>

					<!-- OTP BOX -->
					<tr>
						<td align="center" style="padding:10px 20px;">
							<div style="
								display:inline-block;
								background:#fef3ee;
								border:1px dashed #ff6b35;
								color:#ff6b35;
								font-size:28px;
								letter-spacing:6px;
								font-weight:bold;
								padding:12px 20px;
								border-radius:6px;">
								%s
							</div>
						</td>
					</tr>

					<!-- VALIDITY -->
					<tr>
						<td style="padding:15px 20px; color:#555; text-align:center;">
							This OTP is valid for <strong>5 minutes</strong>.
						</td>
					</tr>

					<!-- SECURITY NOTE -->
					<tr>
						<td style="padding:0 20px 20px; font-size:13px; color:#777;">
							For your security, please do not share this OTP with anyone.
						</td>
					</tr>

					<!-- FOOTER -->
					<tr>
						<td style="background:#f9f9f9; padding:15px; text-align:center; font-size:12px; color:#999;">
							Regards,<br>
							<strong style="color:#ff6b35;">GoEats Team</strong>
						</td>
					</tr>

				</table>

			</td>
		</tr>
	</table>
</body>
</html>
`,
			otp,
		)

		if err = mailer.SendHTMLEmail(email, subject, htmlBody); err != nil {
			fmt.Println("SMTP ERROR:", err)
			sendErrorResponse(w, "Failed to send OTP email")
			return
		}
	}

	// Generate JWT
	token, err := GenerateJWT(email, userID)
	if err != nil {
		sendErrorResponse(w, "Failed to generate token")
		return
	}

	// Fetch customer details
	var fullName, phone, countryCode, dob, profileImage, login_id string
	var user_id int

	customerQuery := `
		SELECT id, full_name, phone_number, country_code, dob, profile_image, login_id
		FROM customer WHERE login_id = ?`

	err = db.DB.QueryRow(customerQuery, userID).
		Scan(&user_id, &fullName, &phone, &countryCode, &dob, &profileImage, &login_id)
	if err != nil {
		sendErrorResponse(w, "Customer details not found")
		return
	}

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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid JSON format")
		return
	}

	// Fetch user
	var userID int
	var email string
	var email_verified int

	query := `
		SELECT id, email, email_verified
		FROM login WHERE email = ?`

	err := db.DB.QueryRow(query, req.Email).
		Scan(&userID, &email, &email_verified)
	if err != nil {
		sendErrorResponse(w, "Email not found")
		return
	}

	// If already verified, do not resend OTP
	if email_verified == 1 {
		sendErrorResponse(w, "Email already verified")
		return
	}

	// Generate new OTP
	otp := generateOTP()

	// Update OTP in DB
	_, err = db.DB.Exec(
		"UPDATE login SET verification_code = ? WHERE id = ?",
		otp, userID,
	)
	if err != nil {
		sendErrorResponse(w, "Failed to update OTP")
		return
	}

	// Send OTP email
	subject := "Your Verification OTP"
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Your GoEats OTP</title>
</head>
<body style="margin:0; padding:0; font-family: Arial, sans-serif; background-color:#f4f4f4;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="padding:20px;">
		<tr>
			<td align="center">

				<table width="600" cellpadding="0" cellspacing="0"
					style="background:#ffffff; border-radius:8px; overflow:hidden;">

					<!-- HEADER -->
					<tr>
						<td style="background-color:#ff6b35; padding:20px; text-align:center;">
							<img src="https://yourdomain.com/assets/logo.png"
								alt="GoEats"
								style="max-height:50px;">
						</td>
					</tr>

					<!-- TITLE -->
					<tr>
						<td style="padding:20px; text-align:center;">
							<h2 style="color:#2c3e50; margin:0;">🔐 Login Verification</h2>
						</td>
					</tr>

					<!-- CONTENT -->
					<tr>
						<td style="padding:0 20px 10px; color:#555;">
							Dear User,
						</td>
					</tr>

					<tr>
						<td style="padding:0 20px 15px; color:#555;">
							Your OTP for login is:
						</td>
					</tr>

					<!-- OTP BOX -->
					<tr>
						<td align="center" style="padding:10px 20px;">
							<div style="
								display:inline-block;
								background:#fef3ee;
								border:1px dashed #ff6b35;
								color:#ff6b35;
								font-size:28px;
								letter-spacing:6px;
								font-weight:bold;
								padding:12px 20px;
								border-radius:6px;">
								%s
							</div>
						</td>
					</tr>

					<!-- VALIDITY -->
					<tr>
						<td style="padding:15px 20px; color:#555; text-align:center;">
							This OTP is valid for <strong>10 minutes</strong>.
						</td>
					</tr>

					<!-- SECURITY NOTE -->
					<tr>
						<td style="padding:0 20px 20px; font-size:13px; color:#777;">
							For your security, please do not share this OTP with anyone.
						</td>
					</tr>

					<!-- FOOTER -->
					<tr>
						<td style="background:#f9f9f9; padding:15px; text-align:center; font-size:12px; color:#999;">
							Regards,<br>
							<strong style="color:#ff6b35;">GoEats Team</strong>
						</td>
					</tr>

				</table>

			</td>
		</tr>
	</table>
</body>
</html>
`,
		otp,
	)

	err = mailer.SendHTMLEmail(email, subject, htmlBody)
	if err != nil {
		fmt.Println("SMTP ERROR:", err)
		sendErrorResponse(w, "Failed to send OTP email")
		return
	}

	// Success response
	response := map[string]interface{}{
		"status":  true,
		"message": "OTP sent successfully to your email",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateRandomPassword(length int) (string, error) {
	const (
		upper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lower   = "abcdefghijklmnopqrstuvwxyz"
		number  = "0123456789"
		special = "!@#$%^&*"
		all     = upper + lower + number + special
	)

	if length < 8 {
		return "", fmt.Errorf("password length must be at least 8")
	}

	password := make([]byte, length)
	sets := []string{upper, lower, number, special}

	for i, set := range sets {
		n, err := crand.Int(crand.Reader, big.NewInt(int64(len(set))))
		if err != nil {
			return "", err
		}
		password[i] = set[n.Int64()]
	}

	for i := len(sets); i < length; i++ {
		n, err := crand.Int(crand.Reader, big.NewInt(int64(len(all))))
		if err != nil {
			return "", err
		}
		password[i] = all[n.Int64()]
	}

	for i := range password {
		j, err := crand.Int(crand.Reader, big.NewInt(int64(len(password))))
		if err != nil {
			return "", err
		}
		password[i], password[j.Int64()] = password[j.Int64()], password[i]
	}

	return string(password), nil
}

func ForgotPassword(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Invalid request method")
		return
	}

	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" {
		sendErrorResponse(w, "Invalid email")
		return
	}

	// Check if email exists
	var loginID int
	var email string

	err := db.DB.QueryRow(
		"SELECT id, email FROM login WHERE email = ?",
		req.Email,
	).Scan(&loginID, &email)

	if err != nil {
		sendErrorResponse(w, "Email not registered")
		return
	}

	// Generate random password
	newPassword, err := generateRandomPassword(8)
	if err != nil {
		sendErrorResponse(w, "Failed to generate password")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(newPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		sendErrorResponse(w, "Failed to encrypt password")
		return
	}

	// Update login table
	_, err = db.DB.Exec(
		"UPDATE login SET password = ? WHERE id = ?",
		string(hashedPassword), loginID,
	)
	if err != nil {
		sendErrorResponse(w, "Failed to update login password")
		return
	}

	// Update customer table
	_, err = db.DB.Exec(
		"UPDATE customer SET password = ? WHERE login_id = ?",
		string(hashedPassword), loginID,
	)
	if err != nil {
		sendErrorResponse(w, "Failed to update customer password")
		return
	}

	// Send email with new password
	subject := "Your New Password"
	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Your New GoEats Password</title>
</head>
<body style="margin:0; padding:0; font-family: Arial, sans-serif; background-color:#f4f4f4;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="padding:20px;">
		<tr>
			<td align="center">

				<table width="600" cellpadding="0" cellspacing="0"
					style="background:#ffffff; border-radius:8px; overflow:hidden;">

					<!-- HEADER -->
					<tr>
						<td style="background-color:#ff6b35; padding:20px; text-align:center;">
							<img src="https://yourdomain.com/assets/logo.png"
								alt="GoEats"
								style="max-height:50px;">
						</td>
					</tr>

					<!-- TITLE -->
					<tr>
						<td style="padding:20px; text-align:center;">
							<h2 style="color:#2c3e50; margin:0;">🔑 New Password Generated</h2>
						</td>
					</tr>

					<!-- CONTENT -->
					<tr>
						<td style="padding:0 20px 10px; color:#555;">
							Dear User,
						</td>
					</tr>

					<tr>
						<td style="padding:0 20px 15px; color:#555;">
							Your new password is:
						</td>
					</tr>

					<!-- PASSWORD BOX -->
					<tr>
						<td align="center" style="padding:10px 20px;">
							<div style="
								display:inline-block;
								background:#fef3ee;
								border:1px dashed #ff6b35;
								color:#ff6b35;
								font-size:18px;
								letter-spacing:1px;
								font-weight:bold;
								padding:12px 20px;
								border-radius:6px;">
								%s
							</div>
						</td>
					</tr>

					<!-- WARNING -->
					<tr>
						<td style="padding:15px 20px; color:#555;">
							Please login and <strong>change your password immediately</strong>.
						</td>
					</tr>

					<!-- SECURITY NOTE -->
					<tr>
						<td style="padding:0 20px 20px; font-size:13px; color:#777;">
							For your security, do not share this password with anyone.
						</td>
					</tr>

					<!-- FOOTER -->
					<tr>
						<td style="background:#f9f9f9; padding:15px; text-align:center; font-size:12px; color:#999;">
							Regards,<br>
							<strong style="color:#ff6b35;">GoEats Team</strong>
						</td>
					</tr>

				</table>

			</td>
		</tr>
	</table>
</body>
</html>
`,
		newPassword,
	)

	if err = mailer.SendHTMLEmail(email, subject, htmlBody); err != nil {
		sendErrorResponse(w, "Failed to send email")
		return
	}

	// Success response
	response := map[string]interface{}{
		"status":  true,
		"message": "New password sent to your registered email",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
