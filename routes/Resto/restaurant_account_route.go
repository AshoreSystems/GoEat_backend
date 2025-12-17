package resto

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/account"
	"github.com/stripe/stripe-go/v76/accountlink"
)

func CreateStripeRestaurantOnboarding(w http.ResponseWriter, r *http.Request) {

	// -------------------------------
	// 1. Read Authorization Token
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
	loginID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 2. FETCH stripe_account_id DIRECTLY
	// -------------------------------
	var stripeAccountID sql.NullString

	err = db.DB.QueryRow(`
		SELECT bank_account_number 
		FROM restaurants 
		WHERE id = ?
	`, loginID).Scan(&stripeAccountID)

	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error fetching Stripe account ID:", err)
		utils.JSON(w, 500, false, "Database error", nil)
		return
	}

	// -------------------------------
	// 3. CREATE STRIPE ACCOUNT IF EMPTY
	// -------------------------------
	if !stripeAccountID.Valid || stripeAccountID.String == "" {

		params := &stripe.AccountParams{
			Type:         stripe.String("express"),
			BusinessType: stripe.String("individual"),

			BusinessProfile: &stripe.AccountBusinessProfileParams{
				MCC:                stripe.String("5812"), // Restaurants
				ProductDescription: stripe.String("Restaurant selling food items"),
			},

			Capabilities: &stripe.AccountCapabilitiesParams{
				CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{
					Requested: stripe.Bool(true),
				},
				Transfers: &stripe.AccountCapabilitiesTransfersParams{
					Requested: stripe.Bool(true),
				},
			},
		}

		acc, err := account.New(params)
		if err != nil {
			utils.JSON(w, 500, false, "Failed to create Stripe account", err.Error())
			return
		}

		stripeAccountID = sql.NullString{
			String: acc.ID, // actual string
			Valid:  true,   // marks it as non-null
		}

		// 🔥 ALWAYS UPDATE (NO INSERT)
		_, err = db.DB.Exec(`
			UPDATE restaurants
			SET bank_account_number = ?
			WHERE id = ?
		`, stripeAccountID, loginID)

		if err != nil {
			utils.JSON(w, 500, false, "Failed to update Stripe account ID", nil)
			return
		}
	}

	// -------------------------------
	// 4. GENERATE ONBOARDING URL
	// -------------------------------
	params := &stripe.AccountLinkParams{
		Account:    stripe.String(stripeAccountID.String),
		RefreshURL: stripe.String("https://yourapp.com/reauth"),
		ReturnURL:  stripe.String("https://yourapp.com/account-complete"),
		Type:       stripe.String("account_onboarding"),
	}

	link, err := accountlink.New(params)
	if err != nil {
		utils.JSON(w, 500, false, "Failed to generate onboarding link", err.Error())
		return
	}

	utils.JSON(w, 200, true, "Onboarding URL generated", map[string]interface{}{
		"url":               link.URL,
		"stripe_account_id": stripeAccountID,
	})

}

func GetStripe_Account_details_handler(w http.ResponseWriter, r *http.Request) {

	// -------------------------------
	// 1. Read Authorization Token
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
	loginID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 2. Get Stripe Account ID from DB
	// -------------------------------
	var stripeAccountID sql.NullString

	query := `
		SELECT bank_account_number
		FROM restaurants 
		WHERE id = ?
	`

	err = db.DB.QueryRow(query, loginID).Scan(&stripeAccountID)
	if err == sql.ErrNoRows {
		utils.JSON(w, 404, false, "Stripe account not found for this partner", nil)
		return
	} else if err != nil {
		utils.JSON(w, 500, false, "Database error", nil)
		return
	}

	if !stripeAccountID.Valid {
		utils.JSON(w, 400, false, "Stripe account not created yet", nil)
		return
	}

	// -------------------------------
	// 3. Get Stripe Account From Stripe
	// -------------------------------
	acc, err := account.GetByID(stripeAccountID.String, nil)

	if err != nil {
		utils.JSON(w, 500, false, "Failed to fetch Stripe account", err.Error())
		return
	}

	// -------------------------------
	// 4. Prepare Custom JSON Response
	// -------------------------------
	resp := map[string]interface{}{
		"account_id":      acc.ID,
		"first_name":      acc.Individual.FirstName,
		"last_name":       acc.Individual.LastName,
		"email":           acc.Individual.Email,
		"phone":           acc.Individual.Phone,
		"charges_enabled": acc.ChargesEnabled,
		"payouts_enabled": acc.PayoutsEnabled,
		"requirements": map[string]interface{}{
			"currently_due":        acc.Requirements.CurrentlyDue,
			"eventually_due":       acc.Requirements.EventuallyDue,
			"past_due":             acc.Requirements.PastDue,
			"pending_verification": acc.Requirements.PendingVerification,
			"disabled_reason":      acc.Requirements.DisabledReason,
		},
	}

	// -------------------------------
	// 5. Send Response
	// -------------------------------
	utils.JSON(w, 200, true, "Stripe account details fetched", resp)
}

type UpdateRestaurantAddressRequest struct {
	BusinessAddress string  `json:"business_address"`
	City            string  `json:"city"`
	State           string  `json:"state"`
	Zipcode         string  `json:"zipcode"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
}

func UpdateRestaurantAddress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		utils.JSON(w, http.StatusMethodNotAllowed, false, "Method not allowed", nil)
		return
	}

	// Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.JSON(w, http.StatusUnauthorized, false, "Authorization header missing", nil)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		utils.JSON(w, http.StatusUnauthorized, false, "Invalid token format", nil)
		return
	}

	tokenString := parts[1]
	restaurantID, _, err := utils.ParseToken(tokenString)
	if err != nil || restaurantID == 0 {
		utils.JSON(w, http.StatusUnauthorized, false, "Invalid or expired token", nil)
		return
	}

	var req UpdateRestaurantAddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, false, "Invalid request payload", nil)
		return
	}

	// 🧪 Validation
	if req.BusinessAddress == "" || req.City == "" || req.State == "" || req.Zipcode == "" {
		utils.JSON(w, http.StatusBadRequest, false, "All address fields are required", nil)
		return
	}

	result, err := db.DB.Exec(`
		UPDATE restaurants
		SET business_address = ?,
		    city = ?,
		    state = ?,
		    zipcode = ?,
		    latitude = ?,
		    longitude = ?
		WHERE id = ?
	`,
		req.BusinessAddress,
		req.City,
		req.State,
		req.Zipcode,
		req.Latitude,
		req.Longitude,
		restaurantID,
	)

	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "Failed to update restaurant address", nil)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		utils.JSON(w, http.StatusNotFound, false, "Restaurant not found", nil)
		return
	}

	utils.JSON(w, http.StatusOK, true, "Restaurant address updated successfully", nil)
}

type UpdateRestaurantTimeRequest struct {
	OpenTime  string `json:"open_time"`  // HH:mm
	CloseTime string `json:"close_time"` // HH:mm
}

func UpdateRestaurantTime(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		utils.JSON(w, http.StatusMethodNotAllowed, false, "Method not allowed", nil)
		return
	}

	// 🔐 Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.JSON(w, http.StatusUnauthorized, false, "Authorization header missing", nil)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		utils.JSON(w, http.StatusUnauthorized, false, "Invalid token format", nil)
		return
	}

	tokenString := parts[1]
	restaurantID, _, err := utils.ParseToken(tokenString)
	if err != nil || restaurantID == 0 {
		utils.JSON(w, http.StatusUnauthorized, false, "Invalid or expired token", nil)
		return
	}

	var req UpdateRestaurantTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, false, "Invalid request payload", nil)
		return
	}

	// 🧪 Validate time format
	openTime, err := time.Parse("15:04", req.OpenTime)
	if err != nil {
		utils.JSON(w, http.StatusBadRequest, false, "Invalid open_time format (HH:mm)", nil)
		return
	}

	closeTime, err := time.Parse("15:04", req.CloseTime)
	if err != nil {
		utils.JSON(w, http.StatusBadRequest, false, "Invalid close_time format (HH:mm)", nil)
		return
	}

	// ❌ Close time must be after open time
	if !closeTime.After(openTime) {
		utils.JSON(w, http.StatusBadRequest, false, "close_time must be after open_time", nil)
		return
	}

	result, err := db.DB.Exec(`
		UPDATE restaurants
		SET open_time = ?,
		    close_time = ?,
		    is_open = 1
		WHERE id = ?
	`,
		req.OpenTime,
		req.CloseTime,
		restaurantID,
	)

	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "Failed to update restaurant time", nil)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		utils.JSON(w, http.StatusNotFound, false, "Restaurant not found", nil)
		return
	}

	utils.JSON(w, http.StatusOK, true, "Restaurant open and close time updated successfully", nil)
}

func UpdateRestaurant_cover_photo(w http.ResponseWriter, r *http.Request) {
	// -------------------------------
	// 1. Read Authorization Token
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
	loginID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 2. Parse Form
	// -------------------------------
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}

	imageURL := r.FormValue("cover_image") // ❗ FIXED FIELD NAME

	// -------------------------------
	// 3. Validate
	// -------------------------------
	if imageURL == "" {
		utils.JSON(w, 400, false, "Cover image URL is required", nil)
		return
	}

	// -------------------------------
	// 4. Update DB
	// -------------------------------
	result, err := db.DB.Exec(`
		UPDATE restaurants
		SET cover_image = ?
		WHERE id = ?
	`, imageURL, loginID)

	if err != nil {
		fmt.Println("DB Update Error:", err)
		utils.JSON(w, 500, false, "Failed to update cover image", nil)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		utils.JSON(w, 500, false, "Failed to check update result", nil)
		return
	}

	// -------------------------------
	// 5. Check rows affected
	// -------------------------------
	if rowsAffected == 0 {
		utils.JSON(w, 404, false, "Restaurant not found or image unchanged", nil)
		return
	}

	// -------------------------------
	// 6. Success Response
	// -------------------------------
	utils.JSON(w, 200, true, "Cover image updated successfully", map[string]interface{}{
		"cover_image": imageURL,
	})
}
