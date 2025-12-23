package Admin

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"encoding/json"
	"net/http"
	"strings"
)

func GetCustomerList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid request method",
		})
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
			id, full_name, email, country_code, phone_number,
			dob, profile_image, created_at, updated_at
		FROM customer
		ORDER BY id DESC
	`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Database error",
		})
		return
	}
	defer rows.Close()

	type Customer struct {
		ID           int     `json:"id"`
		FullName     string  `json:"full_name"`
		Email        string  `json:"email"`
		CountryCode  string  `json:"country_code"`
		PhoneNumber  string  `json:"phone_number"`
		DOB          *string `json:"dob"`
		ProfileImage *string `json:"profile_image"`
		CreatedAt    string  `json:"created_at"`
		UpdatedAt    string  `json:"updated_at"`
	}

	var customers []Customer

	for rows.Next() {
		var c Customer
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
		)
		if err != nil {
			continue
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
	   1️⃣ Customer Addresses
	======================= */
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

	var addresses []Address
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

	var orders []Order

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

	utils.JSON(w, 200, true, "Customer details fetched", map[string]interface{}{
		"addresses": addresses,
		"orders":    orders,
	})
}
