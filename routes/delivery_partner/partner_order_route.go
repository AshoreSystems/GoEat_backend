package DeliveryPartner

import (
	"GoEatsapi/db"
	"GoEatsapi/mailer"
	"GoEatsapi/utils"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
	"github.com/stripe/stripe-go/v76/transfer"
)

func GetPartnerOrder(w http.ResponseWriter, r *http.Request) {

	// -------------------------------
	// 1. Read Bearer Token
	// -------------------------------

	rawStatus := r.URL.Query().Get("status")
	if rawStatus == "" {
		utils.JSON(w, 400, false, "Status is required", nil)
		return
	}

	// Convert "preparing,pickup_ready" → []string{"preparing", "pickup_ready"}
	statuses := strings.Split(rawStatus, ",")

	// Generate placeholders same count as statuses → "?,?" etc.
	placeholders := strings.Repeat("?,", len(statuses))
	placeholders = placeholders[:len(placeholders)-1] // remove last comma

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

	// -------------------------------
	// 2. Fetch ALL pending orders
	// -------------------------------

	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	radiusStr := r.URL.Query().Get("radius")

	if latStr == "" || lngStr == "" || radiusStr == "" {
		utils.JSON(w, 400, false, "lat, lng and radius are required", nil)
		return
	}

	partnerLat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		utils.JSON(w, 400, false, "Invalid latitude", nil)
		return
	}

	partnerLng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		utils.JSON(w, 400, false, "Invalid longitude", nil)
		return
	}

	radiusKM, err := strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		utils.JSON(w, 400, false, "Invalid radius", nil)
		return
	}

	query := fmt.Sprintf(`
SELECT 
	o.id,
	o.order_number,
	o.status,
	o.order_placed_at,
	o.total_amount,
	o.address_id,
	o.restaurant_id,

	-- Customer delivery address
	COALESCE(cda.address, '') AS customer_address,
	COALESCE(cda.latitude, 0) AS customer_latitude,
	COALESCE(cda.longitude, 0) AS customer_longitude,

	-- Restaurant address info
	COALESCE(r.restaurant_name, '') AS restaurant_name,
	COALESCE(r.business_address, '') AS restaurant_address,
	COALESCE(r.latitude, 0) AS restaurant_latitude,
	COALESCE(r.longitude, 0) AS restaurant_longitude,

	-- Customer distance
	(
		6371 * ACOS(
			COS(RADIANS(?)) * COS(RADIANS(cda.latitude)) *
			COS(RADIANS(cda.longitude) - RADIANS(?)) +
			SIN(RADIANS(?)) * SIN(RADIANS(cda.latitude))
		)
	) AS customer_distance

FROM tbl_orders o
LEFT JOIN customer_delivery_addresses cda ON cda.id = o.address_id
LEFT JOIN restaurants r ON r.id = o.restaurant_id

WHERE o.status IN (%s)
	AND o.partner_id IS NULL

	-- TODAY'S ORDERS ONLY
	AND o.order_placed_at >= CONVERT_TZ(CURDATE(), 'UTC', '+05:30')
    AND o.order_placed_at < CONVERT_TZ(CURDATE() + INTERVAL 1 DAY, 'UTC', '+05:30')

	-- Customer radius
	AND (
		6371 * ACOS(
			COS(RADIANS(?)) * COS(RADIANS(cda.latitude)) *
			COS(RADIANS(cda.longitude) - RADIANS(?)) +
			SIN(RADIANS(?)) * SIN(RADIANS(cda.latitude))
		)
	) <= ?

	-- Restaurant radius
	AND (
		6371 * ACOS(
			COS(RADIANS(?)) * COS(RADIANS(r.latitude)) *
			COS(RADIANS(r.longitude) - RADIANS(?)) +
			SIN(RADIANS(?)) * SIN(RADIANS(r.latitude))
		)
	) <= ?

ORDER BY o.id DESC
`, placeholders)

	args := []interface{}{
		// SELECT customer distance
		partnerLat, partnerLng, partnerLat,
	}

	// status IN (?, ?, ?)
	for _, s := range statuses {
		args = append(args, s)
	}

	// WHERE customer radius
	args = append(args,
		partnerLat, partnerLng, partnerLat, radiusKM,
	)

	// WHERE restaurant radius
	args = append(args,
		partnerLat, partnerLng, partnerLat, radiusKM,
	)

	orderRows, err := db.DB.Query(query, args...)

	if err != nil {
		fmt.Println("Order Query Error:", err)
		utils.JSON(w, 500, false, "Failed to fetch orders", nil)
		return
	}
	defer orderRows.Close()

	var orders []map[string]interface{}
	var distance float64

	for orderRows.Next() {

		var (
			id                int
			ordernumber       string
			status            string
			orderplacedat     string
			totalamount       string
			addressID         int
			restaurantID      int
			customerAddress   string
			customerLat       float64
			customerLng       float64
			restaurantName    string
			restaurantAddress string
			restaurantLat     float64
			restaurantLng     float64
		)

		if err := orderRows.Scan(
			&id,
			&ordernumber,
			&status,
			&orderplacedat,
			&totalamount,
			&addressID,
			&restaurantID,
			&customerAddress,
			&customerLat,
			&customerLng,
			&restaurantName,
			&restaurantAddress,
			&restaurantLat,
			&restaurantLng,
			&distance,
		); err != nil {
			fmt.Println("Order Scan Error:", err)
			continue
		}

		// -------------------------------
		// 3. Fetch items for each order
		// -------------------------------
		itemRows, err := db.DB.Query(`
            SELECT 
                COALESCE(mi.item_name, ''),
                oi.qty,
                oi.base_price,
                COALESCE(mi.is_veg, 0)
            FROM tbl_order_items oi
            LEFT JOIN menu_items mi ON mi.id = oi.menu_item_id
            WHERE oi.order_id = ?
        `, id)

		if err != nil {
			fmt.Println("Item Query Error:", err)
			continue
		}

		var orderItems []map[string]interface{}

		for itemRows.Next() {

			var itemName string
			var quantity int
			var price float64
			var isVeg int

			if err := itemRows.Scan(&itemName, &quantity, &price, &isVeg); err != nil {
				fmt.Println("Item Scan Error:", err)
				continue
			}

			orderItems = append(orderItems, map[string]interface{}{
				"item_name": itemName,
				"quantity":  quantity,
				"price":     price,
				"is_veg":    isVeg == 1,
			})
		}
		itemRows.Close()

		// -------------------------------
		// 4. Add this order into array
		// -------------------------------
		orders = append(orders, map[string]interface{}{
			"id":            id,
			"ordernumber":   ordernumber,
			"status":        status,
			"orderplacedat": orderplacedat,
			"totalamount":   totalamount,
			"orderitems":    orderItems,

			// Customer delivery address info
			"customer_address":   customerAddress,
			"customer_latitude":  customerLat,
			"customer_longitude": customerLng,

			// Restaurant address info
			"restaurant_name":      restaurantName,
			"restaurant_address":   restaurantAddress,
			"restaurant_latitude":  restaurantLat,
			"restaurant_longitude": restaurantLng,
			"distance_km":          math.Round(distance*100) / 100,
		})
	}

	// -------------------------------
	// 5. Final response
	// -------------------------------
	utils.JSON(w, 200, true, "Orders fetched successfully", orders)
}

func Get_active_Partner_Order(w http.ResponseWriter, r *http.Request) {
	// -------------------------------
	// 1. Read Bearer Token
	// -------------------------------
	rawStatus := r.URL.Query().Get("status")
	if rawStatus == "" {
		utils.JSON(w, 400, false, "Status is required", nil)
		return
	}
	// Convert "preparing,pickup_ready" → []string{"preparing", "pickup_ready"}
	statuses := strings.Split(rawStatus, ",")
	// Generate placeholders same count as statuses → "?,?" etc.
	placeholders := strings.Repeat("?,", len(statuses))
	placeholders = placeholders[:len(placeholders)-1] // remove last comma
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
	// 2. Fetch ALL pending orders
	// -------------------------------
	query := fmt.Sprintf(`
    SELECT
        o.id,
        o.order_number,
        o.status,
        o.order_placed_at,
        o.total_amount,
		o.delivery_fee,
        o.address_id,
        o.restaurant_id,
        -- Customer delivery address
        COALESCE(cda.address, '') AS customer_address,
        COALESCE(cda.latitude, 0) AS customer_latitude,
        COALESCE(cda.longitude, 0) AS customer_longitude,
        -- Restaurant address info
        COALESCE(r.restaurant_name, '') AS restaurant_name,
        COALESCE(r.business_address, '') AS restaurant_address,
        COALESCE(r.latitude, 0) AS restaurant_latitude,
        COALESCE(r.longitude, 0) AS restaurant_longitude
    FROM tbl_orders o
    LEFT JOIN customer_delivery_addresses cda ON cda.id = o.address_id
    LEFT JOIN restaurants r ON r.id = o.restaurant_id
    WHERE o.status IN (%s)
        AND o.partner_id = ?

		-- ✅ TODAY'S ORDERS ONLY
        AND o.order_placed_at >= CONVERT_TZ(CURDATE(), 'UTC', '+05:30')
    	AND o.order_placed_at < CONVERT_TZ(CURDATE() + INTERVAL 1 DAY, 'UTC', '+05:30')

    ORDER BY o.id DESC
`, placeholders)
	args := make([]interface{}, 0, len(statuses)+1)

	// status IN (?, ?, ...)
	for _, s := range statuses {
		args = append(args, s)
	}

	// partner_id = loginID
	args = append(args, loginID)
	orderRows, err := db.DB.Query(query, args...)
	if err != nil {
		fmt.Println("Order Query Error:", err)
		utils.JSON(w, 500, false, "Failed to fetch orders", nil)
		return
	}
	defer orderRows.Close()
	var orders []map[string]interface{}
	for orderRows.Next() {
		var (
			id                int
			ordernumber       string
			status            string
			orderplacedat     string
			totalamount       string
			delivery_fee      string
			addressID         int
			restaurantID      int
			customerAddress   string
			customerLat       float64
			customerLng       float64
			restaurantName    string
			restaurantAddress string
			restaurantLat     float64
			restaurantLng     float64
		)
		if err := orderRows.Scan(
			&id,
			&ordernumber,
			&status,
			&orderplacedat,
			&totalamount,
			&delivery_fee,
			&addressID,
			&restaurantID,
			&customerAddress,
			&customerLat,
			&customerLng,
			&restaurantName,
			&restaurantAddress,
			&restaurantLat,
			&restaurantLng,
		); err != nil {
			fmt.Println("Order Scan Error:", err)
			continue
		}
		// -------------------------------
		// 3. Fetch items for each order
		// -------------------------------
		itemRows, err := db.DB.Query(`
            SELECT
                COALESCE(mi.item_name, ''),
                oi.qty,
                oi.base_price,
                COALESCE(mi.is_veg, 0)
            FROM tbl_order_items oi
            LEFT JOIN menu_items mi ON mi.id = oi.menu_item_id
            WHERE oi.order_id = ?
        `, id)
		if err != nil {
			fmt.Println("Item Query Error:", err)
			continue
		}
		var orderItems []map[string]interface{}
		for itemRows.Next() {
			var itemName string
			var quantity int
			var price float64
			var isVeg int
			if err := itemRows.Scan(&itemName, &quantity, &price, &isVeg); err != nil {
				fmt.Println("Item Scan Error:", err)
				continue
			}
			orderItems = append(orderItems, map[string]interface{}{
				"item_name": itemName,
				"quantity":  quantity,
				"price":     price,
				"is_veg":    isVeg == 1,
			})
		}
		itemRows.Close()
		// -------------------------------
		// 4. Add this order into array
		// -------------------------------
		orders = append(orders, map[string]interface{}{
			"id":            id,
			"ordernumber":   ordernumber,
			"status":        status,
			"orderplacedat": orderplacedat,
			"totalamount":   totalamount,
			"delivery_fee":  delivery_fee,
			"orderitems":    orderItems,
			// Customer delivery address info
			"customer_address":   customerAddress,
			"customer_latitude":  customerLat,
			"customer_longitude": customerLng,
			// Restaurant address info
			"restaurant_name":      restaurantName,
			"restaurant_address":   restaurantAddress,
			"restaurant_latitude":  restaurantLat,
			"restaurant_longitude": restaurantLng,
		})
	}
	// -------------------------------
	// 5. Final response
	// -------------------------------
	utils.JSON(w, 200, true, "Orders fetched successfully", orders)
}

func Generate_Order_Delivery_OTP(w http.ResponseWriter, r *http.Request) {

	subject := "Your OTP Verification Code"

	// -------------------------------
	// 1. Auth (Bearer Token)
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

	partnerID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 2. Parse request
	// -------------------------------
	var req struct {
		OrderID int `json:"order_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, 400, false, "Invalid request body", nil)
		return
	}

	if req.OrderID == 0 {
		utils.JSON(w, 400, false, "Order ID is required", nil)
		return
	}

	// -------------------------------
	// 3. Validate order belongs to partner
	// -------------------------------
	var exists int
	err = db.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM tbl_orders 
		WHERE id = ? AND partner_id = ?
	`, req.OrderID, partnerID).Scan(&exists)

	if err != nil {
		utils.JSON(w, 500, false, "Database error", nil)
		return
	}

	if exists == 0 {
		utils.JSON(w, 403, false, "Order not assigned to this partner", nil)
		return
	}

	// -------------------------------
	// 4. Generate OTP
	// -------------------------------
	otp := utils.GenerateOTP() // 6-digit string

	// hashedOTP, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	// if err != nil {
	// 	utils.JSON(w, 500, false, "Failed to generate OTP", nil)
	// 	return
	// }

	// -------------------------------
	// 5. Store OTP in order
	// -------------------------------
	_, err = db.DB.Exec(`
		UPDATE tbl_orders 
		SET delivery_otp = ?
		WHERE id = ? AND partner_id = ?
	`, otp, req.OrderID, partnerID)

	if err != nil {
		utils.JSON(w, 500, false, "Failed to save OTP", nil)
		return
	}

	// -------------------------------
	// 6. Get customer email using order_id
	// -------------------------------
	var customerEmail string
	var order_number string

	err = db.DB.QueryRow(`
	SELECT c.email,o.order_number
	FROM tbl_orders o
	INNER JOIN customer c ON c.id = o.customer_id
	WHERE o.id = ?
`, req.OrderID).Scan(&customerEmail, &order_number)

	if err != nil {
		utils.ErrorLog.Println("Failed to fetch customer email", err)
		utils.JSON(w, 500, false, "Failed to fetch customer email", nil)
		return
	}

	body := fmt.Sprintf(`
		Hello,

		Your One-Time Password (OTP) is for order: %s

		👉 %s

		Please do not share it with anyone.

		Thanks,
		Team GoEats
	`, order_number, otp)

	err = mailer.SendOTPviaSMTP(customerEmail, subject, body)
	if err != nil {
		utils.ErrorLog.Println("SMTP ERROR:", err)
		utils.JSON(w, 500, false, "Failed to send OTP email", nil)
		return
	}

	utils.JSON(w, 200, true, "Delivery OTP generated successfully", map[string]interface{}{
		"order_id": req.OrderID,
		"otp":      otp, // send only once
	})
}

func Update_Order_Status(w http.ResponseWriter, r *http.Request) {

	// -------------------------------
	// 1. Read Bearer Token
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

	// Parse token -> loginID
	loginID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 2. Read order ID and status from request body
	// -------------------------------

	orderID := r.FormValue("order_id")
	status := r.FormValue("status")

	// -------------------------------
	// 3. Update order status
	// -------------------------------

	_, err = db.DB.Exec("UPDATE tbl_orders SET status = ?, partner_id = ? WHERE id = ?", status, loginID, orderID)
	if err != nil {
		fmt.Println("Order Update Error:", err)
		utils.JSON(w, 500, false, "Failed to update order status", nil)
		return
	}

	// -------------------------------
	// 4. Final response
	// -------------------------------
	utils.JSON(w, 200, true, "Order status updated successfully", nil)
}

func Verify_Order_Delivery_OTP(w http.ResponseWriter, r *http.Request) {

	// -------------------------------
	// 1. Auth (Bearer Token)
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

	partnerID, _, err := utils.ParseToken(parts[1])
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// -------------------------------
	// 2. Parse form-data
	// -------------------------------
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		utils.JSON(w, 400, false, "Invalid form data", nil)
		return
	}

	orderIDStr := r.FormValue("order_id")
	otp := r.FormValue("otp")

	if orderIDStr == "" || otp == "" {
		utils.JSON(w, 400, false, "order_id and otp are required", nil)
		return
	}

	orderID, err := strconv.Atoi(orderIDStr)
	if err != nil {
		utils.JSON(w, 400, false, "Invalid order_id", nil)
		return
	}

	// -------------------------------
	// 3. Fetch order OTP + details
	// -------------------------------
	var savedOTP string
	var orderNumber string
	var customerEmail string

	err = db.DB.QueryRow(`
		SELECT o.delivery_otp, o.order_number, c.email
		FROM tbl_orders o
		INNER JOIN customer c ON c.id = o.customer_id
		WHERE o.id = ? AND o.partner_id = ?
	`, orderID, partnerID).Scan(&savedOTP, &orderNumber, &customerEmail)

	if err != nil {
		utils.ErrorLog.Println("Order not found", err)
		utils.JSON(w, 404, false, "Order not found", nil)
		return
	}

	// -------------------------------
	// 4. Verify OTP (PLAIN)
	// -------------------------------
	if savedOTP != otp {
		utils.ErrorLog.Println("Invalid OTP")
		utils.JSON(w, 400, false, "Invalid OTP", nil)
		return
	}

	// -------------------------------
	// 5. Update order status
	// -------------------------------
	_, err = db.DB.Exec(`
		UPDATE tbl_orders
		SET status = 'delivered'
		WHERE id = ? AND partner_id = ?
	`, orderID, partnerID)

	if err != nil {
		utils.ErrorLog.Println("Failed to update order status", err)
		utils.JSON(w, 500, false, "Failed to update order status", nil)
		return
	}

	//Fetch PaymentIntent ID
	var paymentIntentID string

	err = db.DB.QueryRow(`
		SELECT payment_intent
		FROM tbl_payment_transactions
		WHERE order_id = ?
		LIMIT 1
	`, orderID).Scan(&paymentIntentID)

	if err != nil || paymentIntentID == "" {
		utils.ErrorLog.Println("PaymentIntent not found", err)
		utils.JSON(w, 400, false, "Payment not completed for this order", nil)
		return
	}

	// Get Charge ID from Stripe
	pi, err := paymentintent.Get(paymentIntentID, nil)
	if err != nil {
		utils.ErrorLog.Println("Failed to fetch PaymentIntent", err)
		utils.JSON(w, 500, false, "Payment verification failed", nil)
		return
	}

	if pi.Status != stripe.PaymentIntentStatusSucceeded {
		utils.JSON(w, 400, false, "Payment not completed", nil)
		return
	}

	if pi.LatestCharge == nil {
		utils.JSON(w, 400, false, "No charge found for payment", nil)
		return
	}

	chargeID := pi.LatestCharge.ID

	//  end

	var restaurantStripeID string

	err = db.DB.QueryRow(`
		SELECT r.bank_account_number
		FROM tbl_orders o
		INNER JOIN restaurants r ON r.id = o.restaurant_id
		WHERE o.id = ?
	`, orderID).Scan(&restaurantStripeID)

	if err != nil || restaurantStripeID == "" {
		utils.ErrorLog.Println("Restaurant Stripe account not found", err)
	}

	rows, err := db.DB.Query(`
		SELECT base_price, qty
		FROM tbl_order_items
		WHERE order_id = ?
	`, orderID)
	if err != nil {
		utils.ErrorLog.Println("Failed to fetch order items", err)
	}
	defer rows.Close()

	var restaurantAmount float64

	for rows.Next() {
		var basePrice float64
		var qty int

		rows.Scan(&basePrice, &qty)

		itemTotal := basePrice * float64(qty)
		restaurantAmount += itemTotal * 0.80 // 80% payout
	}

	// Transfer to Restaurant
	if restaurantAmount > 0 && restaurantStripeID != "" {
		_, err = transfer.New(&stripe.TransferParams{
			Amount:            stripe.Int64(int64(restaurantAmount * 100)),
			Currency:          stripe.String(string(stripe.CurrencyUSD)),
			Destination:       stripe.String(restaurantStripeID),
			SourceTransaction: stripe.String(chargeID), // 🔥 IMPORTANT
			Description:       stripe.String("Restaurant payout for order " + orderNumber),
		})

		if err != nil {
			utils.ErrorLog.Println("Restaurant payout failed", err)
		}
	}

	// Fetch Partner Stripe Account + Fee

	var partnerStripeID string
	var deliveryFee float64

	err = db.DB.QueryRow(`
		SELECT pba.stripe_account_id, o.delivery_fee
		FROM tbl_orders o
		INNER JOIN tbl_partner_bank_accounts pba ON pba.partner_id = o.partner_id
		WHERE o.id = ?
	`, orderID).Scan(&partnerStripeID, &deliveryFee)

	if err != nil || partnerStripeID == "" {
		utils.ErrorLog.Println("Partner Stripe account not found", err)
	}
	// end Fetch Partner Stripe Account + Fee

	// Transfer Delivery Fee to Partner
	if deliveryFee > 0 && partnerStripeID != "" {
		_, err = transfer.New(&stripe.TransferParams{
			Amount:            stripe.Int64(int64(deliveryFee * 100)),
			Currency:          stripe.String(string(stripe.CurrencyUSD)),
			Destination:       stripe.String(partnerStripeID),
			SourceTransaction: stripe.String(chargeID), // 🔥 IMPORTANT
			Description:       stripe.String("Delivery payout for order " + orderNumber),
		})

		if err != nil {
			utils.ErrorLog.Println("Partner payout failed", err)
		}
	}

	// end Transfer Delivery Fee to Partner

	// -------------------------------
	// 6. Send delivery confirmation email
	// -------------------------------
	subject := "Your order has been delivered 🎉"

	body := fmt.Sprintf(`
		Hello,

		Great news! 🎉  
		Your order %s has been successfully delivered.

		Thank you for choosing GoEats.
		We hope you enjoyed your meal 😄

		Warm regards,
		Team GoEats
	`, orderNumber)

	err = mailer.SendOTPviaSMTP(customerEmail, subject, body)
	if err != nil {
		utils.ErrorLog.Println("MAIL ERROR", err)
		fmt.Println("MAIL ERROR:", err)
	}

	// -------------------------------
	// 7. Response
	// -------------------------------
	utils.JSON(w, 200, true, "Order delivered successfully", map[string]interface{}{
		"order_number": orderNumber,
	})

	// flow
	// OTP Verified
	// 		↓
	// Order Status = Delivered
	// 		↓
	// Restaurant Payout (80%)
	// 		↓
	// Partner Payout (Delivery Fee)
	// 		↓
	// Email Sent
	// 		↓
	// API Response
}

func Get_Partner_Order_histroy(w http.ResponseWriter, r *http.Request) {
	// -------------------------------
	// 1. Read Bearer Token
	// -------------------------------
	rawStatus := r.URL.Query().Get("status")
	if rawStatus == "" {
		utils.JSON(w, 400, false, "Status is required", nil)
		return
	}
	// Convert "preparing,pickup_ready" → []string{"preparing", "pickup_ready"}
	statuses := strings.Split(rawStatus, ",")
	// Generate placeholders same count as statuses → "?,?" etc.
	placeholders := strings.Repeat("?,", len(statuses))
	placeholders = placeholders[:len(placeholders)-1] // remove last comma
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
	// 2. Fetch ALL pending orders
	// -------------------------------
	query := fmt.Sprintf(`
    SELECT
        o.id,
        o.order_number,
        o.status,
        o.order_placed_at,
        o.total_amount,
		o.delivery_fee,
        o.address_id,
        o.restaurant_id,
        -- Customer delivery address
        COALESCE(cda.address, '') AS customer_address,
        COALESCE(cda.latitude, 0) AS customer_latitude,
        COALESCE(cda.longitude, 0) AS customer_longitude,
        -- Restaurant address info
        COALESCE(r.restaurant_name, '') AS restaurant_name,
        COALESCE(r.business_address, '') AS restaurant_address,
        COALESCE(r.latitude, 0) AS restaurant_latitude,
        COALESCE(r.longitude, 0) AS restaurant_longitude
    FROM tbl_orders o
    LEFT JOIN customer_delivery_addresses cda ON cda.id = o.address_id
    LEFT JOIN restaurants r ON r.id = o.restaurant_id
    WHERE o.status IN (%s)
        AND o.partner_id = ?
    ORDER BY o.id DESC
`, placeholders)
	args := make([]interface{}, 0, len(statuses)+1)

	// status IN (?, ?, ...)
	for _, s := range statuses {
		args = append(args, s)
	}

	// partner_id = loginID
	args = append(args, loginID)
	orderRows, err := db.DB.Query(query, args...)
	if err != nil {

		utils.JSON(w, 500, false, "Failed to fetch orders", nil)
		return
	}
	defer orderRows.Close()
	var orders []map[string]interface{}
	for orderRows.Next() {
		var (
			id                int
			ordernumber       string
			status            string
			orderplacedat     string
			totalamount       string
			delivery_fee      string
			addressID         int
			restaurantID      int
			customerAddress   string
			customerLat       float64
			customerLng       float64
			restaurantName    string
			restaurantAddress string
			restaurantLat     float64
			restaurantLng     float64
		)
		if err := orderRows.Scan(
			&id,
			&ordernumber,
			&status,
			&orderplacedat,
			&totalamount,
			&delivery_fee,
			&addressID,
			&restaurantID,
			&customerAddress,
			&customerLat,
			&customerLng,
			&restaurantName,
			&restaurantAddress,
			&restaurantLat,
			&restaurantLng,
		); err != nil {
			fmt.Println("Order Scan Error:", err)
			continue
		}
		// -------------------------------
		// 3. Fetch items for each order
		// -------------------------------
		itemRows, err := db.DB.Query(`
            SELECT
                COALESCE(mi.item_name, ''),
                oi.qty,
                oi.base_price,
                COALESCE(mi.is_veg, 0)
            FROM tbl_order_items oi
            LEFT JOIN menu_items mi ON mi.id = oi.menu_item_id
            WHERE oi.order_id = ?
        `, id)
		if err != nil {

			continue
		}
		var orderItems []map[string]interface{}
		for itemRows.Next() {
			var itemName string
			var quantity int
			var price float64
			var isVeg int
			if err := itemRows.Scan(&itemName, &quantity, &price, &isVeg); err != nil {
				fmt.Println("Item Scan Error:", err)
				continue
			}
			orderItems = append(orderItems, map[string]interface{}{
				"item_name": itemName,
				"quantity":  quantity,
				"price":     price,
				"is_veg":    isVeg == 1,
			})
		}
		itemRows.Close()
		// -------------------------------
		// 4. Add this order into array
		// -------------------------------
		orders = append(orders, map[string]interface{}{
			"id":            id,
			"ordernumber":   ordernumber,
			"status":        status,
			"orderplacedat": orderplacedat,
			"totalamount":   totalamount,
			"delivery_fee":  delivery_fee,
			"orderitems":    orderItems,
			// Customer delivery address info
			"customer_address":   customerAddress,
			"customer_latitude":  customerLat,
			"customer_longitude": customerLng,
			// Restaurant address info
			"restaurant_name":      restaurantName,
			"restaurant_address":   restaurantAddress,
			"restaurant_latitude":  restaurantLat,
			"restaurant_longitude": restaurantLng,
		})
	}
	// -------------------------------
	// 5. Final response
	// -------------------------------
	utils.JSON(w, 200, true, "Orders fetched successfully", orders)
}
