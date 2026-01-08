package DeliveryPartner

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"fmt"
	"net/http"
	"strings"
)

func GetOrderGraph(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
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

	// Parse token → get partner_id (loginID)
	partnerID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}

	// SQL Query to get last 3 months data
	query := `
    SELECT 
        DATE_FORMAT(created_at, '%Y-%m') AS month,
        COUNT(*) AS total_orders,
        COALESCE(SUM(delivery_fee), 0) AS delivery_fee
    FROM tbl_orders
    WHERE partner_id = ? and status="delivered"
      AND created_at >= DATE_SUB(CURDATE(), INTERVAL 3 MONTH)
    GROUP BY DATE_FORMAT(created_at, '%Y-%m')
    ORDER BY month;
`

	rows, err := db.DB.Query(query, partnerID)
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

	// Generate last 3 months in IST (ensures always 3 rows)
	months := []string{}
	for i := 2; i >= 0; i-- {
		m := utils.GetISTTime().AddDate(0, -i, 0).Format("2006-01")
		months = append(months, m)
	}

	// Temporary map for existing DB rows
	dbData := make(map[string]GraphRow)

	// Read SQL results
	for rows.Next() {
		var g GraphRow
		err := rows.Scan(&g.Month, &g.TotalOrders, &g.TotalAmount)
		if err != nil {
			fmt.Println(err)
			utils.JSON(w, 500, false, "Scan error", nil)
			return
		}
		dbData[g.Month] = g
	}

	graph := []GraphRow{}
	totalOrders := 0
	totalAmount := 0.0

	// Build final 3-month array
	for _, m := range months {
		if val, exists := dbData[m]; exists {
			graph = append(graph, val)
			totalOrders += val.TotalOrders
			totalAmount += val.TotalAmount
		} else {
			// Add zero entry for missing month
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
