package routes

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"fmt"
	"net/http"
	"os"
	"strings"
)

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
