package routes

import (
	"GoEatsapi/db"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type CustomerRequest struct {
	CustomerID int `json:"customer_id"`
}

type CustomerDetails struct {
	ID           int    `json:"id"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	PhoneNumber  string `json:"phone_number"`
	CountryCode  string `json:"country_code"`
	ProfileImage string `json:"profile_image"`
	DOB          string `json:"dob"`
	LoginId      int    `json:"login_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// Success Response Struct
type CustomerAPIResponse struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    CustomerDetails `json:"data"`
}

func GetCustomerDetails(w http.ResponseWriter, r *http.Request) {

	// Validate Method
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Invalid request method")
		return
	}

	// Parse JSON Body
	var req CustomerRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.CustomerID <= 0 {
		sendErrorResponse(w, "customer_id required")
		return
	}

	// SQL Query
	query := `
		SELECT id, full_name, email, phone_number, country_code,
		       profile_image, dob, login_id, created_at, updated_at
		FROM customer
		WHERE id = ?
	`

	row := db.DB.QueryRow(query, req.CustomerID)
	var customer CustomerDetails

	err = row.Scan(
		&customer.ID,
		&customer.FullName,
		&customer.Email,
		&customer.PhoneNumber,
		&customer.CountryCode,
		&customer.ProfileImage,
		&customer.DOB,
		&customer.LoginId,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		response := map[string]interface{}{
			"message": "Customer not found",
			"status":  false,
			"data":    map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	} else if err != nil {
		sendErrorResponse(w, "Database error: "+err.Error())
		return
	}

	// // Success response
	// response := map[string]interface{}{
	// 	"status":  true,
	// 	"message": "Customer details fetched successfully",
	// 	"data":    customer,
	// }
	// On success
	response := CustomerAPIResponse{
		Status:  true,
		Message: "Customer details fetched successfully",
		Data:    customer,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type AddAddressRequest struct {
	CustomerID  int     `json:"customer_id"`
	FullName    string  `json:"full_name"`
	PhoneNumber string  `json:"phone_number"`
	AddressLine string  `json:"address"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	Country     string  `json:"country"`
	PostalCode  string  `json:"postal_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	IsDefault   int     `json:"is_default"`
}

func AddCustomerAddress(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req CustomerAddress
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CustomerID == 0 || req.AddressLine1 == "" || req.FullName == "" || req.PhoneNumber == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// ✅ CHECK IF SAME ADDRESS EXISTS
	var existingID int
	checkQuery := `
		SELECT id 
		FROM customer_delivery_addresses 
		WHERE customer_id = ?
		  AND address = ?
		  AND city = ?
		  AND postal_code = ?
		LIMIT 1
	`

	err := db.DB.QueryRow(
		checkQuery,
		req.CustomerID,
		req.AddressLine1,
		req.City,
		req.PostalCode,
	).Scan(&existingID)

	if err == nil {
		// Address already exists
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(AddressResponse{
			Status:  false,
			Message: "Address already exists",
			Data:    []CustomerAddress{},
		})
		return
	}

	if err != sql.ErrNoRows {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// ✅ HANDLE DEFAULT ADDRESS
	if req.IsDefault {
		_, err := db.DB.Exec(
			`UPDATE customer_delivery_addresses SET is_default = FALSE WHERE customer_id = ?`,
			req.CustomerID,
		)
		if err != nil {
			http.Error(w, "Failed to update default flag: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	query := `
		INSERT INTO customer_delivery_addresses 
		(customer_id, full_name, phone_number, address, city, state, country, postal_code, latitude, longitude, is_default)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.DB.Exec(
		query,
		req.CustomerID,
		req.FullName,
		req.PhoneNumber,
		req.AddressLine1,
		req.City,
		req.State,
		req.Country,
		req.PostalCode,
		req.Latitude,
		req.Longitude,
		req.IsDefault,
	)

	if err != nil {
		http.Error(w, "DB insert error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	insertedID, _ := result.LastInsertId()

	response := AddressResponse{
		Status:  true,
		Message: "Address added successfully",
		Data: []CustomerAddress{
			{
				ID:           int(insertedID),
				CustomerID:   req.CustomerID,
				FullName:     req.FullName,
				PhoneNumber:  req.PhoneNumber,
				AddressLine1: req.AddressLine1,
				City:         req.City,
				State:        req.State,
				Country:      req.Country,
				PostalCode:   req.PostalCode,
				Latitude:     req.Latitude,
				Longitude:    req.Longitude,
				IsDefault:    req.IsDefault,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type CustomerAddress struct {
	ID           int     `json:"id"`
	CustomerID   int     `json:"customer_id"`
	FullName     string  `json:"full_name"`
	PhoneNumber  string  `json:"phone_number"`
	AddressLine1 string  `json:"address"`
	City         string  `json:"city"`
	State        string  `json:"state"`
	Country      string  `json:"country"`
	PostalCode   string  `json:"postal_code"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	IsDefault    bool    `json:"is_default"`
	CreatedAt    string  `json:"created_at"`
}

func GetCustomerAddresses(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req CustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CustomerID == 0 {
		http.Error(w, "customer_id is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT 
			id,
			customer_id,
			full_name,
			phone_number,
			address,
			city,
			state,
			country,
			postal_code,
			latitude,
			longitude,
			is_default,
			created_at
		FROM customer_delivery_addresses
		WHERE customer_id = ?
		ORDER BY is_default DESC, created_at DESC;
	`

	rows, err := db.DB.Query(query, req.CustomerID)
	if err != nil {
		http.Error(w, "DB query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	addresses := []CustomerAddress{}

	for rows.Next() {
		var addr CustomerAddress
		if err := rows.Scan(
			&addr.ID,
			&addr.CustomerID,
			&addr.FullName,
			&addr.PhoneNumber,
			&addr.AddressLine1,
			&addr.City,
			&addr.State,
			&addr.Country,
			&addr.PostalCode,
			&addr.Latitude,
			&addr.Longitude,
			&addr.IsDefault,
			&addr.CreatedAt,
		); err != nil {
			http.Error(w, "DB scan error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		addresses = append(addresses, addr)
	}

	// If no records found
	if len(addresses) == 0 {
		response := AddressResponse{
			Status:  false,
			Message: "No address found",
			Data:    []CustomerAddress{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Success
	response := AddressResponse{
		Status:  true,
		Message: "Address list fetched successfully",
		Data:    addresses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type AddressResponse struct {
	Status  bool              `json:"status"`
	Message string            `json:"message"`
	Data    []CustomerAddress `json:"data"`
}

type DeleteAddressRequest struct {
	CustomerID int `json:"customer_id"`
	AddressID  int `json:"address_id"`
}

func DeleteCustomerAddress(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	type DeleteAddressRequest struct {
		CustomerID int `json:"customer_id"`
		AddressID  int `json:"address_id"`
	}

	var req DeleteAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CustomerID == 0 || req.AddressID == 0 {
		http.Error(w, "customer_id and address_id are required", http.StatusBadRequest)
		return
	}

	// Check if record exists & is default or not
	checkQuery := `SELECT is_default FROM customer_delivery_addresses WHERE id = ? AND customer_id = ? LIMIT 1`

	var isDefault bool
	err := db.DB.QueryRow(checkQuery, req.AddressID, req.CustomerID).Scan(&isDefault)
	if err == sql.ErrNoRows {
		response := map[string]interface{}{
			"status":  false,
			"message": "Address not found",
			"data":    []CustomerAddress{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	} else if err != nil {
		http.Error(w, "DB query error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prevent deleting default address
	if isDefault {
		response := map[string]interface{}{
			"status":  false,
			"message": "Default address cannot be deleted",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Delete address
	deleteQuery := `DELETE FROM customer_delivery_addresses WHERE id = ? AND customer_id = ?`
	_, delErr := db.DB.Exec(deleteQuery, req.AddressID, req.CustomerID)
	if delErr != nil {
		http.Error(w, "DB delete error: "+delErr.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch remaining address list
	listQuery := `
		SELECT id, customer_id, full_name, phone_number, address, city, state, country,
		       postal_code, latitude, longitude, is_default, created_at
		FROM customer_delivery_addresses
		WHERE customer_id = ?
		ORDER BY is_default DESC, created_at DESC;
	`

	rows, err := db.DB.Query(listQuery, req.CustomerID)
	if err != nil {
		http.Error(w, "DB query error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	addresses := []CustomerAddress{}

	for rows.Next() {
		var addr CustomerAddress
		if err := rows.Scan(&addr.ID, &addr.CustomerID, &addr.FullName, &addr.PhoneNumber, &addr.AddressLine1,
			&addr.City, &addr.State, &addr.Country, &addr.PostalCode, &addr.Latitude, &addr.Longitude, &addr.IsDefault, &addr.CreatedAt); err != nil {

			http.Error(w, "DB scan error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		addresses = append(addresses, addr)
	}

	// Success response with updated list
	response := map[string]interface{}{
		"status":  true,
		"message": "Address deleted successfully",
		"data":    addresses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type UpdateProfileRequest struct {
	UserID       int    `json:"user_id"`
	LoginID      int    `json:"login_id"`
	FullName     string `json:"full_name"`
	DOB          string `json:"dob"`
	Email        string `json:"email"`
	PhoneNumber  string `json:"phone_number"`
	ProfileImage string `json:"profile_image"`
}

func UpdateCustomerProfile(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPut {
		sendErrorResponse(w, "Invalid request method")
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid body: "+err.Error())
		return
	}

	// Validate required
	if req.UserID == 0 || req.FullName == "" || req.Email == "" {
		sendErrorResponse(w, "Required fields are missing")
		return
	}

	// Get login_id from customer table
	var loginID int
	err := db.DB.QueryRow("SELECT login_id FROM customer WHERE id=?", req.UserID).Scan(&loginID)
	if err != nil || loginID == 0 {
		sendErrorResponse(w, "Login ID not found for user")
		return
	}

	// Email unique check in login table except this user
	var emailCount int
	err = db.DB.QueryRow(
		"SELECT COUNT(*) FROM login WHERE email=? AND id!=?",
		req.Email, loginID,
	).Scan(&emailCount)

	if err != nil {
		sendErrorResponse(w, "DB error: "+err.Error())
		return
	}

	// if emailCount > 0 {
	// 	sendErrorResponse(w, "Email already exists")
	// 	return
	// }

	// Validate DOB
	var dobVal sql.NullString
	if req.DOB != "" {
		if _, err := time.Parse("2006-01-02", req.DOB); err != nil {
			sendErrorResponse(w, "Invalid DOB format")
			return
		}
		dobVal = sql.NullString{String: req.DOB, Valid: true}
	}

	// Transaction
	tx, err := db.DB.Begin()
	if err != nil {
		sendErrorResponse(w, "Transaction error")
		return
	}

	// Update customer
	_, err = tx.Exec(
		`UPDATE customer 
		 SET full_name=?, dob=?, profile_image=?, email=?, phone_number=?
		 WHERE id=?`,
		req.FullName, dobVal, req.ProfileImage, req.Email, req.PhoneNumber, req.UserID,
	)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(w, "DB Error: "+err.Error())
		return
	}

	// Update login
	_, err = tx.Exec(
		`UPDATE login
		 SET name=?, email=?, phone=?
		 WHERE id=?`,
		req.FullName, req.Email, req.PhoneNumber, loginID,
	)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(w, "DB Error: "+err.Error())
		return
	}

	tx.Commit()

	response := map[string]interface{}{
		"message": "Profile updated successfully",
	}

	successResponse(w, response)
}

func successResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status": true,
		"data":   data,
	}
	json.NewEncoder(w).Encode(response)
}

type ContactUsRequest struct {
	UserType string `json:"user_type"`
	UserID   uint64 `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Message  string `json:"message"`
}

type APIContactusResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

func CreateContactUs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req ContactUsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIContactusResponse{
			Status:  false,
			Message: "Invalid request payload",
		})
		return
	}

	// Basic validation
	if req.Name == "" || req.Email == "" || req.Message == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIContactusResponse{
			Status:  false,
			Message: "Name, email, and message are required",
		})
		return
	}

	// Default user_type
	if req.UserType == "" {
		req.UserType = "guest"
	}

	query := `
		INSERT INTO tbl_contact_us
		(user_type, user_id, name, email, phone, message)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.DB.Exec(
		query,
		req.UserType,
		req.UserID,
		req.Name,
		req.Email,
		req.Phone,
		req.Message,
	)

	if err != nil {
		//log.Println("Contact Us Insert Error:", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIContactusResponse{
			Status:  false,
			Message: "Failed to submit contact request",
		})
		return
	}

	json.NewEncoder(w).Encode(APIContactusResponse{
		Status:  true,
		Message: "Your message has been submitted successfully",
	})
}
