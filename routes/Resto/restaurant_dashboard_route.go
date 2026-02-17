package resto

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"fmt"
	"net/http"
	"strings"
)

func Get_restaurant_Order_Graph(w http.ResponseWriter, r *http.Request) {
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

	// Get restaurant_id from token
	restaurantID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	query := `
	SELECT 
		DATE_FORMAT(o.created_at, '%Y-%m') AS month,
		COUNT(DISTINCT o.id) AS total_orders,
		COALESCE(SUM(oi.base_price * oi.qty) * 0.8, 0) AS total_amount
	FROM tbl_orders o
	JOIN tbl_order_items oi ON oi.order_id = o.id
	WHERE o.restaurant_id = ?
	  AND o.status = 'delivered'
	  AND o.created_at >= DATE_SUB(CURDATE(), INTERVAL 3 MONTH)
	GROUP BY DATE_FORMAT(o.created_at, '%Y-%m')
	ORDER BY month;
	`

	rows, err := db.DB.Query(query, restaurantID)
	if err != nil {
		fmt.Println(err)
		utils.JSON(w, 500, false, "DB error", nil)
		return
	}
	defer rows.Close()

	type GraphRow struct {
		Month       string  `json:"month"`
		TotalOrders int     `json:"total_orders"`
		TotalAmount float64 `json:"total_amount"`
	}

	// Ensure always last 3 months in IST
	months := []string{}
	for i := 2; i >= 0; i-- {
		months = append(months, utils.GetISTTime().AddDate(0, -i, 0).Format("2006-01"))
	}

	dbData := make(map[string]GraphRow)

	for rows.Next() {
		var g GraphRow
		if err := rows.Scan(&g.Month, &g.TotalOrders, &g.TotalAmount); err != nil {
			utils.JSON(w, 500, false, "Scan error", nil)
			return
		}
		dbData[g.Month] = g
	}

	graph := []GraphRow{}
	totalOrders := 0
	totalAmount := 0.0

	for _, m := range months {
		if val, ok := dbData[m]; ok {
			graph = append(graph, val)
			totalOrders += val.TotalOrders
			totalAmount += val.TotalAmount
		} else {
			graph = append(graph, GraphRow{
				Month:       m,
				TotalOrders: 0,
				TotalAmount: 0,
			})
		}
	}

	response := map[string]interface{}{
		"graph":        graph,
		"total_orders": totalOrders,
		"total_amount": totalAmount,
	}

	utils.JSON(w, 200, true, "Success", response)
}
