package DeliveryPartner

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"GoEatsapi/config"
	"GoEatsapi/db"
	"GoEatsapi/models"
	"GoEatsapi/utils"

	"golang.org/x/crypto/bcrypt"
)

// JSON response helper
func jsonResponse(w http.ResponseWriter, status int, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": success,
		"message": message,
		"data":    data,
	})
}

// SignUp handles user registration
func SignUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, false, "Only POST method is allowed", nil)
		return
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		jsonResponse(w, http.StatusBadRequest, false, "Error parsing form: "+err.Error(), nil)
		return
	}

	password := strings.TrimSpace(r.FormValue("password"))
	confirmPassword := strings.TrimSpace(r.FormValue("confirm_password"))

	user := models.User{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Email:       strings.TrimSpace(r.FormValue("email")),
		Phone:       strings.TrimSpace(r.FormValue("phone")),
		Address:     strings.TrimSpace(r.FormValue("address")),
		City:        strings.TrimSpace(r.FormValue("city")),
		State:       strings.TrimSpace(r.FormValue("state")),
		Zipcode:     strings.TrimSpace(r.FormValue("zipcode")),
		Country:     strings.TrimSpace(r.FormValue("country")),
		Gender:      strings.ToLower(strings.TrimSpace(r.FormValue("gender"))),
		DateOfBirth: strings.TrimSpace(r.FormValue("date_of_birth")),
		UserType:    strings.ToLower(strings.TrimSpace(r.FormValue("user_type"))),
		IDNumber:    strings.TrimSpace(r.FormValue("id_number")),
	}

	// ✅ Validate required fields
	if user.Name == "" || user.Email == "" || password == "" || confirmPassword == "" {
		jsonResponse(w, http.StatusBadRequest, false, "Name, Email, Password, and Confirm Password are required", nil)
		return
	}

	if password != confirmPassword {
		jsonResponse(w, http.StatusBadRequest, false, "Password and Confirm Password do not match", nil)
		return
	}

	// ✅ Validate email format
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(user.Email) {
		jsonResponse(w, http.StatusBadRequest, false, "Invalid email format", nil)
		return
	}

	// ✅ Set defaults
	if user.Gender == "" {
		user.Gender = "other"
	}
	if user.UserType == "" {
		user.UserType = "customer"
	}
	if user.Country == "" {
		user.Country = "United States"
	}

	// ✅ Validate date_of_birth
	if user.DateOfBirth != "" {
		if _, err := time.Parse("2006-01-02", user.DateOfBirth); err != nil {
			jsonResponse(w, http.StatusBadRequest, false, "Invalid date_of_birth format (expected YYYY-MM-DD)", nil)
			return
		}
	}

	// ✅ Save uploaded files
	user.IDDocFront, _ = saveFile(r, "id_doc_front")
	user.IDDocBack, _ = saveFile(r, "id_doc_back")
	user.ProfilePic, _ = saveFile(r, "profile_pic")

	// ✅ Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Error hashing password", nil)
		return
	}

	// ✅ Check if email already exists
	var existingEmail string
	checkQuery := `SELECT email FROM login WHERE email = ?`
	err = db.DB.QueryRow(checkQuery, user.Email).Scan(&existingEmail)
	if err == nil {
		jsonResponse(w, http.StatusBadRequest, false, "Email already registered", nil)
		return
	}

	// ✅ Start transaction (both tables)
	tx, err := db.DB.Begin()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Transaction start failed", nil)
		return
	}

	// Insert into users table
	userQuery := `
        INSERT INTO users (name, email, phone, address, city, state, zipcode, country, gender,
        date_of_birth, user_type, profile_pic, id_number, id_doc_front, id_doc_back)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
	result, err := tx.Exec(userQuery,
		user.Name, user.Email, user.Phone, user.Address, user.City, user.State,
		user.Zipcode, user.Country, user.Gender, user.DateOfBirth, user.UserType,
		user.ProfilePic, user.IDNumber, user.IDDocFront, user.IDDocBack,
	)
	if err != nil {
		tx.Rollback()
		jsonResponse(w, http.StatusInternalServerError, false, fmt.Sprintf("User insert failed: %v", err), nil)
		return
	}

	insertedID, _ := result.LastInsertId()
	user.ID = int(insertedID)

	// Insert into login table
	loginQuery := `
        INSERT INTO login (name, email, password, phone, type)
        VALUES (?, ?, ?, ?, ?)
    `
	_, err = tx.Exec(loginQuery, user.Name, user.Email, string(hashedPassword), user.Phone, user.UserType)
	if err != nil {
		tx.Rollback()
		jsonResponse(w, http.StatusInternalServerError, false, fmt.Sprintf("Login insert failed: %v", err), nil)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		jsonResponse(w, http.StatusInternalServerError, false, "Transaction commit failed", nil)
		return
	}

	// ✅ Success Response
	jsonResponse(w, http.StatusOK, true, "User registered successfully ✅", map[string]interface{}{
		"user_id":      user.ID,
		"name":         user.Name,
		"email":        user.Email,
		"user_type":    user.UserType,
		"id_doc_front": user.IDDocFront,
		"id_doc_back":  user.IDDocBack,
	})
}

// saveFile saves the uploaded file and returns its relative path
func saveFile(r *http.Request, fieldName string) (string, error) {
	config.LoadEnv()

	// Read file from form
	file, handler, err := r.FormFile(fieldName)
	if err != nil {
		// No file uploaded for this field
		return "", nil
	}
	defer file.Close()

	// Ensure upload directory exists
	uploadDir := "uploads/ids"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %v", err)
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(handler.Filename))
	fullPath := filepath.Join(uploadDir, filename)

	// Save file to disk
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	// Get base URL from .env
	baseURL := config.GetEnv("APP_BASE_URL", "http://localhost:8080")

	// Return full accessible URL (normalize slashes)
	fileURL := fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), filepath.ToSlash(fullPath))

	return fileURL, nil
}

// func uploadImageToSupabase(r *http.Request, fieldName string) (string, error) {
// 	file, handler, err := r.FormFile(fieldName)
// 	if err != nil {
// 		return "", fmt.Errorf("FormFile error: %w", err)
// 	}
// 	defer file.Close()

// 	// Check MIME type
// 	buff := make([]byte, 512)
// 	_, err = file.Read(buff)
// 	if err != nil {
// 		return "", err
// 	}
// 	filetype := http.DetectContentType(buff)
// 	if !strings.HasPrefix(filetype, "image/") {
// 		return "", fmt.Errorf("file is not an image")
// 	}
// 	file.Seek(0, 0)

// 	// Unique filename
// 	fileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)

// 	// Upload
// 	resp, err := utils.StorageClient.UploadFile("GoEats_partner_docs", fileName, file)
// 	if err != nil {
// 		return "", fmt.Errorf("upload failed: %w", err)
// 	}
// 	fmt.Printf("Upload response: %+v\n", resp)

// 	// Get signed URL
// 	publicResp, err := utils.StorageClient.CreateSignedUrl("GoEats_partner_docs", fileName, 3600) // 1 hour
// 	if err != nil {
// 		return "", fmt.Errorf("signed URL failed: %w", err)
// 	}

// 	return publicResp.SignedURL, nil
// }

func UpdateDeliveryPartnerHandler(w http.ResponseWriter, r *http.Request) {

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

	// Parse token → loginID
	loginID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 2. Parse FormData
	// -------------------------------
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		JSON(w, 400, false, "Invalid form data", nil)
		return
	}

	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	dob := r.FormValue("date_of_birth")
	primaryMobile := r.FormValue("primary_mobile")
	gender := r.FormValue("gender")

	password := r.FormValue("password")
	drivingLicense := r.FormValue("driving_license_number")
	drivingLicenseexpiry := r.FormValue("driving_license_expire")
	// ---------------------------------------
	// 🔥 Upload Profile Photo (OPTIONAL)
	// ---------------------------------------
	profilePhotoURL := ""
	drivingLicenseURL := ""
	// Check if file was uploaded
	if _, _, err := r.FormFile("profile_photo"); err == nil {
		// User uploaded a file → upload to B2
		profilePhotoURL, _ = saveFile(r, "profile_photo")
	}

	if _, _, err := r.FormFile("driving_license"); err == nil {
		drivingLicenseURL, _ = saveFile(r, "driving_license")
	}

	// -------------------------------
	// 3. Build dynamic SQL for delivery_partners
	// -------------------------------
	dpFields := []string{}
	dpValues := []interface{}{}

	if firstName != "" {
		dpFields = append(dpFields, "first_name = ?")
		dpValues = append(dpValues, firstName)
	}
	if lastName != "" {
		dpFields = append(dpFields, "last_name = ?")
		dpValues = append(dpValues, lastName)
	}
	if dob != "" {
		dpFields = append(dpFields, "date_of_birth = ?")
		dpValues = append(dpValues, dob)
	}
	if primaryMobile != "" {
		dpFields = append(dpFields, "primary_mobile = ?")
		dpValues = append(dpValues, primaryMobile)
	}
	if gender != "" {
		dpFields = append(dpFields, "gender = ?")
		dpValues = append(dpValues, gender)
	}
	// 🔥 Add profile image
	if profilePhotoURL != "" {
		dpFields = append(dpFields, "profile_photo_url = ?")
		dpValues = append(dpValues, profilePhotoURL)
	}

	if drivingLicenseURL != "" {
		dpFields = append(dpFields, "driving_license_url = ?")
		dpValues = append(dpValues, drivingLicenseURL)
	}
	if drivingLicense != "" {
		dpFields = append(dpFields, "driving_license_number = ?")
		dpValues = append(dpValues, drivingLicense)
	}
	if drivingLicenseexpiry != "" {
		dpFields = append(dpFields, "driving_license_expire = ?")
		dpValues = append(dpValues, drivingLicenseexpiry)
	}

	if len(dpFields) > 0 {
		dpValues = append(dpValues, loginID)

		updateDPQuery := `
			UPDATE delivery_partners 
			SET ` + strings.Join(dpFields, ", ") + `, updated_at = NOW()
			WHERE login_id = ?
		`

		_, err = db.DB.Exec(updateDPQuery, dpValues...)
		if err != nil {
			fmt.Println("Failed to update delivery partner: ", err)
			JSON(w, 500, false, "Failed to update delivery partner", nil)
			return
		}
	}

	// -------------------------------
	// 4. Update LOGIN table (name + phone + password)
	// -------------------------------
	loginFields := []string{}
	loginValues := []interface{}{}

	// Name update
	if firstName != "" || lastName != "" {
		fullName := strings.TrimSpace(firstName + " " + lastName)
		loginFields = append(loginFields, "name = ?")
		loginValues = append(loginValues, fullName)
	}

	// Mobile update
	if primaryMobile != "" {
		loginFields = append(loginFields, "phone = ?")
		loginValues = append(loginValues, primaryMobile)
	}

	// ---------------------------------------
	// 🔥 Password update
	// ---------------------------------------
	if password != "" {
		// Encrypt password
		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			JSON(w, 500, false, "Failed to encrypt password", nil)
			return
		}

		loginFields = append(loginFields, "password = ?")
		loginValues = append(loginValues, string(hashed))
	}

	// Run login table update
	if len(loginFields) > 0 {
		loginValues = append(loginValues, loginID)

		updateLoginQuery := `
			UPDAte login 
			SET ` + strings.Join(loginFields, ", ") + `, updated_at = NOW()
			WHERE id = ?
		`

		_, err = db.DB.Exec(updateLoginQuery, loginValues...)
		if err != nil {
			JSON(w, 500, false, "Failed to update login table", nil)
			return
		}
	}

	// -------------------------------
	// 5. CHECK STEP 1 COMPLETION
	// -------------------------------
	var check struct {
		FName                string `db:"first_name"`
		LName                string `db:"last_name"`
		DOB                  string `db:"date_of_birth"`
		Mobile               string `db:"primary_mobile"`
		Gender               string `db:"gender"`
		ProfilePhotoURL      string `db:"profile_photo_url"` // 🔥 NEW
		DrivingLicenseURL    string `db:"driving_license_url"`
		DrivingLicense       string `db:"driving_license_number"`
		DrivingLicenseexpiry string `db:"driving_license_expire"`
	}

	step1Query := `
        SELECT first_name, last_name, date_of_birth, primary_mobile, gender,COALESCE(profile_photo_url,''),COALESCE(driving_license_url,''),COALESCE(driving_license_number,''),COALESCE(driving_license_expire,'')
        FROM delivery_partners
        WHERE login_id = ?
    `
	err = db.DB.QueryRow(step1Query, loginID).Scan(
		&check.FName,
		&check.LName,
		&check.DOB,
		&check.Mobile,
		&check.Gender,
		&check.ProfilePhotoURL, // 🔥 NEW
		&check.DrivingLicenseURL,
		&check.DrivingLicense,
		&check.DrivingLicenseexpiry,
	)

	if err != nil {
		JSON(w, 500, false, "Failed to verify step 1", nil)
		return
	}

	step1Completed := check.FName != "" &&
		check.LName != "" &&
		check.DOB != "" &&
		check.Mobile != "" &&
		check.Gender != ""

	// -------------------------------
	// 6. CHECK STEP 2 (password completed)
	// -------------------------------
	var hasPassword bool
	err = db.DB.QueryRow(`SELECT password IS NOT NULL AND password != '' FROM login WHERE id = ?`, loginID).Scan(&hasPassword)
	if err != nil {
		hasPassword = false
	}

	step2Completed := hasPassword
	step3Completed := check.ProfilePhotoURL != ""
	step4Completed := check.DrivingLicenseURL != "" && check.DrivingLicense != "" && check.DrivingLicenseexpiry != ""

	// -------------------------------
	// Final JSON Response
	// -------------------------------

	// -------------------------------
	// 7. CHECK IF ALL STEPS COMPLETED → UPDATE profile_completed
	// -------------------------------
	allCompleted := step1Completed && step2Completed && step3Completed && step4Completed

	if allCompleted {
		_, err = db.DB.Exec(`
		UPDATE delivery_partners 
		SET profile_completed = 1, updated_at = NOW()
		WHERE login_id = ?
	`, loginID)

		if err != nil {
			fmt.Println("Failed to update profile_completed:", err)
		}
	}

	JSON(w, 200, true, "Profile updated successfully", map[string]interface{}{
		"step_1_completed":  step1Completed,
		"step_2_completed":  step2Completed,
		"step_3_completed":  step3Completed,
		"step_4_completed":  step4Completed,
		"profile_completed": allCompleted,
	})
}

func Get_partner_details(w http.ResponseWriter, r *http.Request) {

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
		utils.JSON(w, 500, false, "Failed to fetch login info", nil)
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
		utils.JSON(w, 500, false, "Failed to fetch partner info", nil)
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
	utils.JSON(w, 200, true, "Success", map[string]interface{}{
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
