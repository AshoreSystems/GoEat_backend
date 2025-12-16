package resto

import (
	"GoEatsapi/db"
	"GoEatsapi/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Response struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// Category model
type Category struct {
	ID           int64  `json:"id"`
	CategoryName string `json:"category_name"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func sendErrorResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  false,
		"message": message,
	})
}

func GetRestoCategories(w http.ResponseWriter, r *http.Request) {

	// // Allow only GET request
	// if r.Method != http.MethodGet {
	// 	sendErrorResponse(w, "Invalid request method")
	// 	return
	// }

	query := `
		SELECT id, category_name, description, status, created_at, updated_at
		FROM categories`

	rows, err := db.DB.Query(query)
	if err != nil {
		sendErrorResponse(w, "Failed to fetch categories")
		return
	}
	defer rows.Close()

	var categories []Category

	for rows.Next() {
		var category Category
		err := rows.Scan(
			&category.ID,
			&category.CategoryName,
			&category.Description,
			&category.Status,
			&category.CreatedAt,
			&category.UpdatedAt,
		)
		if err != nil {
			sendErrorResponse(w, "Failed to parse category data")
			return
		}
		categories = append(categories, category)
	}

	response := Response{
		Status:  true,
		Message: "Categories fetched successfully",
		Data:    categories,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type AddMenuItemRequest struct {
	CategoryID      uint64  `json:"category_id"`
	ItemName        string  `json:"item_name"`
	Description     string  `json:"description"`
	Price           float64 `json:"price"`
	ImageURL        string  `json:"image_url"`
	IsVeg           bool    `json:"is_veg"`
	PreparationTime int     `json:"preparation_time"`
}

type APIResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func AddMenuItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Method not allowed",
		})
		return
	}

	var req AddMenuItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println(req.ItemName)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Invalid request payload",
		})
		return
	}

	// Validation
	if req.ItemName == "" || req.CategoryID == 0 || loginID == 0 || req.Price <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Required fields are missing",
		})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback()

	var menuItemID uint64

	// 1️⃣ Check existing menu item
	err = tx.QueryRow(`
		SELECT id FROM menu_items
		WHERE item_name = ? AND category_id = ? AND status = 'active'
		LIMIT 1
	`, req.ItemName, req.CategoryID).Scan(&menuItemID)

	if err != nil && err != sql.ErrNoRows {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Menu item lookup failed",
		})
		return
	}

	// 2️⃣ Create menu item if not exists
	if err == sql.ErrNoRows {
		result, err := tx.Exec(`
			INSERT INTO menu_items
			(category_id, item_name, description, price, image_url, is_veg, preparation_time)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
			req.CategoryID,
			req.ItemName,
			req.Description,
			req.Price,
			req.ImageURL,
			req.IsVeg,
			req.PreparationTime,
		)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Status:  false,
				Message: "Failed to create menu item",
			})
			return
		}

		lastID, _ := result.LastInsertId()
		menuItemID = uint64(lastID)
	}

	// 3️⃣ Check restaurant mapping duplicate
	var exists int
	err = tx.QueryRow(`
		SELECT COUNT(1)
		FROM restaurant_menu_items
		WHERE restaurant_id = ? AND menu_item_id = ?
	`, loginID, menuItemID).Scan(&exists)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Mapping check failed",
		})
		return
	}

	if exists > 0 {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Menu item already exists for this restaurant",
		})
		return
	}

	// 4️⃣ Insert mapping
	_, err = tx.Exec(`
		INSERT INTO restaurant_menu_items
		(restaurant_id, menu_item_id, price, preparation_time)
		VALUES (?, ?, ?, ?)
	`,
		loginID,
		menuItemID,
		req.Price,
		req.PreparationTime,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Failed to map menu item to restaurant",
		})
		return
	}

	if err := tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Transaction commit failed",
		})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(APIResponse{
		Status:  true,
		Message: "Menu item added successfully",
		Data: map[string]interface{}{
			"menu_item_id": menuItemID,
		},
	})
}

type UpdateMenuItemRequest struct {
	MenuItemID      uint64  `json:"menu_item_id"`
	ItemName        string  `json:"item_name"`
	Description     string  `json:"description"`
	Price           float64 `json:"price"`
	ImageURL        string  `json:"image_url"`
	IsVeg           bool    `json:"is_veg"`
	PreparationTime int     `json:"preparation_time"`
}

func UpdateMenuItem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Method not allowed",
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

	tokenString := parts[1]
	loginID, _, err := utils.ParseToken(tokenString)
	if err != nil {
		utils.JSON(w, 401, false, "Invalid or expired token", nil)
		return
	}
	var req UpdateMenuItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Invalid request payload",
		})
		return
	}

	if loginID == 0 || req.MenuItemID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "restaurant_id and menu_item_id are required",
		})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback()

	// 1️⃣ Update menu_items
	menuResult, err := tx.Exec(`
	UPDATE menu_items
	SET item_name = ?, 
	    description = ?, 
	    price = ?, 
	    image_url = ?, 
	    is_veg = ?, 
	    preparation_time = ?
	WHERE id = ?
`,
		req.ItemName,
		req.Description,
		req.Price,
		req.ImageURL,
		req.IsVeg,
		req.PreparationTime,
		req.MenuItemID,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Failed to update menu_items",
		})
		return
	}

	menuRows, _ := menuResult.RowsAffected()
	if menuRows == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Menu item not found",
		})
		return
	}

	// 2️⃣ Update restaurant_menu_items
	restResult, err := tx.Exec(`
	UPDATE restaurant_menu_items
	SET price = ?, 
	    preparation_time = ?
	WHERE restaurant_id = ? AND menu_item_id = ?
`,
		req.Price,
		req.PreparationTime,
		loginID,
		req.MenuItemID,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Failed to update restaurant menu item",
		})
		return
	}

	restRows, _ := restResult.RowsAffected()
	if restRows == 0 {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Restaurant menu item not found",
		})
		return
	}

	if err := tx.Commit(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Status:  false,
			Message: "Transaction commit failed",
		})
		return
	}

	json.NewEncoder(w).Encode(APIResponse{
		Status:  true,
		Message: "Menu item updated successfully",
	})
}

type RestaurantMenuResponse struct {
	MenuItemID      uint64  `json:"menu_item_id"`
	ItemName        string  `json:"item_name"`
	Description     string  `json:"description"`
	Price           float64 `json:"price"`
	ImageURL        string  `json:"image_url"`
	IsVeg           bool    `json:"is_veg"`
	IsAvailable     bool    `json:"is_available"`
	PreparationTime int     `json:"preparation_time"`
}

func GetMenuByRestaurant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 🔐 Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		utils.JSON(w, http.StatusUnauthorized, false, "Authorization header missing", nil)
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		utils.JSON(w, http.StatusUnauthorized, false, "Invalid token format", nil)
		return
	}

	tokenString := parts[1]
	loginID, _, err := utils.ParseToken(tokenString)
	if err != nil || loginID == 0 {
		utils.JSON(w, http.StatusUnauthorized, false, "Invalid or expired token", nil)
		return
	}

	rows, err := db.DB.Query(`
		SELECT 
			mi.id,
			mi.item_name,
			mi.description,
			rmi.price,
			mi.image_url,
			mi.is_veg,
			rmi.is_available,
			rmi.preparation_time
		FROM restaurant_menu_items rmi
		JOIN menu_items mi ON mi.id = rmi.menu_item_id
		WHERE rmi.restaurant_id = ?
		ORDER BY mi.item_name ASC
	`, loginID)

	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, false, "Failed to fetch menu", nil)
		return
	}
	defer rows.Close()

	menu := []RestaurantMenuResponse{}

	for rows.Next() {
		var item RestaurantMenuResponse
		if err := rows.Scan(
			&item.MenuItemID,
			&item.ItemName,
			&item.Description,
			&item.Price,
			&item.ImageURL,
			&item.IsVeg,
			&item.IsAvailable,
			&item.PreparationTime,
		); err != nil {
			utils.JSON(w, http.StatusInternalServerError, false, "Failed to parse menu", nil)
			return
		}
		menu = append(menu, item)
	}

	utils.JSON(w, http.StatusOK, true, "Menu fetched successfully", menu)
}
