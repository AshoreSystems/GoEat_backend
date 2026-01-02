package Admin

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func GetCustomerList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		utils.JSON(w, 405, false, "Invalid request method", nil)
		return
	}

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
		SELECT 
			c.id, c.full_name, c.email, c.country_code, c.phone_number,
			c.dob, c.profile_image, c.created_at, c.updated_at,

			l.id, l.type, l.status, l.email_verified, l.created_at
		FROM customer c
		LEFT JOIN login l ON c.login_id = l.id
		ORDER BY c.id DESC
	`)
	if err != nil {
		utils.JSON(w, 500, false, "Database error", nil)
		return
	}
	defer rows.Close()

	type LoginDetails struct {
		LoginID        *int    `json:"login_id"`
		Type           *string `json:"type"`
		Status         *string `json:"status"`
		EmailVerified  *bool   `json:"email_verified"`
		LoginCreatedAt *string `json:"login_created_at"`
	}
	type Customer struct {
		ID           int          `json:"id"`
		FullName     string       `json:"full_name"`
		Email        string       `json:"email"`
		CountryCode  string       `json:"country_code"`
		PhoneNumber  string       `json:"phone_number"`
		DOB          *string      `json:"dob"`
		ProfileImage *string      `json:"profile_image"`
		CreatedAt    string       `json:"created_at"`
		UpdatedAt    string       `json:"updated_at"`
		Login        LoginDetails `json:"login"`
	}

	var customers []Customer

	for rows.Next() {
		var c Customer
		var loginID *int
		var loginType, loginStatus *string
		var emailVerified *bool
		var loginCreatedAt *string

		err := rows.Scan(
			&c.ID,
			&c.FullName,
			&c.Email,
			&c.CountryCode,
			&c.PhoneNumber,
			&c.DOB,
			&c.ProfileImage,
			&c.CreatedAt,
			&c.UpdatedAt,
			&loginID,
			&loginType,
			&loginStatus,
			&emailVerified,
			&loginCreatedAt,
		)
		if err != nil {
			continue
		}

		c.Login = LoginDetails{
			LoginID:        loginID,
			Type:           loginType,
			Status:         loginStatus,
			EmailVerified:  emailVerified,
			LoginCreatedAt: loginCreatedAt,
		}

		customers = append(customers, c)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    true,
		"count":     len(customers),
		"customers": customers,
	})
}

func GetCustomerDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSON(w, 405, false, "Invalid request method", nil)
		return
	}

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

	var req struct {
		CustomerID int `json:"customer_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CustomerID == 0 {
		utils.JSON(w, 400, false, "Invalid customer_id", nil)
		return
	}

	/* =======================
	   0️⃣ Customer Profile
	======================= */
	row := db.DB.QueryRow(`
		SELECT 
			id,
			full_name,
			email,
			country_code,
			phone_number,
			dob,
			profile_image,
			created_at
		FROM customer
		WHERE id = ?
	`, req.CustomerID)

	type Customer struct {
		ID           int     `json:"id"`
		FullName     string  `json:"full_name"`
		Email        string  `json:"email"`
		CountryCode  string  `json:"country_code"`
		PhoneNumber  string  `json:"phone_number"`
		DOB          *string `json:"dob"`
		ProfileImage *string `json:"profile_image"`
		CreatedAt    string  `json:"created_at"`
	}

	var customer Customer
	if err := row.Scan(
		&customer.ID,
		&customer.FullName,
		&customer.Email,
		&customer.CountryCode,
		&customer.PhoneNumber,
		&customer.DOB,
		&customer.ProfileImage,
		&customer.CreatedAt,
	); err != nil {
		utils.JSON(w, 404, false, "Customer not found", nil)
		return
	}

	/* =======================
	   1️⃣ Customer Addresses
	======================= */

	type Address struct {
		ID         int      `json:"id"`
		FullName   string   `json:"full_name"`
		Phone      string   `json:"phone_number"`
		Address    string   `json:"address"`
		City       string   `json:"city"`
		State      string   `json:"state"`
		Country    string   `json:"country"`
		PostalCode string   `json:"postal_code"`
		Latitude   *float64 `json:"latitude"`
		Longitude  *float64 `json:"longitude"`
		IsDefault  int      `json:"is_default"`
	}
	addresses := []Address{} // ✅ IMPORTANT

	addressRows, err := db.DB.Query(`
	SELECT 
		id,
		COALESCE(full_name,''),
		COALESCE(phone_number,''),
		address,
		city,
		COALESCE(state,''),
		COALESCE(country,''),
		COALESCE(postal_code,''),
		latitude,
		longitude,
		is_default
	FROM customer_delivery_addresses
	WHERE customer_id = ?
	ORDER BY is_default DESC, id DESC
`, req.CustomerID)
	if err != nil {
		utils.JSON(w, 500, false, "Failed to fetch addresses", nil)
		return
	}
	defer addressRows.Close()

	for addressRows.Next() {
		var a Address
		addressRows.Scan(
			&a.ID,
			&a.FullName,
			&a.Phone,
			&a.Address,
			&a.City,
			&a.State,
			&a.Country,
			&a.PostalCode,
			&a.Latitude,
			&a.Longitude,
			&a.IsDefault,
		)
		addresses = append(addresses, a)
	}

	// if err != nil {
	// 	utils.JSON(w, 500, false, "Failed to fetch addresses", nil)
	// 	return
	// }
	defer addressRows.Close()

	for addressRows.Next() {
		var a Address
		addressRows.Scan(
			&a.ID,
			&a.FullName,
			&a.Phone,
			&a.Address,
			&a.City,
			&a.State,
			&a.Country,
			&a.PostalCode,
			&a.Latitude,
			&a.Longitude,
			&a.IsDefault,
		)
		addresses = append(addresses, a)
	}

	/* =======================
	   2️⃣ Orders with Items
	======================= */
	type OrderItem struct {
		ID    int     `json:"id"`
		Title string  `json:"title"`
		Qty   int     `json:"qty"`
		Price float64 `json:"price"`
		Image string  `json:"image_url"`
	}

	type Order struct {
		ID              int         `json:"id"`
		OrderNumber     string      `json:"order_number"`
		Subtotal        float64     `json:"subtotal"`
		TaxAmount       float64     `json:"tax_amount"`
		DeliveryFee     float64     `json:"delivery_fee"`
		TipAmount       float64     `json:"tip_amount"`
		TotalAmount     float64     `json:"total_amount"`
		PaymentMethod   string      `json:"payment_method"`
		PaymentStatus   string      `json:"payment_status"`
		Status          string      `json:"status"`
		OrderPlacedAt   string      `json:"order_placed_at"`
		RestaurantName  string      `json:"restaurant_name"`
		DeliveryAddress string      `json:"delivery_address"`
		City            string      `json:"city"`
		Items           []OrderItem `json:"items"`
	}
	orders := []Order{}

	orderRows, err := db.DB.Query(`
		SELECT 
			o.id,
			o.order_number,
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
		JOIN restaurants r ON r.id = o.restaurant_id
		JOIN customer_delivery_addresses a ON a.id = o.address_id
		WHERE o.customer_id = ?
		ORDER BY o.id DESC
	`, req.CustomerID)
	if err != nil {
		utils.JSON(w, 500, false, "Failed to fetch orders", nil)
		return
	}
	defer orderRows.Close()

	for orderRows.Next() {
		var o Order
		orderRows.Scan(
			&o.ID,
			&o.OrderNumber,
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
		)

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

	/* =======================
	   Final Response
	======================= */
	utils.JSON(w, 200, true, "Customer details fetched", map[string]interface{}{
		"customer":  customer,
		"addresses": addresses,
		"orders":    orders,
	})
}

func GetDeliveredOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, 405, false, "Invalid request method", nil)
		return
	}

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
		SELECT 
			o.id,
			o.order_number,
			o.customer_id,
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
		JOIN restaurants r ON r.id = o.restaurant_id
		JOIN customer_delivery_addresses a ON a.id = o.address_id
		WHERE o.status = 'delivered'
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

	utils.JSON(w, 200, true, "Delivered orders list", orders)

}

func GetRestaurantDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	// Request body
	var req struct {
		RestaurantID int64 `json:"restaurant_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RestaurantID == 0 {
		utils.JSON(w, 400, false, "Invalid restaurant_id", nil)
		return
	}

	// Response struct
	type Restaurant struct {
		ID                  int64    `json:"id"`
		RestaurantName      string   `json:"restaurant_name"`
		BusinessOwnerName   string   `json:"business_owner_name"`
		Email               string   `json:"email"`
		PhoneNumber         string   `json:"phone_number"`
		BusinessAddress     string   `json:"business_address"`
		City                string   `json:"city"`
		State               string   `json:"state"`
		Zipcode             string   `json:"zipcode"`
		Latitude            *float64 `json:"latitude"`
		Longitude           *float64 `json:"longitude"`
		BusinessDescription string   `json:"business_description"`
		CoverImage          string   `json:"cover_image"`
		Status              string   `json:"status"`
		IsVerified          int      `json:"is_verified"`
		Rating              float64  `json:"rating"`
		OpenTime            string   `json:"open_time"`
		CloseTime           string   `json:"close_time"`
		IsOpen              int      `json:"is_open"`
		MinimumOrderAmount  float64  `json:"minimum_order_amount"`
		CreatedAt           string   `json:"created_at"`
	}

	var restaurant Restaurant

	err := db.DB.QueryRow(`
		SELECT 
			id,
			restaurant_name,
			business_owner_name,
			email,
			phone_number,
			business_address,
			city,
			state,
			zipcode,
			latitude,
			longitude,
			COALESCE(business_description,''),
			COALESCE(cover_image,''),
			status,
			is_verified,
			rating,
			COALESCE(open_time,''),
			COALESCE(close_time,''),
			is_open,
			minimum_order_amount,
			created_at
		FROM restaurants
		WHERE id = ?
	`, req.RestaurantID).Scan(
		&restaurant.ID,
		&restaurant.RestaurantName,
		&restaurant.BusinessOwnerName,
		&restaurant.Email,
		&restaurant.PhoneNumber,
		&restaurant.BusinessAddress,
		&restaurant.City,
		&restaurant.State,
		&restaurant.Zipcode,
		&restaurant.Latitude,
		&restaurant.Longitude,
		&restaurant.BusinessDescription,
		&restaurant.CoverImage,
		&restaurant.Status,
		&restaurant.IsVerified,
		&restaurant.Rating,
		&restaurant.OpenTime,
		&restaurant.CloseTime,
		&restaurant.IsOpen,
		&restaurant.MinimumOrderAmount,
		&restaurant.CreatedAt,
	)

	if err == sql.ErrNoRows {
		utils.JSON(w, 404, false, "Restaurant not found", nil)
		return
	}
	if err != nil {
		utils.JSON(w, 500, false, "Failed to fetch restaurant details", nil)
		return
	}

	utils.JSON(w, 200, true, "Restaurant details fetched", restaurant)
}

func GetTrakingOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSON(w, 405, false, "Invalid request method", nil)
		return
	}

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
	SELECT 
		o.id,
		o.order_number,
		o.customer_id,
		o.subtotal,
		o.tax_amount,
		o.delivery_fee,
		o.tip_amount,
		o.total_amount,
		o.payment_method,
		o.payment_status,
		o.status,
		o.order_placed_at,
		COALESCE(r.restaurant_name,'') AS restaurant_name,
		COALESCE(a.address,'') AS address,
		COALESCE(a.city,'') AS city
	FROM tbl_orders o
	LEFT JOIN restaurants r ON r.id = o.restaurant_id
	LEFT JOIN customer_delivery_addresses a ON a.id = o.address_id
	WHERE o.status NOT IN ('delivered', 'cancelled')
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

	utils.JSON(w, 200, true, "Orders list", orders)

}

func UpdateCustomerStatus(w http.ResponseWriter, r *http.Request) {
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

	// Request payload
	type Request struct {
		LoginID int    `json:"login_id"`
		Status  string `json:"status"`
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, 400, false, "Invalid JSON payload", nil)
		return
	}

	if req.LoginID == 0 || req.Status == "" {
		utils.JSON(w, 400, false, "login_id and status are required", nil)
		return
	}

	// Validate status
	validStatus := map[string]bool{
		"active":   true,
		"inactive": true,
		"blocked":  true,
		"pending":  true,
	}

	if !validStatus[req.Status] {
		utils.JSON(w, 400, false, "Invalid status value", nil)
		return
	}

	// Update query
	result, err := db.DB.Exec(`
		UPDATE login 
		SET status = ?, updated_at = NOW()
		WHERE id = ?
	`, req.Status, req.LoginID)

	if err != nil {
		utils.JSON(w, 500, false, "Database update failed", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		utils.JSON(w, 404, false, "Customer record not found", nil)
		return
	}

	utils.JSON(w, 200, true, "Customer status updated successfully", nil)
}
