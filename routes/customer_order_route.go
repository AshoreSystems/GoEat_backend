package routes

import (
	"GoEatsapi/db"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/paymentintent"
)

type CreatePaymentIntentRequest struct {
	Amount   int64  `json:"amount"`   // in smallest currency unit (e.g. 100 = ₹1)
	Currency string `json:"currency"` // "inr", "usd", etc.
}

func GenerateOrderNumber(db *sql.DB) (string, error) {
	var lastOrderID int64
	err := db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM tbl_orders").Scan(&lastOrderID)
	if err != nil {
		return "", err
	}

	nextID := lastOrderID + 1
	today := time.Now().Format("20060102")
	return fmt.Sprintf("#GOEATS-%s-%05d", today, nextID), nil
}

func sendSuccessResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"status":  true,
		"message": "Success",
		"data":    data,
	}

	json.NewEncoder(w).Encode(response)
}

func PlaceOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	//db := Database() // ensure your Database() returns *sql.DB

	type OrderItem struct {
		MenuItemID int64   `json:"menu_item_id"`
		MenuName   string  `json:"title"`
		Qty        int     `json:"qty"`
		Price      float64 `json:"price"`
	}

	type RequestBody struct {
		CustomerID      int64       `json:"customer_id"`
		RestaurantID    int64       `json:"restaurant_id"`
		AddressID       int64       `json:"address_id"`
		Subtotal        float64     `json:"subtotal"`
		TaxAmount       float64     `json:"tax_amount"`
		DeliveryFee     float64     `json:"delivery_fee"`
		TotalAmount     float64     `json:"total_amount"`
		Items           []OrderItem `json:"items"`
		StripeToken     string      `json:"stripe_token"`
		PaymentintentId string      `json:"payment_intent_id"`
	}

	var req RequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "Invalid input")
		return
	}

	if len(req.Items) == 0 {
		sendErrorResponse(w, "No items found in order")
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		sendErrorResponse(w, "DB transaction start failed")
		return
	}

	orderNumber, err := GenerateOrderNumber(db.DB)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(w, "Order number generation failed")
		return
	}

	result, err := tx.Exec(`
INSERT INTO tbl_orders 
(order_number, customer_id, restaurant_id, address_id, subtotal, tax_amount, delivery_fee, total_amount, payment_method, status, order_placed_at) 
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'Online','pending', NOW())
`, orderNumber, req.CustomerID, req.RestaurantID, req.AddressID, req.Subtotal, req.TaxAmount, req.DeliveryFee, req.TotalAmount)

	if err != nil {
		tx.Rollback()
		fmt.Println("SQL Insert Error:", err)
		sendErrorResponse(w, "Order create failed")
		return
	}

	orderID, _ := result.LastInsertId()

	for _, item := range req.Items {
		_, err := tx.Exec(`
        INSERT INTO tbl_order_items 
        (order_id, menu_item_id, title,qty, base_price, price, created_at)
        VALUES (?, ?, ?, ?, ?, ?, NOW())
    `, orderID, item.MenuItemID, item.MenuName, item.Qty, item.Price, item.Price)

		if err != nil {
			fmt.Println("Order items insert error:", err)
			tx.Rollback()
			sendErrorResponse(w, "Order items insert failed")
			return
		}
	}
	id := req.PaymentintentId
	if id == "" {
		http.Error(w, "payment_intent_id is required", http.StatusBadRequest)
		return
	}
	stripe.Key = os.Getenv("STRIPE_SK")
	pi, err := paymentintent.Get(id, nil)
	if err != nil {
		http.Error(w, "Stripe error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`
INSERT INTO tbl_payment_transactions 
(order_id, customer_id, transaction_reference, payment_mode, payment_gateway, amount, status,brand,last4,payment_intent,created_at) 
VALUES (?, ?, ?, 'Card', 'stripe', ?, ?, ?, ?, ?, NOW())
`, orderID, req.CustomerID, "", req.TotalAmount, "success", pi.Charges.Data[0].PaymentMethodDetails.Card.Brand, pi.Charges.Data[0].PaymentMethodDetails.Card.Last4, req.PaymentintentId)

	if err != nil {
		tx.Rollback()
		sendErrorResponse(w, "Payment transaction save failed")
		return
	}

	tx.Exec("UPDATE tbl_orders SET payment_status='success', status='pending' WHERE id=?", orderID)
	tx.Commit()

	sendSuccessResponse(w, map[string]interface{}{
		"message":        "Order placed successfully",
		"order_id":       orderID,
		"order_number":   orderNumber,
		"payment_status": "success",
	})
}

func Create_payment_intent(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	amountStr := r.FormValue("amount")

	userAmount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	// Stripe needs smallest currency unit (paise)
	// Example: 158 → 15800
	stripeAmount := int64(userAmount * 100)

	// Validate
	if stripeAmount <= 0 {
		http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
		return
	}

	stripe.Key = os.Getenv("STRIPE_SK")
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(stripeAmount),
		Currency:           stripe.String("usd"),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}), // Force card only
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		http.Error(w, "Stripe error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"client_secret":  pi.ClientSecret,
		"payment_intent": pi.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetDefaultAddress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse customer_id from request
	var req struct {
		CustomerID int `json:"customer_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Query database
	query := `
		SELECT 
			id, customer_id, full_name, phone_number, address, city, state, 
			country, postal_code, latitude, longitude, is_default
		FROM customer_delivery_addresses
		WHERE customer_id = ? AND is_default = 1
		LIMIT 1
	`

	var addr CustomerAddress

	err := db.DB.QueryRow(query, req.CustomerID).Scan(
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
	)

	if err == sql.ErrNoRows {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "No default address found",
		})
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send Response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    addr,
	})
}

func UpdateDefaultAddress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse request
	var req struct {
		CustomerID int `json:"customer_id"`
		AddressID  int `json:"address_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Step 1: Reset all addresses for this customer
	_, err = tx.Exec(
		`UPDATE customer_delivery_addresses 
		 SET is_default = 0 
		 WHERE customer_id = ?`,
		req.CustomerID,
	)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Step 2: Set default to selected address
	res, err := tx.Exec(
		`UPDATE customer_delivery_addresses 
		 SET is_default = 1 
		 WHERE id = ? AND customer_id = ?`,
		req.AddressID,
		req.CustomerID,
	)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		tx.Rollback()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Address not found for this customer",
		})
		return
	}

	// Commit
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch updated default address
	var updated CustomerAddress
	query := `
		SELECT id, customer_id, full_name, phone_number, address, city, state, 
		       country, postal_code, latitude, longitude, is_default
		FROM customer_delivery_addresses
		WHERE id = ?`
	err = db.DB.QueryRow(query, req.AddressID).Scan(
		&updated.ID,
		&updated.CustomerID,
		&updated.FullName,
		&updated.PhoneNumber,
		&updated.AddressLine1,
		&updated.City,
		&updated.State,
		&updated.Country,
		&updated.PostalCode,
		&updated.Latitude,
		&updated.Longitude,
		&updated.IsDefault,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Default address updated successfully",
		"data":    updated,
	})
}

type CustomerOrderRequest struct {
	CustomerID uint64 `json:"customer_id"`
}

type OrderItem struct {
	ItemID     uint64  `json:"item_id"`
	MenuItemID uint64  `json:"menu_item_id"`
	Title      string  `json:"title"`
	Qty        int     `json:"qty"`
	BasePrice  float64 `json:"base_price"`
	Price      float64 `json:"price"`
	ImageURL   string  `json:"image_url"`
}

type Order struct {
	OrderID       uint64      `json:"order_id"`
	OrderNumber   string      `json:"order_number"`
	RestaurantID  uint64      `json:"restaurant_id"`
	Subtotal      float64     `json:"subtotal"`
	TaxAmount     float64     `json:"tax_amount"`
	DeliveryFee   float64     `json:"delivery_fee"`
	TipAmount     float64     `json:"tip_amount"`
	TotalAmount   float64     `json:"total_amount"`
	PaymentMethod string      `json:"payment_method"`
	PaymentStatus string      `json:"payment_status"`
	Status        string      `json:"status"`
	OrderPlacedAt string      `json:"order_placed_at"`
	DeliveryTime  *string     `json:"delivery_time"`
	Items         []OrderItem `json:"items"`
}

type APIOrderResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func GetCustomerOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req CustomerOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{Status: false, Message: "Invalid JSON body"})
		return
	}

	if req.CustomerID == 0 {
		json.NewEncoder(w).Encode(APIResponse{Status: false, Message: "customer_id is required"})
		return
	}

	query := `
        SELECT 
            o.id, o.order_number, o.restaurant_id,
            o.subtotal, o.tax_amount, o.delivery_fee,
            o.tip_amount, o.total_amount,
            o.payment_method, o.payment_status, o.status,
            o.order_placed_at, o.delivery_time,

            oi.id, oi.menu_item_id, oi.title, oi.qty,
            oi.base_price, oi.price, oi.image_url

        FROM tbl_orders o
        LEFT JOIN tbl_order_items oi ON o.id = oi.order_id
        WHERE o.customer_id = ?
        ORDER BY oi.id DESC;
    `

	rows, err := db.DB.Query(query, req.CustomerID)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{Status: false, Message: err.Error()})
		return
	}
	defer rows.Close()

	orders := make(map[uint64]*Order)

	for rows.Next() {
		var o Order
		var i OrderItem

		// NULL-safe fields
		var (
			itemID, menuItemID, qty sql.NullInt64
			title, imageURL         sql.NullString
			basePrice, price        sql.NullFloat64
		)

		err := rows.Scan(
			&o.OrderID, &o.OrderNumber, &o.RestaurantID,
			&o.Subtotal, &o.TaxAmount, &o.DeliveryFee,
			&o.TipAmount, &o.TotalAmount,
			&o.PaymentMethod, &o.PaymentStatus, &o.Status,
			&o.OrderPlacedAt, &o.DeliveryTime,

			&itemID, &menuItemID, &title, &qty,
			&basePrice, &price, &imageURL,
		)
		if err != nil {
			json.NewEncoder(w).Encode(APIResponse{Status: false, Message: err.Error()})
			return
		}

		// Create order entry if not already created
		if _, exists := orders[o.OrderID]; !exists {
			orders[o.OrderID] = &Order{
				OrderID:       o.OrderID,
				OrderNumber:   o.OrderNumber,
				RestaurantID:  o.RestaurantID,
				Subtotal:      o.Subtotal,
				TaxAmount:     o.TaxAmount,
				DeliveryFee:   o.DeliveryFee,
				TipAmount:     o.TipAmount,
				TotalAmount:   o.TotalAmount,
				PaymentMethod: o.PaymentMethod,
				PaymentStatus: o.PaymentStatus,
				Status:        o.Status,
				OrderPlacedAt: o.OrderPlacedAt,
				DeliveryTime:  o.DeliveryTime,
				Items:         []OrderItem{},
			}
		}

		// Add items only if available
		if itemID.Valid {
			i = OrderItem{
				ItemID:     uint64(itemID.Int64),
				MenuItemID: uint64(menuItemID.Int64),
				Title:      title.String,
				Qty:        int(qty.Int64),
				BasePrice:  basePrice.Float64,
				Price:      price.Float64,
				ImageURL:   imageURL.String,
			}

			orders[o.OrderID].Items = append(orders[o.OrderID].Items, i)
		}
	}

	// Convert map to slice
	finalOrders := make([]*Order, 0)
	for _, v := range orders {
		finalOrders = append(finalOrders, v)
	}

	json.NewEncoder(w).Encode(APIOrderResponse{
		Status:  true,
		Message: "Orders fetched successfully",
		Data:    finalOrders,
	})
}

func CancelCustomerOrder(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		OrderID      uint64 `json:"order_id"`
		CustomerID   uint64 `json:"customer_id"`
		CancelReason string `json:"cancel_reason"`
	}

	// Parse JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest) // 400
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Invalid JSON format.",
			ErrorCode: "INVALID_JSON",
		})
		return
	}

	// Validations
	if req.OrderID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "order_id is required.",
			ErrorCode: "INVALID_ORDER_ID",
		})
		return
	}

	if req.CustomerID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "customer_id is required.",
			ErrorCode: "INVALID_CUSTOMER_ID",
		})
		return
	}

	if req.CancelReason == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "cancel_reason is required.",
			ErrorCode: "INVALID_REASON",
		})
		return
	}

	// Fetch order
	var status string
	var dbCustomerID uint64

	err := db.DB.QueryRow(`
		SELECT status, customer_id 
		FROM tbl_orders 
		WHERE id = ?`, req.OrderID).
		Scan(&status, &dbCustomerID)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound) // 404
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Order not found.",
			ErrorCode: "ORDER_NOT_FOUND",
		})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError) // 500
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Database error.",
			ErrorCode: "DB_ERROR",
		})
		return
	}

	// Wrong customer
	if dbCustomerID != req.CustomerID {
		w.WriteHeader(http.StatusUnauthorized) // 401
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "You are not authorized to cancel this order.",
			ErrorCode: "UNAUTHORIZED_CUSTOMER",
		})
		return
	}

	// Allowed cancellation statuses
	allowed := map[string]bool{
		"pending":   true,
		"accepted":  true,
		"preparing": true,
	}

	if !allowed[status] {
		w.WriteHeader(http.StatusUnprocessableEntity) // 422
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Order cannot be cancelled at this stage.",
			ErrorCode: "CANCELLATION_NOT_ALLOWED",
		})
		return
	}

	// Cancel the order
	_, updateErr := db.DB.Exec(`
		UPDATE tbl_orders 
		SET status = 'cancelled', cancel_reason = ?, updated_at = NOW()
		WHERE id = ?`,
		req.CancelReason, req.OrderID,
	)

	if updateErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Failed to cancel order.",
			ErrorCode: "UPDATE_FAILED",
		})
		return
	}

	// SUCCESS → ALWAYS 200
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CancleAPIResponse{
		Status:  true,
		Message: "Order cancelled successfully.",
		Data: map[string]interface{}{
			"order_id":      req.OrderID,
			"status":        "cancelled",
			"cancel_reason": req.CancelReason,
		},
	})
}

type CancleAPIResponse struct {
	Status    bool        `json:"status"`               // success or failure
	Message   string      `json:"message"`              // readable message
	ErrorCode string      `json:"error_code,omitempty"` // optional error code
	Data      interface{} `json:"data,omitempty"`       // result payload
}

func CreateRatingReview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		UserID       uint64 `json:"user_id"`
		RestaurantID uint64 `json:"restaurant_id"`
		OrderID      uint64 `json:"order_id"`
		ItemID       string `json:"item_id"`
		Rating       int    `json:"rating"`
		Review       string `json:"review"`
	}

	// Parse JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "Invalid JSON format.",
			ErrorCode: "INVALID_JSON",
		})
		return
	}

	// ==========================
	// BASIC VALIDATIONS
	// ==========================
	if req.UserID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "user_id is required.",
			ErrorCode: "INVALID_USER",
		})
		return
	}

	if req.RestaurantID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "restaurant_id is required.",
			ErrorCode: "INVALID_RESTAURANT",
		})
		return
	}

	if req.OrderID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "order_id is required.",
			ErrorCode: "INVALID_ORDER_ID",
		})
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "rating must be between 1 and 5.",
			ErrorCode: "INVALID_RATING",
		})
		return
	}

	// ==========================
	//  CHECK USER EXISTS
	// ==========================
	var temp uint64
	err := db.DB.QueryRow("SELECT id FROM customer WHERE id = ?", req.UserID).Scan(&temp)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "User not found.",
			ErrorCode: "USER_NOT_FOUND",
		})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "Database error.",
			ErrorCode: "DB_ERROR",
		})
		return
	}

	// ==========================
	// CHECK RESTAURANT EXISTS
	// ==========================
	err = db.DB.QueryRow("SELECT id FROM restaurants WHERE id = ?", req.RestaurantID).Scan(&temp)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "Restaurant not found.",
			ErrorCode: "RESTAURANT_NOT_FOUND",
		})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "Database error.",
			ErrorCode: "DB_ERROR",
		})
		return
	}

	// ==========================
	// CHECK ORDER EXISTS
	// ==========================
	err = db.DB.QueryRow("SELECT id FROM tbl_orders WHERE id = ?", req.OrderID).Scan(&temp)
	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "Order not found.",
			ErrorCode: "ORDER_NOT_FOUND",
		})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "Database error.",
			ErrorCode: "DB_ERROR",
		})
		return
	}

	// ==========================
	// CHECK ITEM EXISTS (if provided)
	// ==========================
	if req.ItemID != "" {

		// Split string into slice
		itemIDs := strings.Split(req.ItemID, ",")

		for _, idStr := range itemIDs {

			idStr = strings.TrimSpace(idStr)
			if idStr == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(RatingAPIResponse{
					Status:    false,
					Message:   "Invalid item_id format.",
					ErrorCode: "INVALID_ITEM_ID",
				})
				return
			}

			// Convert to int
			itemID, convErr := strconv.Atoi(idStr)
			if convErr != nil || itemID <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(RatingAPIResponse{
					Status:    false,
					Message:   "item_id must contain valid numbers.",
					ErrorCode: "INVALID_ITEM_ID",
				})
				return
			}

			// Check each item exists
			var temp int
			dbErr := db.DB.QueryRow("SELECT id FROM menu_items WHERE id = ?", itemID).Scan(&temp)
			if dbErr == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(RatingAPIResponse{
					Status:    false,
					Message:   fmt.Sprintf("Item not found: %d", itemID),
					ErrorCode: "ITEM_NOT_FOUND",
				})
				return
			} else if dbErr != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(RatingAPIResponse{
					Status:    false,
					Message:   "Database error.",
					ErrorCode: "DB_ERROR",
				})
				return
			}
		}
	}

	// ==========================
	//  CHECK DUPLICATE RATING (same user + order)
	// ==========================
	var existing uint64
	err = db.DB.QueryRow(`
		SELECT id 
		FROM tbl_ratings_reviews 
		WHERE user_id = ? AND order_id = ?
	`, req.UserID, req.OrderID).Scan(&existing)

	if err != sql.ErrNoRows && err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "Database error.",
			ErrorCode: "DB_ERROR",
		})
		return
	}

	if existing > 0 {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "Rating already submitted for this order.",
			ErrorCode: "RATING_EXISTS",
		})
		return
	}

	// ==========================
	// INSERT RATING
	// ==========================
	_, err = db.DB.Exec(`
		INSERT INTO tbl_ratings_reviews 
		(user_id, restaurant_id, order_id, item_id, rating, review, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
	`,
		req.UserID, req.RestaurantID, req.OrderID, req.ItemID, req.Rating, req.Review,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(RatingAPIResponse{
			Status:    false,
			Message:   "Failed to submit rating.",
			ErrorCode: "DB_INSERT_ERROR",
		})
		return
	}

	// SUCCESS → ALWAYS 200
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(RatingAPIResponse{
		Status:  true,
		Message: "Rating submitted successfully.",
		Data: map[string]interface{}{
			"user_id":       req.UserID,
			"restaurant_id": req.RestaurantID,
			"order_id":      req.OrderID,
			"item_id":       req.ItemID,
			"rating":        req.Rating,
			"review":        req.Review,
		},
	})
}

type RatingAPIResponse struct {
	Status    bool        `json:"status"`
	Message   string      `json:"message"`
	ErrorCode string      `json:"error_code,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}
