package resto

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"fmt"
	"net/http"
	"strings"
)

func GetRestoOrders(w http.ResponseWriter, r *http.Request) {
	// -------------------------------
	// 1. Read Bearer Token
	// -------------------------------

	status := strings.Trim(r.URL.Query().Get("status"), `"`)

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
	// 2. Fetch ALL pending orders
	// -------------------------------
	orderRows, err := db.DB.Query(`
        SELECT id, order_number, status, order_placed_at, total_amount
        FROM tbl_orders
        WHERE restaurant_id = ? AND status = ?
        ORDER BY id DESC
    `, loginID, status)

	if err != nil {
		fmt.Println("Order Query Error:", err)
		utils.JSON(w, 500, false, "Failed to fetch orders", nil)
		return
	}
	defer orderRows.Close()

	var orders []map[string]interface{}

	for orderRows.Next() {

		var id int
		var ordernumber string
		var status string
		var orderplacedat string
		var totalamount string

		if err := orderRows.Scan(&id, &ordernumber, &status, &orderplacedat, &totalamount); err != nil {
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
		})
	}

	// -------------------------------
	// 5. Final response
	// -------------------------------
	utils.JSON(w, 200, true, "Orders fetched successfully", orders)

}

func UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {

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

	_, err = db.DB.Exec("UPDATE tbl_orders SET status = ? WHERE id = ? AND restaurant_id = ?", status, orderID, loginID)
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
