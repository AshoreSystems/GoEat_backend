package routes

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/account"
	"github.com/stripe/stripe-go/v76/accountlink"
)

func CreateStripeAccount(w http.ResponseWriter, r *http.Request) {

	// -------------------------------
	// 1. Read Authorization Token
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
	loginID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 2. Create Stripe Connected Account
	// -------------------------------
	params := &stripe.AccountParams{
		Type: stripe.String("express"),
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
		JSON(w, 500, false, "Failed to create Stripe account", err.Error())
		return
	}

	stripeAccountID := acc.ID

	// -------------------------------
	// 3. CHECK IF BANK ACCOUNT ROW EXISTS
	// -------------------------------
	var count int
	checkQuery := `
		SELECT COUNT(*) 
		FROM tbl_partner_bank_accounts 
		WHERE partner_id = ?
	`
	err = db.DB.QueryRow(checkQuery, loginID).Scan(&count)
	if err != nil {
		JSON(w, 500, false, "Database error", nil)
		return
	}

	// -------------------------------
	// 4A. UPDATE EXISTING ROW
	// -------------------------------
	if count > 0 {

		updateQuery := `
			UPDATE tbl_partner_bank_accounts 
			SET stripe_account_id = ?
			WHERE partner_id = ?
		`

		_, err = db.DB.Exec(updateQuery, stripeAccountID, loginID)
		if err != nil {
			JSON(w, 500, false, "Failed to update Stripe account ID", nil)
			return
		}

		JSON(w, 200, true, "Stripe account created & updated", map[string]interface{}{
			"stripe_account_id": stripeAccountID,
		})

		return
	}

	// -------------------------------
	// 4B. INSERT NEW ROW
	// -------------------------------
	insertQuery := `
		INSERT INTO tbl_partner_bank_accounts 
		(partner_id, stripe_account_id)
		VALUES (?, ?)
	`

	_, err = db.DB.Exec(insertQuery, loginID, stripeAccountID)
	if err != nil {
		fmt.Println("Insert error:", err)
		JSON(w, 500, false, "Failed to save Stripe account ID", nil)
		return
	}

	JSON(w, 200, true, "Stripe account created & saved", map[string]interface{}{
		"stripe_account_id": stripeAccountID,
	})
}

func CreateStripeOnboarding(w http.ResponseWriter, r *http.Request) {

	// -------------------------------
	// 1. Read Authorization Token
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
	loginID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 2. CHECK IF ROW EXISTS
	// -------------------------------
	var rowCount int
	checkRow := `
		SELECT COUNT(*) 
		FROM tbl_partner_bank_accounts 
		WHERE partner_id = ?
	`
	err = db.DB.QueryRow(checkRow, loginID).Scan(&rowCount)
	if err != nil {
		JSON(w, 500, false, "Database error", nil)
		return
	}

	rowExists := rowCount > 0

	// -------------------------------
	// 3. GET STRIPE ACCOUNT ID IF ROW EXISTS
	// -------------------------------
	var stripeAccountID string

	if rowExists {
		err = db.DB.QueryRow(`
			SELECT stripe_account_id 
			FROM tbl_partner_bank_accounts 
			WHERE partner_id = ?
		`, loginID).Scan(&stripeAccountID)

		if err != nil {
			fmt.Println("Query error:", err)
			if err == sql.ErrNoRows {
				JSON(w, 404, false, "Stripe account not found for this partner", nil)
				return
			}
			JSON(w, 500, false, "Database error", nil)
			return
		}
	}

	// -------------------------------
	// 4. CREATE STRIPE ACCOUNT ONLY IF ID IS EMPTY
	// -------------------------------
	if stripeAccountID == "" {

		// Create new account
		params := &stripe.AccountParams{
			Type:         stripe.String("express"),
			BusinessType: stripe.String("individual"),

			BusinessProfile: &stripe.AccountBusinessProfileParams{
				MCC:                stripe.String("4215"), // Industry = Courier Services / Delivery
				ProductDescription: stripe.String("On-demand delivery partner"),
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
			JSON(w, 500, false, "Failed to create Stripe account", err.Error())
			return
		}

		stripeAccountID = acc.ID

		// ⬇ If row exists → UPDATE
		if rowExists {
			_, err = db.DB.Exec(`
				UPDATE tbl_partner_bank_accounts 
				SET stripe_account_id = ?
				WHERE partner_id = ?
			`, stripeAccountID, loginID)

		} else {
			// ⬇ If row does NOT exist → INSERT (first time only)
			_, err = db.DB.Exec(`
				INSERT INTO tbl_partner_bank_accounts 
				(partner_id, stripe_account_id)
				VALUES (?, ?)
			`, loginID, stripeAccountID)
		}

		if err != nil {
			JSON(w, 500, false, "Failed to save Stripe account ID", nil)
			return
		}
	}

	// -------------------------------
	// 5. GENERATE ONBOARDING URL
	// -------------------------------
	params := &stripe.AccountLinkParams{
		Account:    stripe.String(stripeAccountID),
		RefreshURL: stripe.String("https://yourapp.com/reauth"),
		ReturnURL:  stripe.String("https://yourapp.com/account-complete"),
		Type:       stripe.String("account_onboarding"),
	}

	link, err := accountlink.New(params)
	if err != nil {
		JSON(w, 500, false, "Failed to generate onboarding link", err.Error())
		return
	}

	JSON(w, 200, true, "Onboarding URL generated", map[string]interface{}{
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
		JSON(w, 401, false, "Authorization header missing", nil)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		JSON(w, 401, false, "Invalid token format", nil)
		return
	}

	tokenString := parts[1]
	loginID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 2. Get Stripe Account ID from DB
	// -------------------------------
	var stripeAccountID sql.NullString

	query := `
		SELECT stripe_account_id
		FROM tbl_partner_bank_accounts 
		WHERE partner_id = ?
	`

	err = db.DB.QueryRow(query, loginID).Scan(&stripeAccountID)
	if err == sql.ErrNoRows {
		JSON(w, 404, false, "Stripe account not found for this partner", nil)
		return
	} else if err != nil {
		JSON(w, 500, false, "Database error", nil)
		return
	}

	if !stripeAccountID.Valid {
		JSON(w, 400, false, "Stripe account not created yet", nil)
		return
	}

	// -------------------------------
	// 3. Get Stripe Account From Stripe
	// -------------------------------
	acc, err := account.GetByID(stripeAccountID.String, nil)

	if err != nil {
		JSON(w, 500, false, "Failed to fetch Stripe account", err.Error())
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
	JSON(w, 200, true, "Stripe account details fetched", resp)
}

func CreatePartnerBankAccountHandler(w http.ResponseWriter, r *http.Request) {

	// -------------------------------
	// 1. Read Authorization Token
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

	holder := r.FormValue("account_holder_name")
	routing := r.FormValue("routing_number")
	account := r.FormValue("account_number")
	bankName := r.FormValue("bank_name")
	accountType := r.FormValue("account_type")
	bankLogoURL := r.FormValue("bank_logo_url")

	// -------------------------------
	// 3. Validation
	// -------------------------------
	if holder == "" {
		JSON(w, 400, false, "Account holder name is required", nil)
		return
	}

	if routing == "" || len(routing) != 9 {
		JSON(w, 400, false, "Routing number must be exactly 9 digits", nil)
		return
	}

	if account == "" {
		JSON(w, 400, false, "Account number is required", nil)
		return
	}

	if accountType != "checking" && accountType != "savings" {
		JSON(w, 400, false, "account_type must be checking or savings", nil)
		return
	}

	if bankLogoURL != "" && !strings.HasPrefix(bankLogoURL, "http") {
		JSON(w, 400, false, "bank_logo_url must be a valid URL", nil)
		return
	}

	// -------------------------------
	// 4. Compute last 4 digits
	// -------------------------------
	last4 := ""
	if len(account) >= 4 {
		last4 = account[len(account)-4:]
	}

	// -------------------------------
	// 5. Encrypt Account Number
	// -------------------------------
	key := []byte(os.Getenv("ENCRYPTION_KEY"))
	encryptedAcc, err := utils.Encrypt(account, key)
	if err != nil {
		fmt.Println("Encryption error:", err)
		JSON(w, 500, false, "Failed to encrypt account number", nil)
		return
	}

	// -------------------------------
	// 6. CHECK IF BANK ACCOUNT EXISTS FOR THIS PARTNER
	// -------------------------------
	var count int
	checkQuery := `
		SELECT COUNT(*) 
		FROM tbl_partner_bank_accounts 
		WHERE partner_id = ?
	`
	err = db.DB.QueryRow(checkQuery, loginID).Scan(&count)
	if err != nil {
		JSON(w, 500, false, "Database error", nil)
		return
	}

	// -------------------------------
	// 7A. UPDATE EXISTING BANK ACCOUNT
	// -------------------------------
	if count > 0 {

		updateQuery := `
			UPDATE tbl_partner_bank_accounts 
			SET account_holder_name = ?, routing_number = ?, account_number = ?, 
			    last4 = ?, bank_name = ?, bank_logo_url = ?, account_type = ?
			WHERE partner_id = ?
		`

		_, err = db.DB.Exec(updateQuery,
			holder,
			routing,
			encryptedAcc,
			last4,
			bankName,
			bankLogoURL,
			accountType,
			loginID,
		)

		if err != nil {
			fmt.Println("Update error:", err)
			JSON(w, 500, false, "Failed to update bank details", nil)
			return
		}

		JSON(w, 200, true, "Bank account updated successfully", map[string]interface{}{
			"last4":        last4,
			"bank_name":    bankName,
			"bank_logo":    bankLogoURL,
			"account_type": accountType,
		})

		return
	}

	// -------------------------------
	// 7B. INSERT NEW BANK ACCOUNT
	// -------------------------------
	insertQuery := `
		INSERT INTO tbl_partner_bank_accounts 
		(partner_id, account_holder_name, routing_number, account_number, last4, bank_name, bank_logo_url, account_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = db.DB.Exec(insertQuery,
		loginID,
		holder,
		routing,
		encryptedAcc,
		last4,
		bankName,
		bankLogoURL,
		accountType,
	)

	if err != nil {
		fmt.Println("Insert error:", err)
		JSON(w, 500, false, "Failed to save bank details", nil)
		return
	}

	JSON(w, 200, true, "Bank account added successfully", map[string]interface{}{
		"last4":        last4,
		"bank_name":    bankName,
		"bank_logo":    bankLogoURL,
		"account_type": accountType,
	})
}
