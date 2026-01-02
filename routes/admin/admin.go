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
	currentYear := time.Now().Year()
	currentMonth := int(time.Now().Month())

	months := []string{}
	for m := 1; m <= currentMonth; m++ {
		month := time.Date(currentYear, time.Month(m), 1, 0, 0, 0, 0, time.UTC).
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
	`, string(newHashedPassword), time.Now(), loginID)

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
