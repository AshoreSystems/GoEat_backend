package routes

import (
	"GoEatsapi/db"
	"GoEatsapi/mailer"
	"GoEatsapi/utils"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/refund"
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
	today := utils.GetISTDateString()
	return fmt.Sprintf("#GOEATS-%s-%05d", strings.ReplaceAll(today, "-", ""), nextID), nil
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
		Image      string  `json:"image"`
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

	istTime := utils.GetISTTimeString()

	result, err := tx.Exec(`
INSERT INTO tbl_orders 
(order_number, customer_id, restaurant_id, address_id, subtotal, tax_amount, delivery_fee, total_amount, payment_method, status, order_placed_at) 
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'Online','pending', ?)
`, orderNumber, req.CustomerID, req.RestaurantID, req.AddressID, req.Subtotal, req.TaxAmount, req.DeliveryFee, req.TotalAmount, istTime)

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
        (order_id, menu_item_id, title,qty, base_price, price,image_url, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `, orderID, item.MenuItemID, item.MenuName, item.Qty, item.Price, item.Price, item.Image, istTime)

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

	params := &stripe.PaymentIntentParams{}
	params.AddExpand("latest_charge")

	pi, err := paymentintent.Get(id, params)
	if err != nil {
		http.Error(w, "Stripe error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if pi.LatestCharge == nil || pi.LatestCharge.PaymentMethodDetails == nil ||
		pi.LatestCharge.PaymentMethodDetails.Card == nil {
		http.Error(w, "Card details not available", http.StatusBadRequest)
		return
	}

	brand := pi.LatestCharge.PaymentMethodDetails.Card.Brand
	last4 := pi.LatestCharge.PaymentMethodDetails.Card.Last4

	_, err = tx.Exec(`
INSERT INTO tbl_payment_transactions 
(order_id, customer_id, transaction_reference, payment_mode, payment_gateway, amount, status, brand, last4, payment_intent, created_at) 
VALUES (?, ?, ?, 'Card', 'stripe', ?, ?, ?, ?, ?, ?)
`,
		orderID,
		req.CustomerID,
		"",
		req.TotalAmount,
		"success",
		brand,
		last4,
		req.PaymentintentId,
		istTime,
	)

	if err != nil {
		tx.Rollback()
		sendErrorResponse(w, "Payment transaction save failed")
		return
	}

	tx.Exec("UPDATE tbl_orders SET payment_status='success', status='pending' WHERE id=?", orderID)
	tx.Commit()

	var customerEmail, restaurantName string

	err = db.DB.QueryRow(`
	SELECT c.email, r.restaurant_name
	FROM customer c
	JOIN restaurants r ON r.id = ?
	WHERE c.id = ?`,
		req.RestaurantID,
		req.CustomerID,
	).Scan(&customerEmail, &restaurantName)

	if err != nil {
		fmt.Println("EMAIL FETCH ERROR:", err)
		// do NOT return — order already placed
	}
	subject := "Your GoEats Order Has Been Placed Successfully"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Order Placed Successfully</title>
</head>
<body style="margin:0; padding:0; font-family: Arial, sans-serif; background-color:#f4f4f4;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="padding:20px;">
		<tr>
			<td align="center">

				<table width="600" cellpadding="0" cellspacing="0"
					style="background:#ffffff; border-radius:8px; overflow:hidden;">

					<!-- HEADER -->
					<tr>
						<td style="background-color:#ff6b35; padding:20px; text-align:center;">
							<img src="https://yourdomain.com/assets/logo.png"
								alt="GoEats"
								style="max-height:50px;">
						</td>
					</tr>

					<!-- TITLE -->
					<tr>
						<td style="padding:20px; text-align:center;">
							<h2 style="color:#2c3e50; margin:0;">🎉 Order Placed Successfully</h2>
						</td>
					</tr>

					<!-- CONTENT -->
					<tr>
						<td style="padding:0 20px 20px; color:#555;">
							Dear Customer,<br><br>
							Thank you for ordering with <strong>GoEats</strong>! Your order has been placed successfully.
						</td>
					</tr>

					<!-- ORDER DETAILS -->
					<tr>
						<td style="padding:0 20px;">
							<h3 style="border-bottom:1px solid #eee; padding-bottom:5px; color:#2c3e50;">
								Order Details
							</h3>
							<p style="color:#555;">
								<strong>Order Number:</strong> %s<br>
								<strong>Restaurant:</strong> %s<br>
								<strong>Payment Method:</strong> Card (Stripe)
							</p>
						</td>
					</tr>

					<!-- ORDER SUMMARY -->
					<tr>
						<td style="padding:10px 20px 0;">
							<h3 style="border-bottom:1px solid #eee; padding-bottom:5px; color:#2c3e50;">
								Order Summary
							</h3>
							<table width="100%%" cellpadding="6" cellspacing="0">
								<tr>
									<td>Subtotal</td>
									<td align="right">$%.2f</td>
								</tr>
								<tr>
									<td>Tax</td>
									<td align="right">$%.2f</td>
								</tr>
								<tr>
									<td>Delivery Fee</td>
									<td align="right">$%.2f</td>
								</tr>
								<tr>
									<td style="border-top:1px solid #eee;"><strong>Total Paid</strong></td>
									<td align="right" style="border-top:1px solid #eee; color:#27ae60;">
										<strong>$%.2f</strong>
									</td>
								</tr>
							</table>
						</td>
					</tr>

					<!-- FOOTER TEXT -->
					<tr>
						<td style="padding:15px 20px; color:#555;">
							Your food is now being prepared. You’ll receive updates as your order progresses.
						</td>
					</tr>

					<!-- FOOTER -->
					<tr>
						<td style="background:#f9f9f9; padding:15px; text-align:center; font-size:12px; color:#999;">
							Regards,<br>
							<strong style="color:#ff6b35;">GoEats Team</strong>
						</td>
					</tr>

				</table>

			</td>
		</tr>
	</table>
</body>
</html>
`,
		orderNumber,
		restaurantName,
		req.Subtotal,
		req.TaxAmount,
		req.DeliveryFee,
		req.TotalAmount,
	)

	if customerEmail != "" {
		err = mailer.SendHTMLEmail(customerEmail, subject, htmlBody)
		if err != nil {
			fmt.Println("PLACE ORDER EMAIL ERROR:", err)
		}
	}

	sendSuccessResponse(w, map[string]interface{}{
		"message":        "Order placed successfully",
		"order_id":       orderID,
		"order_number":   orderNumber,
		"payment_status": "success",
	})
}

// func Create_payment_intent(w http.ResponseWriter, r *http.Request) {
// 	r.ParseMultipartForm(10 << 20)

// 	amountStr := r.FormValue("amount")

// 	userAmount, err := strconv.ParseFloat(amountStr, 64)
// 	if err != nil {
// 		http.Error(w, "Invalid amount", http.StatusBadRequest)
// 		return
// 	}

// 	// Stripe needs smallest currency unit (paise)
// 	// Example: 158 → 15800
// 	stripeAmount := int64(userAmount * 100)

// 	// Validate
// 	if stripeAmount <= 0 {
// 		http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
// 		return
// 	}

// 	stripe.Key = os.Getenv("STRIPE_SK")
// 	params := &stripe.PaymentIntentParams{
// 		Amount:             stripe.Int64(stripeAmount),
// 		Currency:           stripe.String("usd"),
// 		PaymentMethodTypes: stripe.StringSlice([]string{"card"}), // Force card only
// 	}

// 	pi, err := paymentintent.New(params)
// 	if err != nil {
// 		http.Error(w, "Stripe error: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	response := map[string]interface{}{
// 		"client_secret":  pi.ClientSecret,
// 		"payment_intent": pi.ID,
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(response)
// }

func Create_payment_intent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// -------- Amount --------
	amountStr := r.FormValue("amount")
	userAmount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		http.Error(w, "Invalid amount", http.StatusBadRequest)
		return
	}

	stripeAmount := int64(userAmount * 100)
	if stripeAmount <= 0 {
		http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
		return
	}

	// -------- Restaurant ID --------
	restaurantID := r.FormValue("restaurant_id")
	if restaurantID == "" {
		http.Error(w, "restaurant_id is required", http.StatusBadRequest)
		return
	}

	// -------- Fetch Restaurant Timing --------
	var openTimeStr, closeTimeStr string

	query := `
		SELECT open_time, close_time
		FROM restaurants
		WHERE id = ?
	`
	err = db.DB.QueryRow(query, restaurantID).Scan(&openTimeStr, &closeTimeStr)
	if err != nil {
		http.Error(w, "Restaurant not found", http.StatusNotFound)
		return
	}

	// -------- Time Validation --------
	//now := utils.GetISTTime()

	isOpen := utils.IsWithinBusinessHours(openTimeStr, closeTimeStr)
	if !isOpen {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Restaurant is currently closed. Order cannot be placed.",
		})
		return
	}

	// -------- Stripe Payment Intent --------
	stripe.Key = os.Getenv("STRIPE_SK")
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(stripeAmount),
		Currency: stripe.String("usd"),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		http.Error(w, "Stripe error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// -------- Response --------
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":        true,
		"client_secret":  pi.ClientSecret,
		"payment_intent": pi.ID,
	})
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

	// // Convert map to slice
	// finalOrders := make([]*Order, 0)
	// for _, v := range orders {
	// 	finalOrders = append(finalOrders, v)
	// }
	// If no orders found
	if len(orders) == 0 {
		json.NewEncoder(w).Encode(APIOrderResponse{
			Status:  true,
			Message: "No orders found",
			Data:    []*Order{}, // same type as finalOrders
		})
		return
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

	// json.NewEncoder(w).Encode(APIOrderResponse{
	// 	Status:  true,
	// 	Message: "Orders fetched successfully",
	// 	Data:    finalOrders,
	// })
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
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Invalid JSON format.",
			ErrorCode: "INVALID_JSON",
		})
		return
	}

	// Validations
	if req.OrderID == 0 || req.CustomerID == 0 || req.CancelReason == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Required fields missing.",
			ErrorCode: "INVALID_INPUT",
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
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Order not found.",
			ErrorCode: "ORDER_NOT_FOUND",
		})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Database error.",
			ErrorCode: "DB_ERROR",
		})
		return
	}

	// Customer authorization
	if dbCustomerID != req.CustomerID {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Unauthorized customer.",
			ErrorCode: "UNAUTHORIZED_CUSTOMER",
		})
		return
	}

	// Allowed statuses
	allowed := map[string]bool{
		"pending":   true,
		"accepted":  true,
		"preparing": true,
	}

	if !allowed[status] {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Order cannot be cancelled at this stage.",
			ErrorCode: "CANCELLATION_NOT_ALLOWED",
		})
		return
	}

	// Fetch payment details
	var paymentIntent, paymentStatus string

	err = db.DB.QueryRow(`
	SELECT payment_intent, status
	FROM tbl_payment_transactions
	WHERE order_id = ?`, req.OrderID).
		Scan(&paymentIntent, &paymentStatus)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Payment record not found.",
			ErrorCode: "PAYMENT_NOT_FOUND",
		})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Payment lookup failed.",
			ErrorCode: "PAYMENT_DB_ERROR",
		})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	// Refund
	if paymentStatus == "success" && paymentIntent != "" {

		refundID, err := RefundMinusFiveDollars(paymentIntent)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(CancleAPIResponse{
				Status:    false,
				Message:   "Refund failed.",
				ErrorCode: "REFUND_FAILED",
			})
			return
		}

		// ✅ Update payment table
		if err := UpdatePaymentRefund(tx, req.OrderID, refundID); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(CancleAPIResponse{
				Status:    false,
				Message:   "Refund update failed.",
				ErrorCode: "REFUND_DB_FAILED",
			})
			return
		}
	}
	_, err = tx.Exec(`
	UPDATE tbl_orders
	SET status = 'cancelled',
	    cancel_reason = ?,
	    updated_at = ?
	WHERE id = ?`,
		req.CancelReason, utils.GetISTTimeString(), req.OrderID,
	)

	if err := tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Transaction commit failed.",
			ErrorCode: "TX_COMMIT_FAILED",
		})
		return
	}

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Failed to cancel order.",
			ErrorCode: "UPDATE_FAILED",
		})
		return
	}

	var email, orderNumber string
	var totalAmount float64

	err = db.DB.QueryRow(`
	SELECT c.email, o.order_number, o.total_amount
	FROM tbl_orders o
	JOIN customer c ON c.id = o.customer_id
	WHERE o.id = ?`, req.OrderID).
		Scan(&email, &orderNumber, &totalAmount)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(CancleAPIResponse{
			Status:    false,
			Message:   "Failed to fetch customer details.",
			ErrorCode: "CUSTOMER_FETCH_FAILED",
		})
		return
	}

	refundAmount := totalAmount
	cancellationFee := 5.0

	if refundAmount > cancellationFee {
		refundAmount = refundAmount - cancellationFee
	} else {
		refundAmount = 0
	}
	subject := "Your Order Has Been Cancelled"

	htmlBody := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Order Cancelled</title>
</head>
<body style="margin:0; padding:0; background-color:#f4f4f4; font-family: Arial, sans-serif;">
	<table width="100%%" cellpadding="0" cellspacing="0" style="padding:20px;">
		<tr>
			<td align="center">

				<table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff; border-radius:8px; overflow:hidden;">
					
					<!-- HEADER -->
					<tr>
						<td style="background-color:#ff6b35; padding:20px; text-align:center;">
							<img src="https://yourdomain.com/assets/logo.png" alt="GoEats"
								style="max-height:50px;">
						</td>
					</tr>

					<!-- TITLE -->
					<tr>
						<td style="padding:20px; text-align:center;">
							<h2 style="color:#e74c3c; margin:0;">❌ Order Cancelled</h2>
						</td>
					</tr>

					<!-- CONTENT -->
					<tr>
						<td style="padding:0 20px 20px; color:#333;">
							Dear Customer,<br><br>

							Your order (<strong>Order Number: %s</strong>) has been successfully cancelled.
						</td>
					</tr>

					<!-- CANCELLATION REASON -->
					<tr>
						<td style="padding:0 20px 20px;">
							<strong>Cancellation Reason:</strong>
							<p style="margin:6px 0; color:#555;">%s</p>
						</td>
					</tr>

					<!-- REFUND -->
					<tr>
						<td style="padding:0 20px 20px;">
							<strong>Refund Details:</strong>

							<table width="100%%" cellpadding="8" cellspacing="0"
								style="margin-top:8px; background:#fafafa; border-radius:4px;">
								<tr>
									<td>Refundable Amount</td>
									<td align="right" style="color:#27ae60; font-weight:bold;">
										$%.2f
									</td>
								</tr>
							</table>

							<p style="font-size:13px; color:#777; margin-top:10px;">
								The refunded amount will be credited to your original payment method within
								<strong>5–7 business days</strong>.
							</p>
						</td>
					</tr>

					<!-- SUPPORT -->
					<tr>
						<td style="padding:0 20px 20px; color:#555;">
							If you have any questions, feel free to contact our support team.
						</td>
					</tr>

					<!-- FOOTER -->
					<tr>
						<td style="background:#f9f9f9; padding:15px; text-align:center; font-size:12px; color:#999;">
							Regards,<br>
							<strong style="color:#ff6b35;">GoEats Team</strong>
						</td>
					</tr>

				</table>

			</td>
		</tr>
	</table>
</body>
</html>
`,
		orderNumber,
		req.CancelReason,
		refundAmount,
	)

	err = mailer.SendHTMLEmail(email, subject, htmlBody)
	if err != nil {
		// Log error only — do NOT fail API after successful cancellation
		fmt.Println("CANCEL ORDER EMAIL ERROR:", err)
	}

	//  Success
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CancleAPIResponse{
		Status:  true,
		Message: "Order cancelled and refund processed successfully.",
		Data: map[string]interface{}{
			"order_id": req.OrderID,
			"status":   "cancelled",
		},
	})
}

func RefundMinusFiveDollars(paymentIntentID string) (string, error) {

	stripe.Key = os.Getenv("STRIPE_SK")

	// Get PaymentIntent
	params := &stripe.PaymentIntentParams{}
	params.AddExpand("latest_charge")

	pi, err := paymentintent.Get(paymentIntentID, params)
	if err != nil {
		return "", err
	}

	if pi.Status != stripe.PaymentIntentStatusSucceeded {
		return "", errors.New("payment not successful")
	}

	refundAmount := pi.AmountReceived - 500 // $5

	if refundAmount <= 0 {
		return "", errors.New("invalid refund amount")
	}

	// ✅ Refund using PaymentIntent (BEST)
	ref, err := refund.New(&stripe.RefundParams{
		PaymentIntent: stripe.String(paymentIntentID),
		Amount:        stripe.Int64(refundAmount),
	})

	if err != nil {
		return "", err
	}

	return ref.ID, nil
}

func UpdatePaymentRefund(tx *sql.Tx, orderID uint64, refundID string) error {
	_, err := tx.Exec(`
		UPDATE tbl_payment_transactions
		SET status = 'refunded',
		    updated_at = ?
		WHERE order_id = ?`, utils.GetISTTimeString(), orderID,
	)
	return err
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		req.UserID, req.RestaurantID, req.OrderID, req.ItemID, req.Rating, req.Review, utils.GetISTTimeString(), utils.GetISTTimeString(),
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
