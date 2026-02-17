package Admin

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"GoEatsapi/db"
	"GoEatsapi/utils"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

func Get_Admin_Dashboard_Graph(w http.ResponseWriter, r *http.Request) {

	// ================= AUTH =================
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

	// Only validate token (no role check)
	_, _, err := utils.ParseToken(tokenString)
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// ================= COMMON =================
	currentTime := utils.GetISTTime()
	currentYear := currentTime.Year()
	currentMonth := int(currentTime.Month())

	months := []string{}
	for m := 1; m <= currentMonth; m++ {
		month := time.Date(currentYear, time.Month(m), 1, 0, 0, 0, 0, utils.ISTLocation).
			Format("2006-01")
		months = append(months, month)
	}

	// ================= ORDERS GRAPH =================
	type OrderGraphRow struct {
		Month       string  `json:"month"`
		TotalOrders int     `json:"total_orders"`
		TotalAmount float64 `json:"total_amount"`
	}

	orderQuery := `
		SELECT 
			DATE_FORMAT(created_at, '%Y-%m') AS month,
			COUNT(*) AS total_orders,
			COALESCE(SUM(subtotal), 0) AS total_amount
		FROM tbl_orders
		WHERE status = 'delivered'
		  AND YEAR(created_at) = YEAR(CURDATE())
		GROUP BY DATE_FORMAT(created_at, '%Y-%m')
		ORDER BY month;
	`
	orderRows, err := db.DB.Query(orderQuery)
	if err != nil {
		utils.JSON(w, 500, false, "DB error", nil)
		return
	}
	defer orderRows.Close()

	orderData := make(map[string]OrderGraphRow)

	for orderRows.Next() {
		var g OrderGraphRow
		if err := orderRows.Scan(&g.Month, &g.TotalOrders, &g.TotalAmount); err != nil {
			utils.JSON(w, 500, false, "Scan error", nil)
			return
		}
		orderData[g.Month] = g
	}

	ordersGraph := []OrderGraphRow{}
	totalOrders := 0
	totalAmount := 0.0

	for _, m := range months {
		if val, ok := orderData[m]; ok {
			ordersGraph = append(ordersGraph, val)
			totalOrders += val.TotalOrders
			totalAmount += val.TotalAmount
		} else {
			ordersGraph = append(ordersGraph, OrderGraphRow{
				Month:       m,
				TotalOrders: 0,
				TotalAmount: 0,
			})
		}
	}

	// ================= USERS REGISTRATION GRAPH =================
	type UserGraphRow struct {
		Month    string `json:"month"`
		NewUsers int    `json:"new_users"`
	}

	userQuery := `
		SELECT 
			DATE_FORMAT(created_at, '%Y-%m') AS month,
			COUNT(*) AS new_users
		FROM customer
		WHERE YEAR(created_at) = YEAR(CURDATE())
		GROUP BY DATE_FORMAT(created_at, '%Y-%m')
		ORDER BY month;
	`

	userRows, err := db.DB.Query(userQuery)
	if err != nil {
		utils.JSON(w, 500, false, "DB error", nil)
		return
	}
	defer userRows.Close()

	userData := make(map[string]int)

	for userRows.Next() {
		var month string
		var count int
		if err := userRows.Scan(&month, &count); err != nil {
			utils.JSON(w, 500, false, "Scan error", nil)
			return
		}
		userData[month] = count
	}

	usersGraph := []UserGraphRow{}
	totalNewUsers := 0

	for _, m := range months {
		count := userData[m]
		usersGraph = append(usersGraph, UserGraphRow{
			Month:    m,
			NewUsers: count,
		})
		totalNewUsers += count
	}
	rows, err := db.DB.Query(`
	SELECT 
		o.id,
		o.order_number,
		o.customer_id,
		c.full_name,
		o.subtotal,
		o.tax_amount,
		o.delivery_fee,
		o.tip_amount,
		o.total_amount,
		o.payment_method,
		o.payment_status,
		o.status,
		o.order_placed_at,
		r.restaurant_name,
		a.address,
		a.city
	FROM tbl_orders o
	JOIN customer c ON c.id = o.customer_id
	JOIN restaurants r ON r.id = o.restaurant_id
	JOIN customer_delivery_addresses a ON a.id = o.address_id
	WHERE DATE(o.order_placed_at) = CURDATE()
	ORDER BY o.id DESC
`)

	if err != nil {
		fmt.Println("Failed to fetch delivered orders:", err)
		utils.JSON(w, 500, false, "Failed to fetch orders", nil)
		return
	}
	defer rows.Close()

	type OrderItem struct {
		ID    int     `json:"id"`
		Title string  `json:"title"`
		Qty   int     `json:"qty"`
		Price float64 `json:"price"`
		Image string  `json:"image_url"`
	}

	type Order struct {
		ID            int     `json:"id"`
		OrderNumber   string  `json:"order_number"`
		CustomerID    int     `json:"customer_id"`
		CustomerName  string  `json:"full_name"`
		Subtotal      float64 `json:"subtotal"`
		TaxAmount     float64 `json:"tax_amount"`
		DeliveryFee   float64 `json:"delivery_fee"`
		TipAmount     float64 `json:"tip_amount"`
		TotalAmount   float64 `json:"total_amount"`
		PaymentMethod string  `json:"payment_method"`
		PaymentStatus string  `json:"payment_status"`
		Status        string  `json:"status"`
		OrderPlacedAt string  `json:"order_placed_at"`

		RestaurantName  string `json:"restaurant_name"`
		DeliveryAddress string `json:"delivery_address"`
		City            string `json:"city"`

		Items []OrderItem `json:"items"`
	}

	orders := []Order{}

	for rows.Next() {
		var o Order
		if err := rows.Scan(
			&o.ID,
			&o.OrderNumber,
			&o.CustomerID,
			&o.CustomerName,
			&o.Subtotal,
			&o.TaxAmount,
			&o.DeliveryFee,
			&o.TipAmount,
			&o.TotalAmount,
			&o.PaymentMethod,
			&o.PaymentStatus,
			&o.Status,
			&o.OrderPlacedAt,
			&o.RestaurantName,
			&o.DeliveryAddress,
			&o.City,
		); err != nil {
			utils.JSON(w, 500, false, "Failed to scan order", nil)
			return
		}

		itemRows, _ := db.DB.Query(`
		SELECT id, COALESCE(title,''), qty, price, COALESCE(image_url,'')
		FROM tbl_order_items
		WHERE order_id = ?
	`, o.ID)

		for itemRows.Next() {
			var item OrderItem
			itemRows.Scan(&item.ID, &item.Title, &item.Qty, &item.Price, &item.Image)
			o.Items = append(o.Items, item)
		}
		itemRows.Close()

		orders = append(orders, o)
	}
	// ================= RESPONSE =================
	response := map[string]interface{}{
		"orders_graph": ordersGraph,
		"users_graph":  usersGraph,
		"summary": map[string]interface{}{
			"total_orders":   totalOrders,
			"total_amount":   totalAmount,
			"new_users_year": totalNewUsers,
		},
		"orders": orders,
	}

	utils.JSON(w, 200, true, "Success", response)
}

func AdimnLogin(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Method not allowed", nil)
		return
	}
	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("ParseForm error:", err)
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}
	r.ParseForm()

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate
	if email == "" || password == "" {
		fmt.Println("Email or password is empty", email, password)
		utils.JSON(w, 400, false, "Email and password are required", nil)
		return
	}

	// ---------------------------------------------
	// 🔍 1. Find user by email
	// ---------------------------------------------
	var storedID int
	var storedName string
	var storedPassword string

	err = db.DB.QueryRow(`
		SELECT id, name, password
		FROM login
		WHERE email = ?
	`, email).Scan(&storedID, &storedName, &storedPassword)

	if err != nil {
		utils.JSON(w, 400, false, "Invalid email or password", nil)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		utils.JSON(w, 400, false, "Invalid email or password", nil)
		return
	}

	token, err := utils.GenerateToken(storedID, email)
	if err != nil {
		utils.JSON(w, 500, false, "Failed to generate token", nil)
		return
	}

	// ---------------------------------------------
	// 🎉 3. Successful login
	// ---------------------------------------------
	utils.JSON(w, 200, true, "Login successful", map[string]interface{}{
		"token": token,
	})
}

func Get_Admin_Notifications(w http.ResponseWriter, r *http.Request) {
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

	_, _, err := utils.ParseToken(parts[1])
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	type Notification struct {
		ID        int    `json:"id"`
		Type      string `json:"type"`
		Message   string `json:"message"`
		Status    string `json:"status,omitempty"`
		CreatedAt string `json:"created_at"`
	}

	notifications := []Notification{}

	// Delivery Partners - New onboarding requests
	partnerRows, err := db.DB.Query(`
		SELECT id, COALESCE(first_name, ''), COALESCE(last_name, ''), created_at
		FROM delivery_partners
		WHERE DATE(created_at) = CURDATE()
		ORDER BY created_at DESC
	`)
	if err == nil {
		defer partnerRows.Close()
		for partnerRows.Next() {
			var id int
			var firstName, lastName, createdAt string
			if partnerRows.Scan(&id, &firstName, &lastName, &createdAt) == nil {
				notifications = append(notifications, Notification{
					ID:        id,
					Type:      "delivery_partner",
					Message:   fmt.Sprintf("New delivery partner onboarding: %s %s", firstName, lastName),
					CreatedAt: createdAt,
				})
			}
		}
	}

	// Restaurants - New onboarding requests
	restaurantRows, err := db.DB.Query(`
		SELECT id, COALESCE(restaurant_name, ''), created_at
		FROM restaurants
		WHERE DATE(created_at) = CURDATE()
		ORDER BY created_at DESC
	`)
	if err == nil {
		defer restaurantRows.Close()
		for restaurantRows.Next() {
			var id int
			var name, createdAt string
			if restaurantRows.Scan(&id, &name, &createdAt) == nil {
				notifications = append(notifications, Notification{
					ID:        id,
					Type:      "restaurant",
					Message:   fmt.Sprintf("New restaurant onboarding: %s", name),
					CreatedAt: createdAt,
				})
			}
		}
	}

	// Orders - New orders and status changes
	orderRows, err := db.DB.Query(`
		SELECT id, order_number, status, created_at
		FROM tbl_orders
		WHERE DATE(created_at) = CURDATE()
		ORDER BY created_at DESC
	`)
	if err == nil {
		defer orderRows.Close()
		for orderRows.Next() {
			var id int
			var orderNumber, status, createdAt string
			if orderRows.Scan(&id, &orderNumber, &status, &createdAt) == nil {
				notifications = append(notifications, Notification{
					ID:        id,
					Type:      "order",
					Message:   fmt.Sprintf("Order %s", orderNumber),
					Status:    status,
					CreatedAt: createdAt,
				})
			}
		}
	}

	message := "Success"
	if len(notifications) == 0 {
		message = "No notifications found for today"
	}

	utils.JSON(w, 200, true, message, map[string]interface{}{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

func Get_partners_list(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.DB.Query(`
		SELECT id, COALESCE(first_name, ''),
		COALESCE(last_name, ''),
		COALESCE(email, ''),
		COALESCE(primary_mobile, ''),
		COALESCE(status, '')
		FROM delivery_partners 
		WHERE profile_completed = 1
	`)
	if err != nil {
		fmt.Println("Failed to fetch partners:", err)
		utils.JSON(w, 500, false, "Failed to fetch partners", nil)
		return
	}
	defer rows.Close()

	type Partner struct {
		ID        int    `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		Phone     string `json:"primary_mobile"`
		Status    string `json:"status"`
	}

	partners := []Partner{}

	for rows.Next() {
		var p Partner
		if err := rows.Scan(&p.ID, &p.FirstName, &p.LastName, &p.Email, &p.Phone, &p.Status); err != nil {
			fmt.Println("Failed to scan partner:", err)
			utils.JSON(w, 500, false, "Failed to scan partner", nil)
			return
		}
		partners = append(partners, p)
	}

	utils.JSON(w, 200, true, "Partners list", partners)

}

func Get_restaurants_list(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.DB.Query(`
		SELECT id, restaurant_name, email, phone_number, status
		FROM restaurants
	`)
	if err != nil {
		fmt.Println("Failed to fetch restaurants:", err)
		utils.JSON(w, 500, false, "Failed to fetch restaurants", nil)
		return
	}
	defer rows.Close()

	type Restaurant struct {
		ID     int    `json:"id"`
		Name   string `json:"restaurant_name"`
		Email  string `json:"email"`
		Phone  string `json:"phone_number"`
		Status string `json:"status"`
	}

	var restaurants []Restaurant

	for rows.Next() {
		var r Restaurant
		if err := rows.Scan(&r.ID, &r.Name, &r.Email, &r.Phone, &r.Status); err != nil {
			fmt.Println("Failed to scan restaurant:", err)
			utils.JSON(w, 500, false, "Failed to scan restaurant", nil)
			return
		}
		restaurants = append(restaurants, r)
	}

	utils.JSON(w, 200, true, "Restaurants list", restaurants)
}

func Update_request_status_of_restaurant(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Method not allowed", nil)
		return
	}
	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("ParseForm error:", err)
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}
	r.ParseForm()

	id := r.FormValue("id")
	status := r.FormValue("status")

	// Validate
	if id == "" || status == "" {
		fmt.Println("id or status is empty", id, status)
		utils.JSON(w, 400, false, "id and status are required", nil)
		return
	}
	fmt.Println("Update request status of partner:", id, status)
	_, err = db.DB.Exec(`
		UPDATE restaurants
		SET status = ?
		WHERE id = ?
	`, status, id)
	if err != nil {
		fmt.Println("Failed to update request status of partner:", err)
		utils.JSON(w, 500, false, "Failed to update request status of partner", nil)
		return
	}

	utils.JSON(w, 200, true, "Request status updated", nil)
}

func Update_request_status_of_partner(w http.ResponseWriter, r *http.Request) {
	// Only POST allowed
	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Method not allowed", nil)
		return
	}
	// Parse form
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		fmt.Println("ParseForm error:", err)
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}
	r.ParseForm()

	id := r.FormValue("id")
	status := r.FormValue("status")

	// Validate
	if id == "" || status == "" {
		fmt.Println("id or status is empty", id, status)
		utils.JSON(w, 400, false, "id and status are required", nil)
		return
	}
	fmt.Println("Update request status of partner:", id, status)
	_, err = db.DB.Exec(`
		UPDATE delivery_partners
		SET status = ?
		WHERE id = ?
	`, status, id)
	if err != nil {
		fmt.Println("Failed to update request status of partner:", err)
		utils.JSON(w, 500, false, "Failed to update request status of partner", nil)
		return
	}

	utils.JSON(w, 200, true, "Request status updated", nil)
}

func GetPartnerDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Method check
	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Invalid request method", nil)
		return
	}

	// Auth check
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

	// Request body
	var req struct {
		PartnerID uint64 `json:"partner_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, 400, false, "Invalid request body", nil)
		return
	}

	if req.PartnerID == 0 {
		utils.JSON(w, 400, false, "partner_id is required", nil)
		return
	}

	// Response struct
	type PartnerDetails struct {
		ID                   uint64 `json:"id"`
		FirstName            string `json:"first_name"`
		LastName             string `json:"last_name"`
		PrimaryMobile        string `json:"primary_mobile"`
		Email                string `json:"email"`
		Gender               string `json:"gender"`
		City                 string `json:"city"`
		FullAddress          string `json:"full_address"`
		LanguagesKnown       string `json:"languages_known"`
		Status               string `json:"status"`
		ProfileCompleted     int    `json:"profile_completed"`
		ProfilePhotoURL      string `json:"profile_photo_url"`
		DrivingLicenseURL    string `json:"driving_license_url"`
		DrivingLicenseNumber string `json:"driving_license_number"`
		DrivingLicenseExpire string `json:"driving_license_expire"`
		CreatedAt            string `json:"created_at"`
	}

	var partner PartnerDetails

	query := `
		SELECT 
			id,
			COALESCE(first_name,''),
			COALESCE(last_name,''),
			COALESCE(primary_mobile,''),
			COALESCE(email,''),
			COALESCE(gender,''),
			COALESCE(city,''),
			COALESCE(full_address,''),
			COALESCE(languages_known,''),
			status,
			profile_completed,
			COALESCE(profile_photo_url,''),
			COALESCE(driving_license_url,''),
			COALESCE(driving_license_number,''),
			COALESCE(driving_license_expire,''),
			created_at
		FROM delivery_partners
		WHERE id = ?
	`

	err := db.DB.QueryRow(query, req.PartnerID).Scan(
		&partner.ID,
		&partner.FirstName,
		&partner.LastName,
		&partner.PrimaryMobile,
		&partner.Email,
		&partner.Gender,
		&partner.City,
		&partner.FullAddress,
		&partner.LanguagesKnown,
		&partner.Status,
		&partner.ProfileCompleted,
		&partner.ProfilePhotoURL,
		&partner.DrivingLicenseURL,
		&partner.DrivingLicenseNumber,
		&partner.DrivingLicenseExpire,
		&partner.CreatedAt,
	)

	if err == sql.ErrNoRows {
		utils.JSON(w, 404, false, "Partner not found", nil)
		return
	} else if err != nil {
		fmt.Println("DB Error:", err)
		utils.JSON(w, 500, false, "Failed to fetch partner details", nil)
		return
	}

	utils.JSON(w, 200, true, "Partner details", partner)
}

func UpdateAdminPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		utils.JSON(w, 405, false, "Invalid request method", nil)
		return
	}

	// Authorization check
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

	// Parse JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError("unexpected signing method", jwt.ValidationErrorSignatureInvalid)
		}
		return []byte("goeats-v01"), nil
	})

	if err != nil || !token.Valid {
		utils.JSON(w, 401, false, "Invalid token", nil)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["login_id"] == nil {
		utils.JSON(w, 401, false, "Invalid token claims", nil)
		return
	}

	loginID := int(claims["login_id"].(float64)) // JWT stores numbers as float64

	// Request payload
	type Request struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, 400, false, "Invalid JSON payload", nil)
		return
	}

	if req.OldPassword == "" || req.NewPassword == "" {
		utils.JSON(w, 400, false, "old_password and new_password are required", nil)
		return
	}

	// Fetch current hashed password from DB
	var hashedPassword string
	err = db.DB.QueryRow("SELECT password FROM login WHERE id = ?", loginID).Scan(&hashedPassword)
	if err != nil {
		utils.JSON(w, 404, false, "User not found", nil)
		return
	}

	// Compare old password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.OldPassword)); err != nil {
		utils.JSON(w, 401, false, "Old password is incorrect", nil)
		return
	}

	// Hash new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.JSON(w, 500, false, "Failed to hash new password", nil)
		return
	}

	// Update password in DB
	result, err := db.DB.Exec(`
		UPDATE login
		SET password = ?, updated_at = ?
		WHERE id = ?
	`, string(newHashedPassword), utils.GetISTTimeString(), loginID)

	if err != nil {
		utils.JSON(w, 500, false, "Database update failed", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.JSON(w, 404, false, "User record not found", nil)
		return
	}

	utils.JSON(w, 200, true, "Password updated successfully", nil)
}

func GetTransactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		utils.JSON(w, 405, false, "Invalid request method", nil)
		return
	}

	// 🔐 Authorization check
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

	// Parse JWT
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError(
				"unexpected signing method",
				jwt.ValidationErrorSignatureInvalid,
			)
		}
		return []byte("goeats-v01"), nil
	})

	if err != nil || !token.Valid {
		utils.JSON(w, 401, false, "Invalid token", nil)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["login_id"] == nil {
		utils.JSON(w, 401, false, "Invalid token claims", nil)
		return
	}

	// ✅ Admin authenticated (login_id available)
	_ = int(claims["login_id"].(float64))

	// 📊 Query transactions
	rows, err := db.DB.Query(`
		SELECT
			t.id,
			o.order_number,
			t.transaction_reference,
			t.payment_mode,
			t.provider,
			t.amount,
			t.currency,
			t.status AS transaction_status,
			DATE_FORMAT(t.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
		FROM tbl_payment_transactions t
		JOIN tbl_orders o ON o.id = t.order_id
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		utils.JSON(w, 500, false, "Failed to fetch transactions", nil)
		return
	}
	defer rows.Close()

	type Transaction struct {
		ID                   uint64  `json:"id"`
		OrderNumber          string  `json:"order_number"`
		TransactionReference string  `json:"transaction_reference"`
		PaymentMode          string  `json:"payment_mode"`
		Provider             string  `json:"provider"`
		Amount               float64 `json:"amount"`
		Currency             string  `json:"currency"`
		TransactionStatus    string  `json:"transaction_status"`
		PaidAt               *string `json:"updated_at"`
	}

	var transactions []Transaction
	var totalIncome float64
	var totalLoss float64

	for rows.Next() {
		var t Transaction

		err := rows.Scan(
			&t.ID,
			&t.OrderNumber,
			&t.TransactionReference,
			&t.PaymentMode,
			&t.Provider,
			&t.Amount,
			&t.Currency,
			&t.TransactionStatus,
			&t.PaidAt,
		)
		if err != nil {
			continue
		}

		// 💰 Summary calculation
		switch t.TransactionStatus {
		case "success":
			totalIncome += t.Amount
		case "refunded":
			totalLoss += t.Amount
		}

		transactions = append(transactions, t)
	}
	var grandAdminTotal float64

	err = db.DB.QueryRow(`
	SELECT 
		IFNULL(
			SUM((oi.price * oi.qty * 0.20)) 
			+ (COUNT(DISTINCT oi.order_id) * 5),
		0)
	FROM tbl_order_items oi
	JOIN tbl_orders o ON o.id = oi.order_id
`).Scan(&grandAdminTotal)

	if err != nil {
		utils.JSON(w, 500, false, "Failed to calculate admin total", nil)
		return
	}

	response := map[string]interface{}{
		"summary": map[string]float64{
			"total_income": totalIncome,
			"total_loss":   totalLoss,
			"admin_total":  grandAdminTotal,
		},
		"transactions": transactions,
	}

	utils.JSON(w, 200, true, "Transactions fetched successfully", response)
}

func GetAllContactRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		utils.JSON(w, 405, false, "Invalid request method", nil)
		return
	}

	// 🔐 Authorization check
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

	// 🔑 Parse JWT
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError(
				"unexpected signing method",
				jwt.ValidationErrorSignatureInvalid,
			)
		}
		return []byte("goeats-v01"), nil
	})

	if err != nil || !token.Valid {
		utils.JSON(w, 401, false, "Invalid token", nil)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["login_id"] == nil {
		utils.JSON(w, 401, false, "Invalid token claims", nil)
		return
	}

	// ✅ Admin authenticated
	_ = int(claims["login_id"].(float64))

	// 📩 Fetch contact requests
	rows, err := db.DB.Query(`
		SELECT
			id,
			user_type,
			user_id,
			name,
			email,
			phone,
			message,
			status,
			DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		FROM tbl_contact_us
		ORDER BY created_at DESC
	`)
	if err != nil {
		utils.JSON(w, 500, false, "Failed to fetch contact requests", nil)
		return
	}
	defer rows.Close()

	type ContactRequest struct {
		ID        uint64  `json:"id"`
		UserType  string  `json:"user_type"`
		UserID    *uint64 `json:"user_id"`
		Name      string  `json:"name"`
		Email     string  `json:"email"`
		Phone     *string `json:"phone"`
		Message   string  `json:"message"`
		Status    string  `json:"status"`
		CreatedAt string  `json:"created_at"`
	}

	var requests []ContactRequest

	for rows.Next() {
		var c ContactRequest

		err := rows.Scan(
			&c.ID,
			&c.UserType,
			&c.UserID,
			&c.Name,
			&c.Email,
			&c.Phone,
			&c.Message,
			&c.Status,
			&c.CreatedAt,
		)
		if err != nil {
			continue
		}

		requests = append(requests, c)
	}

	response := map[string]interface{}{
		"total_count": len(requests),
		"requests":    requests,
	}

	utils.JSON(w, 200, true, "Contact requests fetched successfully", response)
}
