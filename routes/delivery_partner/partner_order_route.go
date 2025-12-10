package DeliveryPartner

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"fmt"
	"net/http"
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
		COALESCE(r.longitude, 0) AS restaurant_longitude

	FROM tbl_orders o
	LEFT JOIN customer_delivery_addresses cda ON cda.id = o.address_id
	LEFT JOIN restaurants r ON r.id = o.restaurant_id
	WHERE o.status IN (%s)
	ORDER BY o.id DESC
`, placeholders)

	args := make([]interface{}, len(statuses))

	for i, s := range statuses {
		args[i] = s
	}

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
		})
	}

	// -------------------------------
	// 5. Final response
	// -------------------------------
	utils.JSON(w, 200, true, "Orders fetched successfully", orders)
}
