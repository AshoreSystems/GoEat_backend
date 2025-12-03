package routes

import (
	"GoEatsapi/db"
	"database/sql"
	"encoding/json"
	"net/http"
)

type CustomerRequest struct {
	CustomerID int64 `json:"customer_id"`
}

type CustomerDetails struct {
	ID           int64  `json:"id"`
	FullName     string `json:"full_name"`
	Email        string `json:"email"`
	PhoneNumber  string `json:"phone_number"`
	CountryCode  string `json:"country_code"`
	ProfileImage string `json:"profile_image"`
	DOB          string `json:"dob"`
	login_id     int    `json:"login_id"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
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
		&customer.login_id,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		response := map[string]interface{}{
			"status":  false,
			"message": "Customer not found",
			"data":    struct{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	} else if err != nil {
		sendErrorResponse(w, "Database error: "+err.Error())
		return
	}

	// Success response
	response := map[string]interface{}{
		"status":  true,
		"message": "Customer details fetched successfully",
		"data":    customer,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
