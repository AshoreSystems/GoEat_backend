package routes

import (
	"GoEatsapi/db"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
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
		Qty        int     `json:"qty"`
		Price      float64 `json:"price"`
	}

	type RequestBody struct {
		CustomerID   int64       `json:"customer_id"`
		RestaurantID int64       `json:"restaurant_id"`
		AddressID    int64       `json:"address_id"`
		Subtotal     float64     `json:"subtotal"`
		TaxAmount    float64     `json:"tax_amount"`
		DeliveryFee  float64     `json:"delivery_fee"`
		TotalAmount  float64     `json:"total_amount"`
		Items        []OrderItem `json:"items"`
		StripeToken  string      `json:"stripe_token"`
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
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'Online', 'pending', NOW())
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
        (order_id, menu_item_id, qty, base_price, price, created_at)
        VALUES (?, ?, ?, ?, ?, NOW())
    `, orderID, item.MenuItemID, item.Qty, item.Price, item.Price)

		if err != nil {
			fmt.Println("Order items insert error:", err)
			tx.Rollback()
			sendErrorResponse(w, "Order items insert failed")
			return
		}
	}

	// Stripe Payment
	//stripe.Key = "sk_test_xxxxxxxxxxxxxxxxxxxxx" // Replace with real key
	stripe.Key = os.Getenv("STRIPE_SK")

	params := &stripe.PaymentIntentParams{
		Amount:        stripe.Int64(int64(req.TotalAmount * 100)),
		Currency:      stripe.String(string(stripe.CurrencyINR)),
		PaymentMethod: stripe.String(req.StripeToken),
		Confirm:       stripe.Bool(true),
	}

	paymentIntent, err := paymentintent.New(params)
	if err != nil {
		tx.Rollback()
		sendErrorResponse(w, "Payment failed")
		return
	}

	_, err = tx.Exec(`
INSERT INTO tbl_payment_transactions 
(order_id, customer_id, transaction_reference, payment_mode, payment_gateway, amount, status, created_at) 
VALUES (?, ?, ?, 'Online', 'stripe', ?, ?, NOW())
`, orderID, req.CustomerID, paymentIntent.ID, req.TotalAmount, paymentIntent.Status)

	if err != nil {
		tx.Rollback()
		sendErrorResponse(w, "Payment transaction save failed")
		return
	}

	tx.Exec("UPDATE tbl_orders SET payment_status='success', status='placed' WHERE id=?", orderID)

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
