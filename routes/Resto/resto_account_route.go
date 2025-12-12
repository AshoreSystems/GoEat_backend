package resto

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

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
