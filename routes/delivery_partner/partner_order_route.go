package DeliveryPartner

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
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
